package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/ray"
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

// HandleMessages handles POST /v1/messages (Anthropic Messages API).
// It translates the request to OpenAI format, proxies to Ray Serve,
// and translates the response back to Anthropic format.
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

	appName, err := ray.GetApplicationMetadataValue(modelName, version.Version)
	if err != nil {
		writeAnthropicError(w, http.StatusInternalServerError, "internal error resolving model", "api_error")
		return
	}

	// Run logging
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

	// Translate Anthropic -> OpenAI
	openAIBody := translateAnthropicToOpenAI(antReq)
	openAIBytes, _ := json.Marshal(openAIBody)

	targetURL := fmt.Sprintf("http://%s:%d/%s/v1/chat/completions",
		config.Config.Ray.Host,
		config.Config.Ray.Port.SERVE,
		appName,
	)

	proxyReq, err := http.NewRequestWithContext(req.Context(), http.MethodPost, targetURL, bytes.NewReader(openAIBytes))
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		writeAnthropicError(w, http.StatusInternalServerError, "failed to create proxy request", "api_error")
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 600 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		if runLog != nil {
			_ = s.UpdateModelRunWithError(ctx, runLog, err)
		}
		writeAnthropicError(w, http.StatusBadGateway, "failed to reach model backend", "api_error")
		return
	}
	defer resp.Body.Close()

	if antReq.Stream {
		streamAnthropicResponse(w, resp, antReq.Model, s, ctx, runLog, usageData, logger)
	} else {
		proxyAnthropicNonStream(w, resp, antReq.Model, s, ctx, runLog, usageData)
	}
}

func translateAnthropicToOpenAI(r anthropicRequest) map[string]any {
	messages := make([]map[string]any, 0, len(r.Messages)+1)

	if r.System != "" {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": r.System,
		})
	}

	for _, m := range r.Messages {
		msg := map[string]any{"role": m.Role}
		var text string
		if err := json.Unmarshal(m.Content, &text); err == nil {
			msg["content"] = text
		} else {
			msg["content"] = m.Content
		}
		messages = append(messages, msg)
	}

	out := map[string]any{
		"model":      r.Model,
		"messages":   messages,
		"max_tokens": r.MaxTokens,
		"stream":     r.Stream,
	}
	if r.Temperature != nil {
		out["temperature"] = *r.Temperature
	}
	if r.TopP != nil {
		out["top_p"] = *r.TopP
	}
	if len(r.StopSeqs) > 0 {
		out["stop"] = r.StopSeqs
	}

	return out
}

func proxyAnthropicNonStream(w http.ResponseWriter, resp *http.Response, model string, s service.Service, ctx context.Context, runLog *datamodel.ModelRun, usageData *utils.UsageMetricData) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		writeAnthropicError(w, http.StatusBadGateway, "failed to read upstream response", "api_error")
		return
	}

	var openAIResp struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		writeAnthropicError(w, http.StatusBadGateway, "invalid upstream response", "api_error")
		return
	}

	content := ""
	stopReason := "end_turn"
	if len(openAIResp.Choices) > 0 {
		content = openAIResp.Choices[0].Message.Content
		if openAIResp.Choices[0].FinishReason == "length" {
			stopReason = "max_tokens"
		}
	}

	antResp := anthropicResponse{
		ID:         openAIResp.ID,
		Type:       "message",
		Role:       "assistant",
		Content:    []anthropicContent{{Type: "text", Text: content}},
		Model:      model,
		StopReason: &stopReason,
		Usage: anthropicUsage{
			InputTokens:  openAIResp.Usage.PromptTokens,
			OutputTokens: openAIResp.Usage.CompletionTokens,
		},
	}

	usageData.Status = mgmtpb.Status_STATUS_COMPLETED
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(antResp)

	if runLog != nil {
		updateRunCompleted(ctx, s, runLog)
	}
}

func streamAnthropicResponse(w http.ResponseWriter, resp *http.Response, model string, s service.Service, ctx context.Context, runLog *datamodel.ModelRun, usageData *utils.UsageMetricData, logger *zap.Logger) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAnthropicError(w, http.StatusInternalServerError, "streaming not supported", "api_error")
		return
	}

	msgID, _ := uuid.NewV4()
	writeSSE(w, flusher, "message_start", anthropicMessageStart(msgID.String(), model))
	writeSSE(w, flusher, "content_block_start", anthropicContentBlockStart(0))

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	outputTokens := 0
	finishReason := "end_turn"

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") || line == "data: [DONE]" {
			continue
		}

		payload := strings.TrimPrefix(line, "data: ")
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				writeSSE(w, flusher, "content_block_delta", anthropicContentBlockDelta(0, delta))
			}
			if chunk.Choices[0].FinishReason != nil {
				if *chunk.Choices[0].FinishReason == "length" {
					finishReason = "max_tokens"
				}
			}
		}
		if chunk.Usage != nil {
			outputTokens = chunk.Usage.CompletionTokens
		}
	}

	writeSSE(w, flusher, "content_block_stop", anthropicContentBlockStop(0))
	writeSSE(w, flusher, "message_delta", anthropicMessageDelta(finishReason, outputTokens))
	writeSSE(w, flusher, "message_stop", anthropicMessageStop())

	usageData.Status = mgmtpb.Status_STATUS_COMPLETED
	if runLog != nil {
		now := time.Now()
		runLog.EndTime = null.TimeFrom(now)
		runLog.TotalDuration = null.IntFrom(now.Sub(runLog.CreateTime).Milliseconds())
		runLog.Status = datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_COMPLETED)
		_ = s.GetRepository().UpdateModelRun(ctx, runLog)
	}
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
