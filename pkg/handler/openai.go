package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/guregu/null.v4"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/utils"

	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
	logx "github.com/instill-ai/x/log"
	resourcex "github.com/instill-ai/x/resource"
)

// parseOpenAIModelField parses the "model" field from an OpenAI-compatible
// request into namespace, model ID, and optional version.
//
// Accepted formats:
//   - "namespace/model-id"           → latest version
//   - "namespace/model-id:version"   → specific version
func parseOpenAIModelField(model string) (namespace, modelID, version string, err error) {
	if model == "" {
		return "", "", "", fmt.Errorf("model field is required")
	}

	parts := strings.SplitN(model, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("model must be in format namespace/model-id or namespace/model-id:version, got %q", model)
	}

	namespace = parts[0]
	modelAndVersion := parts[1]

	if idx := strings.LastIndex(modelAndVersion, ":"); idx >= 0 {
		modelID = modelAndVersion[:idx]
		version = modelAndVersion[idx+1:]
	} else {
		modelID = modelAndVersion
	}

	return namespace, modelID, version, nil
}

// HandleChatCompletions handles POST /v1/chat/completions using the production
// gRPC path to Ray Serve, translating between OpenAI and Instill formats.
func HandleChatCompletions(s service.Service, _ repository.Repository, w http.ResponseWriter, req *http.Request, _ map[string]string) {

	startTime := time.Now()
	ctx := injectMetadataContext(req)
	logger, _ := logx.GetZapLogger(ctx)

	body, err := io.ReadAll(req.Body)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "failed to read request body", "invalid_request_error", "")
		return
	}
	defer req.Body.Close()

	var chatReq openaiChatRequest
	if err := json.Unmarshal(body, &chatReq); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid JSON body", "invalid_request_error", "")
		return
	}

	nsID, modelID, versionStr, err := parseOpenAIModelField(chatReq.Model)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "invalid_model")
		return
	}

	if err := authenticateUser(ctx, false); err != nil {
		writeOpenAIError(w, http.StatusUnauthorized, "authentication required", "authentication_error", "unauthorized")
		return
	}

	ns, err := s.GetRscNamespace(ctx, nsID)
	if err != nil {
		writeOpenAIError(w, http.StatusNotFound, fmt.Sprintf("namespace %q not found", nsID), "not_found_error", "namespace_not_found")
		return
	}

	pbModel, err := s.GetModelByID(ctx, ns, modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		writeOpenAIError(w, http.StatusNotFound, fmt.Sprintf("model %q not found", chatReq.Model), "not_found_error", "model_not_found")
		return
	}

	modelUID, err := s.GetModelUIDByID(ctx, ns, modelID)
	if err != nil {
		writeOpenAIError(w, http.StatusNotFound, fmt.Sprintf("model %q not found", chatReq.Model), "not_found_error", "model_not_found")
		return
	}

	var version *datamodel.ModelVersion
	if versionStr == "" {
		version, err = s.GetRepository().GetLatestModelVersionByModelUID(ctx, modelUID)
	} else {
		version, err = s.GetModelVersionAdmin(ctx, modelUID, versionStr)
	}
	if err != nil {
		writeOpenAIError(w, http.StatusNotFound, "model version not found", "not_found_error", "version_not_found")
		return
	}

	modelName := fmt.Sprintf("%s/%s", ns.Permalink(), modelID)

	_, _, numReplicas, err := s.GetRayClient().ModelReady(ctx, modelName, version.Version)
	if err != nil || numReplicas == 0 {
		w.Header().Set("Retry-After", "30")
		writeOpenAIError(w, http.StatusServiceUnavailable, "model is scaling up, please retry", "server_error", "model_not_ready")
		return
	}

	logUUID, _ := uuid.NewV4()
	requesterUID, userUID := resourcex.GetRequesterUIDAndUserUID(ctx)
	usageData := &utils.UsageMetricData{
		OwnerUID:     ns.NsUID.String(),
		OwnerType:    mgmtpb.OwnerType_OWNER_TYPE_USER,
		UserUID:      userUID.String(),
		UserType:     mgmtpb.OwnerType_OWNER_TYPE_USER,
		RequesterUID: requesterUID.String(),
		ModelID:      pbModel.Id,
		ModelUID:     modelUID.String(),
		Mode:         mgmtpb.Mode_MODE_SYNC,
		TriggerUID:   logUUID.String(),
		TriggerTime:  startTime.Format(time.RFC3339Nano),
		ModelTask:    commonpb.Task_TASK_CHAT,
	}

	runLog, err := s.CreateModelRun(ctx, logUUID, modelUID, version.Version, body)
	if err != nil {
		logger.Warn("failed to create model run log", zap.Error(err))
	}

	defer func() {
		usageData.ComputeTimeDuration = time.Since(startTime).Seconds()
		if writeErr := s.WriteNewDataPoint(ctx, usageData); writeErr != nil {
			logger.Warn("usage/metric write failed", zap.Error(writeErr))
		}
	}()

	taskInput, err := openaiToInstillTaskInput(chatReq, pbModel.Id)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		writeOpenAIError(w, http.StatusBadRequest, "failed to build task input: "+err.Error(), "invalid_request_error", "")
		return
	}

	triggerReq := &modelpb.TriggerModelVersionRequest{
		Name:       fmt.Sprintf("namespaces/%s/models/%s/versions/%s", nsID, modelID, version.Version),
		TaskInputs: []*structpb.Struct{taskInput},
	}

	inferResp, err := s.GetRayClient().ModelInferRequest(ctx, commonpb.Task_TASK_CHAT, triggerReq, modelName, version.Version)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		if runLog != nil {
			_ = s.UpdateModelRunWithError(ctx, runLog, err)
		}
		if strings.Contains(err.Error(), "allocate memory") || strings.Contains(err.Error(), "out of memory") {
			writeOpenAIError(w, http.StatusServiceUnavailable, "model out of memory, try with smaller input", "server_error", "resource_exhausted")
		} else {
			writeOpenAIError(w, http.StatusBadGateway, "model inference failed: "+err.Error(), "server_error", "upstream_error")
		}
		return
	}

	outputs := inferResp.GetTaskOutputs()
	if len(outputs) == 0 {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		writeOpenAIError(w, http.StatusBadGateway, "model returned empty response", "server_error", "empty_response")
		return
	}

	chatResp, err := instillOutputToOpenAIResponse(outputs[0], chatReq.Model, logUUID.String())
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		writeOpenAIError(w, http.StatusBadGateway, "failed to parse model response: "+err.Error(), "server_error", "")
		return
	}

	usageData.Status = mgmtpb.Status_STATUS_COMPLETED

	if chatReq.Stream {
		simulateOpenAIStream(w, chatResp)
	} else {
		writeOpenAIJSON(w, http.StatusOK, chatResp)
	}

	if runLog != nil {
		updateRunCompleted(ctx, s, runLog)
	}
}

