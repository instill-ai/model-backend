package handler

import (
	"bufio"
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

// HandleChatCompletions handles POST /v1/chat/completions by authenticating
// the caller, resolving the model, and proxying the request to Ray Serve HTTP.
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

	appName, err := ray.GetApplicationMetadataValue(modelName, version.Version)
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "internal error resolving model", "server_error", "")
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

	targetURL := fmt.Sprintf("http://%s:%d/%s/v1/chat/completions",
		config.Config.Ray.Host,
		config.Config.Ray.Port.SERVE,
		appName,
	)

	proxyReq, err := http.NewRequestWithContext(req.Context(), http.MethodPost, targetURL, strings.NewReader(string(body)))
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		writeOpenAIError(w, http.StatusInternalServerError, "failed to create proxy request", "server_error", "")
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
		writeOpenAIError(w, http.StatusBadGateway, "failed to reach model backend", "server_error", "upstream_error")
		return
	}
	defer resp.Body.Close()

	if chatReq.Stream {
		streamSSEResponse(ctx, w, resp, s, runLog, usageData, logger)
	} else {
		proxyNonStreamResponse(w, resp, s, ctx, runLog, usageData)
	}
}

func streamSSEResponse(ctx context.Context, w http.ResponseWriter, resp *http.Response, s service.Service, runLog *datamodel.ModelRun, usageData *utils.UsageMetricData, logger *zap.Logger) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeOpenAIError(w, http.StatusInternalServerError, "streaming not supported", "server_error", "")
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(w, "%s\n", line)
		flusher.Flush()

		if strings.HasPrefix(line, "data: ") && line != "data: [DONE]" {
			extractStreamingUsage(line, usageData)
		}
	}

	usageData.Status = mgmtpb.Status_STATUS_COMPLETED
	if runLog != nil {
		updateRunCompleted(ctx, s, runLog)
	}
}

func proxyNonStreamResponse(w http.ResponseWriter, resp *http.Response, s service.Service, ctx context.Context, runLog *datamodel.ModelRun, usageData *utils.UsageMetricData) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		usageData.Status = mgmtpb.Status_STATUS_ERRORED
		writeOpenAIError(w, http.StatusBadGateway, "failed to read upstream response", "server_error", "")
		return
	}

	usageData.Status = mgmtpb.Status_STATUS_COMPLETED

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)

	if runLog != nil {
		updateRunCompleted(ctx, s, runLog)
	}
}

// extractStreamingUsage looks for usage data in an SSE data line.
func extractStreamingUsage(line string, usageData *utils.UsageMetricData) {
	payload := strings.TrimPrefix(line, "data: ")
	var chunk struct {
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal([]byte(payload), &chunk); err == nil && chunk.Usage != nil {
		_ = chunk.Usage // token counts available for future billing integration
	}
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
