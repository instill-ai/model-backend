package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/utils"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
	logx "github.com/instill-ai/x/log"
	resourcex "github.com/instill-ai/x/resource"
)

// HandleMessages handles POST /v1/messages (Anthropic Messages API) using the
// production gRPC path to Ray Serve, translating between Anthropic and Instill
// formats.
func HandleMessages(s service.Service, _ repository.Repository, w http.ResponseWriter, req *http.Request, _ map[string]string) {

	startTime := time.Now()
	ctx := injectMetadataContext(req)
	logger, _ := logx.GetZapLogger(ctx)

	body, err := io.ReadAll(req.Body)
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "failed to read request body", "invalid_request_error")
		return
	}
	defer req.Body.Close()

	var antReq anthropicRequest
	if err := json.Unmarshal(body, &antReq); err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid JSON body", "invalid_request_error")
		return
	}

	nsID, modelID, versionStr, err := parseOpenAIModelField(antReq.Model)
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, err.Error(), "invalid_request_error")
		return
	}

	if err := authenticateUser(ctx, false); err != nil {
		writeAnthropicError(w, http.StatusUnauthorized, "authentication required", "authentication_error")
		return
	}

	ns, err := s.GetRscNamespace(ctx, nsID)
	if err != nil {
		writeAnthropicError(w, http.StatusNotFound, fmt.Sprintf("namespace %q not found", nsID), "not_found_error")
		return
	}

	pbModel, err := s.GetModelByID(ctx, ns, modelID, modelpb.View_VIEW_FULL)
	if err != nil {
		writeAnthropicError(w, http.StatusNotFound, fmt.Sprintf("model %q not found", antReq.Model), "not_found_error")
		return
	}

	modelUID, err := s.GetModelUIDByID(ctx, ns, modelID)
	if err != nil {
		writeAnthropicError(w, http.StatusNotFound, fmt.Sprintf("model %q not found", antReq.Model), "not_found_error")
		return
	}

	var version *datamodel.ModelVersion
	if versionStr == "" {
		version, err = s.GetRepository().GetLatestModelVersionByModelUID(ctx, modelUID)
	} else {
		version, err = s.GetModelVersionAdmin(ctx, modelUID, versionStr)
	}
	if err != nil {
		writeAnthropicError(w, http.StatusNotFound, "model version not found", "not_found_error")
		return
	}

	modelName := fmt.Sprintf("%s/%s", ns.Permalink(), modelID)

	_, _, numReplicas, err := s.GetRayClient().ModelReady(ctx, modelName, version.Version)
	if err != nil || numReplicas == 0 {
		w.Header().Set("Retry-After", "30")
		writeAnthropicError(w, http.StatusServiceUnavailable, "model is scaling up, please retry", "overloaded_error")
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

	// Direct streaming: bypass gRPC unary path and call the inference server
	// HTTP endpoint directly, translating OpenAI SSE chunks to Anthropic SSE
	// on-the-fly.
	if antReq.Stream {
		inferURL, urlErr := s.GetRayClient().GetInferenceServerURL(ctx, modelName, version.Version)
		if urlErr == nil {
			inferReq := anthropicToInferenceRequest(antReq)
			streamResp, streamErr := doInferenceStream(ctx, inferURL, inferReq)
			if streamErr == nil {
				usage := forwardAsAnthropicStream(w, streamResp, "msg_"+logUUID.String(), antReq.Model)
				usageData.Status = mgmtpb.Status_STATUS_COMPLETED
				_ = usage
				if runLog != nil {
					updateRunCompleted(ctx, s, runLog)
				}
				return
			}
			logger.Warn("direct streaming failed, falling back to gRPC", zap.Error(streamErr))
		} else {
			logger.Warn("could not resolve inference server URL, falling back to gRPC", zap.Error(urlErr))
		}
	}

	// gRPC unary path: used for non-streaming requests or as a fallback when
	// direct streaming is unavailable.
	taskInput, err := anthropicToInstillTaskInput(antReq, pbModel.Id)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		writeAnthropicError(w, http.StatusBadRequest, "failed to build task input: "+err.Error(), "invalid_request_error")
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
			writeAnthropicError(w, http.StatusServiceUnavailable, "model out of memory", "overloaded_error")
		} else {
			writeAnthropicError(w, http.StatusBadGateway, "model inference failed: "+err.Error(), "api_error")
		}
		return
	}

	outputs := inferResp.GetTaskOutputs()
	if len(outputs) == 0 {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		writeAnthropicError(w, http.StatusBadGateway, "model returned empty response", "api_error")
		return
	}

	antResp, err := instillOutputToAnthropicResponse(outputs[0], antReq.Model, logUUID.String())
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		writeAnthropicError(w, http.StatusBadGateway, "failed to parse model response: "+err.Error(), "api_error")
		return
	}

	usageData.Status = mgmtpb.Status_STATUS_COMPLETED

	if antReq.Stream {
		simulateAnthropicStream(w, antResp)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(antResp)
	}

	if runLog != nil {
		updateRunCompleted(ctx, s, runLog)
	}
}