// openaiToInstillTaskInput converts an OpenAI chat request into the Instill
// TASK_CHAT task_input protobuf structure expected by Ray Serve gRPC.
func openaiToInstillTaskInput(chatReq openaiChatRequest, modelID string) (*structpb.Struct, error) {
	instillMessages := make([]any, 0, len(chatReq.Messages))
	for _, msg := range chatReq.Messages {
		contentParts, err := convertOpenAIContent(msg.Content)
		if err != nil {
			return nil, fmt.Errorf("message content: %w", err)
		}
		instillMsg := map[string]any{
			"role":    msg.Role,
			"content": contentParts,
		}
		if msg.Name != "" {
			instillMsg["name"] = msg.Name
		}
		instillMessages = append(instillMessages, instillMsg)
	}

	params := map[string]any{
		"stream": false,
	}
	if chatReq.MaxTokens != nil {
		params["max-tokens"] = *chatReq.MaxTokens
	}
	if chatReq.Temperature != nil {
		params["temperature"] = *chatReq.Temperature
	}
	if chatReq.TopP != nil {
		params["top-p"] = *chatReq.TopP
	}
	if chatReq.N != nil {
		params["n"] = *chatReq.N
	}
	if chatReq.Seed != nil {
		params["seed"] = *chatReq.Seed
	}

	return structpb.NewStruct(map[string]any{
		"data": map[string]any{
			"model":    modelID,
			"messages": instillMessages,
		},
		"parameter": params,
	})
}

