package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockVLLMToolCallStream returns SSE chunks that simulate a vLLM response with
// a tool_call (function calling). This is what vLLM produces when
// --enable-auto-tool-choice and --tool-call-parser hermes are active.
func mockVLLMToolCallStream() string {
	chunks := []string{
		// Chunk 1: role
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"default","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
		// Chunk 2: tool_call start (id + function name)
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"default","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_abc123","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":null}]}`,
		// Chunk 3: tool_call arguments fragment 1
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"default","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"location\":"}}]},"finish_reason":null}]}`,
		// Chunk 4: tool_call arguments fragment 2
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"default","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"San Francisco\"}"}}]},"finish_reason":null}]}`,
		// Chunk 5: finish with tool_calls reason
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"default","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":50,"completion_tokens":20,"total_tokens":70}}`,
	}

	var sb strings.Builder
	for _, c := range chunks {
		sb.WriteString("data: ")
		sb.WriteString(c)
		sb.WriteString("\n\n")
	}
	sb.WriteString("data: [DONE]\n\n")
	return sb.String()
}

// mockVLLMTextStream returns SSE chunks that simulate a plain text response.
func mockVLLMTextStream() string {
	chunks := []string{
		`{"id":"chatcmpl-456","object":"chat.completion.chunk","created":1700000000,"model":"default","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
		`{"id":"chatcmpl-456","object":"chat.completion.chunk","created":1700000000,"model":"default","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-456","object":"chat.completion.chunk","created":1700000000,"model":"default","choices":[{"index":0,"delta":{"content":" world!"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-456","object":"chat.completion.chunk","created":1700000000,"model":"default","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
	}

	var sb strings.Builder
	for _, c := range chunks {
		sb.WriteString("data: ")
		sb.WriteString(c)
		sb.WriteString("\n\n")
	}
	sb.WriteString("data: [DONE]\n\n")
	return sb.String()
}

func startMockVLLM(t *testing.T, sseBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate the request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("expected /chat/completions path, got %s", r.URL.Path)
		}

		var req inferenceServerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			http.Error(w, "bad request", 400)
			return
		}

		// Validate tools and tool_choice are forwarded
		if len(req.Tools) > 0 {
			t.Logf("Received tools: %s", string(req.Tools))
		}
		if len(req.ToolChoice) > 0 {
			t.Logf("Received tool_choice: %s", string(req.ToolChoice))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sseBody)
	}))
}

func TestOpenAIToInferenceRequest_WithTools(t *testing.T) {
	tools := json.RawMessage(`[{"type":"function","function":{"name":"get_weather","description":"Get weather","parameters":{"type":"object","properties":{"location":{"type":"string"}}}}}]`)
	toolChoice := json.RawMessage(`"auto"`)

	chatReq := openaiChatRequest{
		Model: "test/model",
		Messages: []openaiMessage{
			{Role: "user", Content: json.RawMessage(`"What's the weather?"`)},
		},
		Tools:      tools,
		ToolChoice: toolChoice,
		Stream:     true,
	}

	inferReq := openaiToInferenceRequest(chatReq)

	if string(inferReq.Tools) != string(tools) {
		t.Errorf("tools not forwarded: got %s, want %s", string(inferReq.Tools), string(tools))
	}
	if string(inferReq.ToolChoice) != string(toolChoice) {
		t.Errorf("tool_choice not forwarded: got %s, want %s", string(inferReq.ToolChoice), string(toolChoice))
	}
	if inferReq.Model != "default" {
		t.Errorf("model should be 'default', got %s", inferReq.Model)
	}
	if !inferReq.Stream {
		t.Error("stream should be true")
	}
}