// anthropicToInstillTaskInput converts an Anthropic Messages request into the
// Instill TASK_CHAT task_input protobuf structure expected by Ray Serve gRPC.
func anthropicToInstillTaskInput(antReq anthropicRequest, modelID string) (*structpb.Struct, error) {
	instillMessages := make([]any, 0, len(antReq.Messages)+1)

	if sysText := antReq.SystemText(); sysText != "" {
		instillMessages = append(instillMessages, map[string]any{
			"role":    "system",
			"content": []any{map[string]any{"type": "text", "text": sysText}},
		})
	}

	for _, msg := range antReq.Messages {
		contentParts, err := convertAnthropicContent(msg.Content)
		if err != nil {
			return nil, fmt.Errorf("message content: %w", err)
		}
		instillMessages = append(instillMessages, map[string]any{
			"role":    msg.Role,
			"content": contentParts,
		})
	}

	params := map[string]any{
		"max-tokens": antReq.MaxTokens,
		"stream":     false,
	}
	if antReq.Temperature != nil {
		params["temperature"] = *antReq.Temperature
	}
	if antReq.TopP != nil {
		params["top-p"] = *antReq.TopP
	}

	return structpb.NewStruct(map[string]any{
		"data": map[string]any{
			"model":    modelID,
			"messages": instillMessages,
		},
		"parameter": params,
	})
}

// convertAnthropicContent converts Anthropic message content (string or array
// of content blocks) into a single Instill text content part. Multiple text
// blocks are merged because the Instill task input expects at most one text
// element per message.
func convertAnthropicContent(raw json.RawMessage) ([]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []any{map[string]any{"type": "text", "text": ""}}, nil
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []any{map[string]any{"type": "text", "text": text}}, nil
	}

	var blocks []map[string]any
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("content must be a string or array of content blocks")
	}

	var texts []string
	for _, b := range blocks {
		if b["type"] == "text" {
			if t, ok := b["text"].(string); ok && t != "" {
				texts = append(texts, t)
			}
		}
	}

	merged := strings.Join(texts, "\n")
	if merged == "" {
		merged = ""
	}
	return []any{map[string]any{"type": "text", "text": merged}}, nil
}

// instillOutputToAnthropicResponse converts an Instill TASK_CHAT task_output
// into an Anthropic Messages response.
func instillOutputToAnthropicResponse(output *structpb.Struct, model, msgID string) (*anthropicResponse, error) {
	dataField := output.Fields["data"]
	if dataField == nil || dataField.GetStructValue() == nil {
		return nil, fmt.Errorf("missing data in task output")
	}
	data := dataField.GetStructValue()

	choicesList := data.Fields["choices"]
	if choicesList == nil || choicesList.GetListValue() == nil {
		return nil, fmt.Errorf("missing choices in task output")
	}

	content := ""
	stopReason := "end_turn"

	values := choicesList.GetListValue().Values
	if len(values) > 0 {
		c := values[0].GetStructValue()
		if c != nil {
			if msgField := c.Fields["message"]; msgField != nil && msgField.GetStructValue() != nil {
				content = msgField.GetStructValue().Fields["content"].GetStringValue()
			}
			if fr, ok := c.Fields["finish-reason"]; ok && fr.GetStringValue() == "length" {
				stopReason = "max_tokens"
			}
		}
	}

	resp := &anthropicResponse{
		ID:         "msg_" + msgID,
		Type:       "message",
		Role:       "assistant",
		Content:    []anthropicContent{{Type: "text", Text: content}},
		Model:      model,
		StopReason: &stopReason,
	}

	if metaField := output.Fields["metadata"]; metaField != nil && metaField.GetStructValue() != nil {
		if usageField := metaField.GetStructValue().Fields["usage"]; usageField != nil && usageField.GetStructValue() != nil {
			u := usageField.GetStructValue()
			resp.Usage = anthropicUsage{
				InputTokens:  int(u.Fields["prompt-tokens"].GetNumberValue()),
				OutputTokens: int(u.Fields["completion-tokens"].GetNumberValue()),
			}
		}
	}

	return resp, nil
}

// simulateAnthropicStream emits Anthropic SSE events from a complete gRPC
// response. gRPC is unary so we receive the full response then stream it.
func simulateAnthropicStream(w http.ResponseWriter, resp *anthropicResponse) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAnthropicError(w, http.StatusInternalServerError, "streaming not supported", "api_error")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	writeSSE(w, flusher, "message_start", anthropicMessageStart(resp.ID, resp.Model))
	writeSSE(w, flusher, "content_block_start", anthropicContentBlockStart(0))

	content := ""
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}
	if content != "" {
		writeSSE(w, flusher, "content_block_delta", anthropicContentBlockDelta(0, content))
	}

	writeSSE(w, flusher, "content_block_stop", anthropicContentBlockStop(0))

	stopReason := "end_turn"
	if resp.StopReason != nil {
		stopReason = *resp.StopReason
	}
	writeSSE(w, flusher, "message_delta", anthropicMessageDelta(stopReason, resp.Usage.OutputTokens))
	writeSSE(w, flusher, "message_stop", anthropicMessageStop())
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, eventType string, data any) {
	jsonBytes, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, jsonBytes)
	flusher.Flush()
}

func writeAnthropicError(w http.ResponseWriter, status int, message, errType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    errType,
			"message": message,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}