// convertOpenAIContent converts OpenAI message content (string or array) into
// the Instill content parts format: [{"type":"text","text":"..."}].
func convertOpenAIContent(raw json.RawMessage) ([]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []any{map[string]any{"type": "text", "text": ""}}, nil
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []any{map[string]any{"type": "text", "text": text}}, nil
	}

	var parts []map[string]any
	if err := json.Unmarshal(raw, &parts); err != nil {
		return nil, fmt.Errorf("content must be a string or array of content parts")
	}

	result := make([]any, 0, len(parts))
	for _, p := range parts {
		switch p["type"] {
		case "text":
			result = append(result, map[string]any{"type": "text", "text": p["text"]})
		case "image_url":
			if urlObj, ok := p["image_url"].(map[string]any); ok {
				result = append(result, map[string]any{"type": "image-url", "image-url": urlObj["url"]})
			}
		default:
			result = append(result, p)
		}
	}
	return result, nil
}

// instillOutputToOpenAIResponse converts an Instill TASK_CHAT task_output
// into an OpenAI ChatCompletion response.
func instillOutputToOpenAIResponse(output *structpb.Struct, model, chatID string) (*openaiChatResponse, error) {
	dataField := output.Fields["data"]
	if dataField == nil || dataField.GetStructValue() == nil {
		return nil, fmt.Errorf("missing data in task output")
	}
	data := dataField.GetStructValue()

	choicesList := data.Fields["choices"]
	if choicesList == nil || choicesList.GetListValue() == nil {
		return nil, fmt.Errorf("missing choices in task output")
	}

	var created int64
	choices := make([]openaiChatChoice, 0)
	for _, cv := range choicesList.GetListValue().Values {
		c := cv.GetStructValue()
		if c == nil {
			continue
		}

		finishReason := "stop"
		if fr, ok := c.Fields["finish-reason"]; ok && fr.GetStringValue() == "length" {
			finishReason = "length"
		}

		var msgContent, msgRole string
		if msgField := c.Fields["message"]; msgField != nil && msgField.GetStructValue() != nil {
			msg := msgField.GetStructValue()
			msgContent = msg.Fields["content"].GetStringValue()
			msgRole = msg.Fields["role"].GetStringValue()
		}
		if msgRole == "" {
			msgRole = "assistant"
		}

		if createdVal, ok := c.Fields["created"]; ok && createdVal.GetNumberValue() > 0 {
			created = int64(createdVal.GetNumberValue())
		}

		choices = append(choices, openaiChatChoice{
			Index:        int(c.Fields["index"].GetNumberValue()),
			Message:      openaiChatMsg{Role: msgRole, Content: msgContent},
			FinishReason: finishReason,
		})
	}

	if created == 0 {
		created = time.Now().Unix()
	}

	resp := &openaiChatResponse{
		ID:      "chatcmpl-" + chatID,
		Object:  "chat.completion",
		Created: created,
		Model:   model,
		Choices: choices,
	}

	if metaField := output.Fields["metadata"]; metaField != nil && metaField.GetStructValue() != nil {
		if usageField := metaField.GetStructValue().Fields["usage"]; usageField != nil && usageField.GetStructValue() != nil {
			u := usageField.GetStructValue()
			prompt := int(u.Fields["prompt-tokens"].GetNumberValue())
			completion := int(u.Fields["completion-tokens"].GetNumberValue())
			total := int(u.Fields["total-tokens"].GetNumberValue())
			if total == 0 {
				total = prompt + completion
			}
			resp.Usage = &openaiChatUsage{
				PromptTokens:     prompt,
				CompletionTokens: completion,
				TotalTokens:      total,
			}
		}
	}

	return resp, nil
}