func TestAnthropicToInferenceRequest_WithTools(t *testing.T) {
	antReq := anthropicRequest{
		Model: "test/model",
		Messages: []anthropicMsg{
			{Role: "user", Content: json.RawMessage(`"What's the weather?"`)},
		},
		MaxTokens: 1024,
		Tools: []anthropicTool{
			{
				Name:        "get_weather",
				Description: "Get current weather",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
			},
		},
		ToolChoice: json.RawMessage(`{"type":"auto"}`),
	}

	inferReq := anthropicToInferenceRequest(antReq)

	// Tools should be converted to OpenAI format
	var tools []map[string]any
	if err := json.Unmarshal(inferReq.Tools, &tools); err != nil {
		t.Fatalf("failed to unmarshal tools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0]["type"] != "function" {
		t.Errorf("expected function type, got %v", tools[0]["type"])
	}
	fn := tools[0]["function"].(map[string]any)
	if fn["name"] != "get_weather" {
		t.Errorf("expected get_weather, got %v", fn["name"])
	}

	// ToolChoice should be converted from {"type":"auto"} to "auto"
	var choice string
	if err := json.Unmarshal(inferReq.ToolChoice, &choice); err != nil {
		t.Fatalf("failed to unmarshal tool_choice: %v", err)
	}
	if choice != "auto" {
		t.Errorf("expected 'auto', got %s", choice)
	}
}

func TestConvertAnthropicToolChoiceToOpenAI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"auto", `{"type":"auto"}`, `"auto"`},
		{"any", `{"type":"any"}`, `"required"`},
		{"tool", `{"type":"tool","name":"get_weather"}`, `{"function":{"name":"get_weather"},"type":"function"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertAnthropicToolChoiceToOpenAI(json.RawMessage(tt.input))
			// Compare as JSON to ignore key ordering
			var got, want any
			_ = json.Unmarshal(result, &got)
			_ = json.Unmarshal([]byte(tt.expected), &want)
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("got %s, want %s", string(gotJSON), string(wantJSON))
			}
		})
	}
}

func TestConvertAnthropicMsgToOpenAI_ToolUse(t *testing.T) {
	// Assistant message with tool_use blocks
	content := json.RawMessage(`[{"type":"text","text":"Let me check the weather."},{"type":"tool_use","id":"toolu_123","name":"get_weather","input":{"location":"SF"}}]`)
	msg := anthropicMsg{Role: "assistant", Content: content}

	result := convertAnthropicMsgToOpenAI(msg)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	m := result[0]
	if m.Role != "assistant" {
		t.Errorf("expected assistant role, got %s", m.Role)
	}
	if m.Content != "Let me check the weather." {
		t.Errorf("unexpected content: %s", m.Content)
	}
	if len(m.ToolCalls) == 0 {
		t.Fatal("expected tool_calls to be set")
	}

	var toolCalls []map[string]any
	if err := json.Unmarshal(m.ToolCalls, &toolCalls); err != nil {
		t.Fatalf("failed to unmarshal tool_calls: %v", err)
	}
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
	}
	if toolCalls[0]["id"] != "toolu_123" {
		t.Errorf("expected toolu_123, got %v", toolCalls[0]["id"])
	}
}

func TestConvertAnthropicMsgToOpenAI_ToolResult(t *testing.T) {
	// User message with tool_result blocks
	content := json.RawMessage(`[{"type":"tool_result","tool_call_id":"call_123","content":"72°F, sunny"}]`)
	msg := anthropicMsg{Role: "user", Content: content}

	result := convertAnthropicMsgToOpenAI(msg)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	m := result[0]
	if m.Role != "tool" {
		t.Errorf("expected tool role, got %s", m.Role)
	}
	if m.ToolCallID != "call_123" {
		t.Errorf("expected call_123, got %s", m.ToolCallID)
	}
	if m.Content != "72°F, sunny" {
		t.Errorf("unexpected content: %s", m.Content)
	}
}

func TestForwardOpenAIStream_ToolCalls(t *testing.T) {
	mock := startMockVLLM(t, mockVLLMToolCallStream())
	defer mock.Close()

	req := inferenceServerRequest{
		Model:    "default",
		Messages: []inferenceServerMsg{{Role: "user", Content: "What's the weather?"}},
		Stream:   true,
		Tools:    json.RawMessage(`[{"type":"function","function":{"name":"get_weather"}}]`),
	}

	resp, err := doInferenceStream(t.Context(), mock.URL+"/v1", req)
	if err != nil {
		t.Fatalf("doInferenceStream failed: %v", err)
	}

	rec := httptest.NewRecorder()
	usage := forwardOpenAIStream(rec, resp, "test-id", "test/model")

	body := rec.Body.String()
	t.Logf("OpenAI stream output:\n%s", body)

	// Verify tool_calls are present in the streamed output
	if !strings.Contains(body, "tool_calls") {
		t.Error("output should contain tool_calls")
	}
	if !strings.Contains(body, "get_weather") {
		t.Error("output should contain get_weather function name")
	}
	if !strings.Contains(body, "call_abc123") {
		t.Error("output should contain tool call ID")
	}
	if !strings.Contains(body, "San Francisco") {
		t.Error("output should contain the function arguments")
	}
	if !strings.Contains(body, "[DONE]") {
		t.Error("output should end with [DONE]")
	}

	// Verify model was rewritten
	if !strings.Contains(body, "test/model") {
		t.Error("model should be rewritten to test/model")
	}

	// Verify usage
	if usage.InputTokens != 50 {
		t.Errorf("expected 50 input tokens, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 20 {
		t.Errorf("expected 20 output tokens, got %d", usage.OutputTokens)
	}
}

func TestForwardAsAnthropicStream_ToolCalls(t *testing.T) {
	mock := startMockVLLM(t, mockVLLMToolCallStream())
	defer mock.Close()

	req := inferenceServerRequest{
		Model:    "default",
		Messages: []inferenceServerMsg{{Role: "user", Content: "What's the weather?"}},
		Stream:   true,
		Tools:    json.RawMessage(`[{"type":"function","function":{"name":"get_weather"}}]`),
	}

	resp, err := doInferenceStream(t.Context(), mock.URL+"/v1", req)
	if err != nil {
		t.Fatalf("doInferenceStream failed: %v", err)
	}

	rec := httptest.NewRecorder()
	usage := forwardAsAnthropicStream(rec, resp, "msg_test", "test/model")

	body := rec.Body.String()
	t.Logf("Anthropic stream output:\n%s", body)

	// Verify Anthropic SSE structure
	if !strings.Contains(body, "event: message_start") {
		t.Error("should have message_start event")
	}
	if !strings.Contains(body, "event: content_block_start") {
		t.Error("should have content_block_start event")
	}
	if !strings.Contains(body, "event: message_stop") {
		t.Error("should have message_stop event")
	}

	// Verify tool_use blocks are produced
	if !strings.Contains(body, "tool_use") {
		t.Error("should contain tool_use content block")
	}
	if !strings.Contains(body, "get_weather") {
		t.Error("should contain get_weather tool name")
	}
	if !strings.Contains(body, "input_json_delta") {
		t.Error("should contain input_json_delta for tool arguments")
	}
	if !strings.Contains(body, "San Francisco") {
		t.Error("should contain function arguments")
	}

	// Verify stop_reason is tool_use
	if !strings.Contains(body, `"stop_reason":"tool_use"`) {
		t.Error("stop_reason should be tool_use")
	}

	if usage.InputTokens != 50 {
		t.Errorf("expected 50 input tokens, got %d", usage.InputTokens)
	}
}

func TestForwardOpenAIStream_TextOnly(t *testing.T) {
	mock := startMockVLLM(t, mockVLLMTextStream())
	defer mock.Close()

	req := inferenceServerRequest{
		Model:    "default",
		Messages: []inferenceServerMsg{{Role: "user", Content: "Say hello"}},
		Stream:   true,
	}

	resp, err := doInferenceStream(t.Context(), mock.URL+"/v1", req)
	if err != nil {
		t.Fatalf("doInferenceStream failed: %v", err)
	}

	rec := httptest.NewRecorder()
	usage := forwardOpenAIStream(rec, resp, "test-id", "test/model")

	body := rec.Body.String()

	if !strings.Contains(body, "Hello") {
		t.Error("should contain Hello")
	}
	if !strings.Contains(body, " world!") {
		t.Error("should contain world!")
	}
	if !strings.Contains(body, "[DONE]") {
		t.Error("should end with [DONE]")
	}
	if usage.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", usage.InputTokens)
	}
}

func TestForwardAsAnthropicStream_TextOnly(t *testing.T) {
	mock := startMockVLLM(t, mockVLLMTextStream())
	defer mock.Close()

	req := inferenceServerRequest{
		Model:    "default",
		Messages: []inferenceServerMsg{{Role: "user", Content: "Say hello"}},
		Stream:   true,
	}

	resp, err := doInferenceStream(t.Context(), mock.URL+"/v1", req)
	if err != nil {
		t.Fatalf("doInferenceStream failed: %v", err)
	}

	rec := httptest.NewRecorder()
	usage := forwardAsAnthropicStream(rec, resp, "msg_test", "test/model")

	body := rec.Body.String()

	if !strings.Contains(body, "event: message_start") {
		t.Error("should have message_start")
	}
	if !strings.Contains(body, "Hello") {
		t.Error("should contain Hello")
	}
	if !strings.Contains(body, " world!") {
		t.Error("should contain world!")
	}
	if !strings.Contains(body, `"stop_reason":"end_turn"`) {
		t.Error("stop_reason should be end_turn")
	}
	if usage.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", usage.InputTokens)
	}
}

func TestDoInferenceStream_VLLMRejects400(t *testing.T) {
	// Simulate vLLM returning 400 when tool choice flags are not set
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		fmt.Fprint(w, `{"error":{"message":"\"auto\" tool choice requires --enable-auto-tool-choice and --tool-call-parser to be set","type":"BadRequestError","param":null,"code":400}}`)
	}))
	defer mock.Close()

	req := inferenceServerRequest{
		Model:      "default",
		Messages:   []inferenceServerMsg{{Role: "user", Content: "hi"}},
		Stream:     true,
		Tools:      json.RawMessage(`[{"type":"function","function":{"name":"bash"}}]`),
		ToolChoice: json.RawMessage(`"auto"`),
	}

	_, err := doInferenceStream(t.Context(), mock.URL+"/v1", req)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "enable-auto-tool-choice") {
		t.Errorf("error should mention enable-auto-tool-choice, got: %v", err)
	}
}