// simulateOpenAIStream emits SSE chunks from a complete gRPC response.
// gRPC is unary so we receive the full response then stream it to the client.
func simulateOpenAIStream(w http.ResponseWriter, resp *openaiChatResponse) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeOpenAIError(w, http.StatusInternalServerError, "streaming not supported", "server_error", "")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	base := openaiStreamChunk{
		ID:      resp.ID,
		Object:  "chat.completion.chunk",
		Created: resp.Created,
		Model:   resp.Model,
	}

	// First chunk: role
	base.Choices = []openaiStreamChoice{{
		Index: 0,
		Delta: openaiDelta{Role: "assistant"},
	}}
	writeOpenAIStreamChunk(w, flusher, base)

	// Content chunk(s)
	if len(resp.Choices) > 0 {
		content := resp.Choices[0].Message.Content
		base.Choices = []openaiStreamChoice{{
			Index: 0,
			Delta: openaiDelta{Content: content},
		}}
		writeOpenAIStreamChunk(w, flusher, base)
	}

	// Finish chunk with usage
	finishReason := "stop"
	if len(resp.Choices) > 0 {
		finishReason = resp.Choices[0].FinishReason
	}
	base.Choices = []openaiStreamChoice{{
		Index:        0,
		Delta:        openaiDelta{},
		FinishReason: &finishReason,
	}}
	base.Usage = resp.Usage
	writeOpenAIStreamChunk(w, flusher, base)

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func writeOpenAIStreamChunk(w http.ResponseWriter, flusher http.Flusher, chunk openaiStreamChunk) {
	data, _ := json.Marshal(chunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func updateRunCompleted(ctx context.Context, s service.Service, runLog *datamodel.ModelRun) {
	now := time.Now()
	runLog.EndTime = null.TimeFrom(now)
	runLog.TotalDuration = null.IntFrom(now.Sub(runLog.CreateTime).Milliseconds())
	runLog.Status = datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_COMPLETED)
	_ = s.GetRepository().UpdateModelRun(ctx, runLog)
}

// HandleListModels handles GET /v1/models, returning deployed chat models in
// OpenAI format for model discovery by coding tools.
func HandleListModels(s service.Service, _ repository.Repository, w http.ResponseWriter, req *http.Request, _ map[string]string) {
	ctx := injectMetadataContext(req)

	if err := authenticateUser(ctx, false); err != nil {
		writeOpenAIError(w, http.StatusUnauthorized, "authentication required", "authentication_error", "unauthorized")
		return
	}

	models, _, _, err := s.ListPublicModels(ctx, 100, "", modelpb.View_VIEW_BASIC, nil, filtering.Filter{}, false, ordering.OrderBy{})
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "failed to list models", "server_error", "")
		return
	}

	data := make([]openaiModel, 0, len(models))
	for _, m := range models {
		if m.Task != commonpb.Task_TASK_CHAT {
			continue
		}
		nsID := ""
		if parts := strings.Split(m.Name, "/"); len(parts) >= 2 {
			nsID = parts[1]
		}
		modelID := m.Id
		data = append(data, openaiModel{
			ID:      fmt.Sprintf("%s/%s", nsID, modelID),
			Object:  "model",
			Created: m.CreateTime.AsTime().Unix(),
			OwnedBy: nsID,
		})
	}

	writeOpenAIJSON(w, http.StatusOK, openaiModelList{
		Object: "list",
		Data:   data,
	})
}

func injectMetadataContext(req *http.Request) context.Context {
	headers := map[string]string{}
	for key, value := range req.Header {
		if len(value) > 0 {
			headers[key] = value[0]
		}
	}
	md := metadata.New(headers)
	return metadata.NewIncomingContext(req.Context(), md)
}
