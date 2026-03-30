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

	logx "github.com/instill-ai/x/log"
)

type inferenceServerRequest struct {
	Model         string               `json:"model"`
	Messages      []inferenceServerMsg `json:"messages"`
	MaxTokens     *int                 `json:"max_tokens,omitempty"`
	Temperature   *float64             `json:"temperature,omitempty"`
	TopP          *float64             `json:"top_p,omitempty"`
	Stream        bool                 `json:"stream"`
	Seed          *int                 `json:"seed,omitempty"`
	StreamOptions *streamOptions       `json:"stream_options,omitempty"`
	Tools         json.RawMessage      `json:"tools,omitempty"`
	ToolChoice    json.RawMessage      `json:"tool_choice,omitempty"`
}

type inferenceServerMsg struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type streamUsage struct {
	InputTokens  int
	OutputTokens int
}

func openaiToInferenceRequest(chatReq openaiChatRequest) inferenceServerRequest {
	msgs := make([]inferenceServerMsg, 0, len(chatReq.Messages))
	for _, m := range chatReq.Messages {
		msg := inferenceServerMsg{
			Role:       m.Role,
			Content:    flattenContent(m.Content),
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
		}
		msgs = append(msgs, msg)
	}
	return inferenceServerRequest{
		Model:         "default",
		Messages:      msgs,
		MaxTokens:     chatReq.MaxTokens,
		Temperature:   chatReq.Temperature,
		TopP:          chatReq.TopP,
		Stream:        true,
		Seed:          chatReq.Seed,
		StreamOptions: &streamOptions{IncludeUsage: true},
		Tools:         chatReq.Tools,
		ToolChoice:    chatReq.ToolChoice,
	}
}

func anthropicToInferenceRequest(antReq anthropicRequest) inferenceServerRequest {
	msgs := make([]inferenceServerMsg, 0, len(antReq.Messages)+1)
	if sysText := antReq.SystemText(); sysText != "" {
		msgs = append(msgs, inferenceServerMsg{Role: "system", Content: sysText})
	}
	for _, m := range antReq.Messages {
		converted := convertAnthropicMsgToOpenAI(m)
		msgs = append(msgs, converted...)
	}
	maxTokens := antReq.MaxTokens

	var tools json.RawMessage
	if len(antReq.Tools) > 0 {
		tools = convertAnthropicToolsToOpenAI(antReq.Tools)
	}

	var toolChoice json.RawMessage
	if len(antReq.ToolChoice) > 0 {
		toolChoice = convertAnthropicToolChoiceToOpenAI(antReq.ToolChoice)
	}

	return inferenceServerRequest{
		Model:         "default",
		Messages:      msgs,
		MaxTokens:     &maxTokens,
		Temperature:   antReq.Temperature,
		TopP:          antReq.TopP,
		Stream:        true,
		StreamOptions: &streamOptions{IncludeUsage: true},
		Tools:         tools,
		ToolChoice:    toolChoice,
	}
}

// convertAnthropicMsgToOpenAI converts a single Anthropic message into one or
// more OpenAI-format messages. An assistant message with tool_use blocks becomes
// a message with tool_calls. A user message with tool_result blocks becomes
// one or more "tool" role messages.
func convertAnthropicMsgToOpenAI(m anthropicMsg) []inferenceServerMsg {
	// Try parsing content as an array of blocks.
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(m.Content, &blocks); err != nil {
		// Simple string content.
		return []inferenceServerMsg{{Role: m.Role, Content: flattenContent(m.Content)}}
	}

	// Classify blocks by type.
	var textParts []string
	var toolUseCalls []json.RawMessage
	var toolResults []inferenceServerMsg

	for _, b := range blocks {
		typeBytes, ok := b["type"]
		if !ok {
			continue
		}
		var blockType string
		_ = json.Unmarshal(typeBytes, &blockType)

		switch blockType {
		case "text":
			var text string
			if t, ok := b["text"]; ok {
				_ = json.Unmarshal(t, &text)
			}
			if text != "" {
				textParts = append(textParts, text)
			}

		case "tool_use":
			var id, name string
			if v, ok := b["id"]; ok {
				_ = json.Unmarshal(v, &id)
			}
			if v, ok := b["name"]; ok {
				_ = json.Unmarshal(v, &name)
			}
			args := "{}"
			if v, ok := b["input"]; ok {
				args = string(v)
			}
			tc, _ := json.Marshal(map[string]any{
				"id":   id,
				"type": "function",
				"function": map[string]string{
					"name":      name,
					"arguments": args,
				},
			})
			toolUseCalls = append(toolUseCalls, tc)

		case "tool_result":
			var toolCallID string
			if v, ok := b["tool_call_id"]; ok {
				_ = json.Unmarshal(v, &toolCallID)
			}
			content := ""
			if v, ok := b["content"]; ok {
				// content can be a string or array of content blocks.
				var s string
				if err := json.Unmarshal(v, &s); err == nil {
					content = s
				} else {
					content = string(v)
				}
			}
			toolResults = append(toolResults, inferenceServerMsg{
				Role:       "tool",
				Content:    content,
				ToolCallID: toolCallID,
			})
		}
	}

	var result []inferenceServerMsg

	if m.Role == "assistant" && len(toolUseCalls) > 0 {
		finalTC, _ := json.Marshal(toolUseCalls)
		result = append(result, inferenceServerMsg{
			Role:      "assistant",
			Content:   strings.Join(textParts, "\n"),
			ToolCalls: finalTC,
		})
	} else if len(toolResults) > 0 {
		// For user messages containing tool_result blocks, emit text first then
		// tool messages.
		if len(textParts) > 0 {
			result = append(result, inferenceServerMsg{
				Role:    m.Role,
				Content: strings.Join(textParts, "\n"),
			})
		}
		result = append(result, toolResults...)
	} else {
		result = append(result, inferenceServerMsg{
			Role:    m.Role,
			Content: strings.Join(textParts, "\n"),
		})
	}

	return result
}

// convertAnthropicToolsToOpenAI converts Anthropic tool definitions to OpenAI
// function-calling format.
func convertAnthropicToolsToOpenAI(tools []anthropicTool) json.RawMessage {
	openaiTools := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		openaiTools = append(openaiTools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  json.RawMessage(t.InputSchema),
			},
		})
	}
	data, _ := json.Marshal(openaiTools)
	return data
}

// convertAnthropicToolChoiceToOpenAI translates Anthropic tool_choice to OpenAI.
func convertAnthropicToolChoiceToOpenAI(raw json.RawMessage) json.RawMessage {
	var choice map[string]any
	if err := json.Unmarshal(raw, &choice); err != nil {
		return nil
	}
	t, _ := choice["type"].(string)
	switch t {
	case "auto":
		data, _ := json.Marshal("auto")
		return data
	case "any":
		data, _ := json.Marshal("required")
		return data
	case "tool":
		name, _ := choice["name"].(string)
		data, _ := json.Marshal(map[string]any{
			"type":     "function",
			"function": map[string]string{"name": name},
		})
		return data
	default:
		return nil
	}
}

// flattenContent extracts plain text from a JSON content field that is either
// a string or an array of content-part objects.
func flattenContent(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	var parts []map[string]any
	if err := json.Unmarshal(raw, &parts); err == nil {
		var texts []string
		for _, p := range parts {
			if p["type"] == "text" {
				if t, ok := p["text"].(string); ok && t != "" {
					texts = append(texts, t)
				}
			}
		}
		return strings.Join(texts, "\n")
	}

	return string(raw)
}

// doInferenceStream sends a streaming POST to the inference server's
// OpenAI-compatible endpoint and returns the raw HTTP response for SSE consumption.
func doInferenceStream(ctx context.Context, baseURL string, req inferenceServerRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimSuffix(baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// No timeout on the client; context cancellation handles disconnects.
	// Streaming responses are long-lived (minutes for large generations).
	resp, err := (&http.Client{}).Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("inference server returned %d: %s", resp.StatusCode, string(errBody))
	}

	return resp, nil
}

// forwardOpenAIStream reads SSE chunks from inference server and forwards them to
// the client, rewriting the chunk ID and model fields. Returns token usage.
func forwardOpenAIStream(w http.ResponseWriter, resp *http.Response, chatID, model string) streamUsage {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeOpenAIError(w, http.StatusInternalServerError, "streaming not supported", "server_error", "")
		return streamUsage{}
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	var usage streamUsage
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			break
		}

		// Pass through the raw chunk, only rewriting ID and model.
		// This preserves tool_calls, tool_call_id, and all other fields
		// that vLLM produces (including function calling chunks).
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(data), &raw); err == nil {
			if idBytes, _ := json.Marshal("chatcmpl-" + chatID); idBytes != nil {
				raw["id"] = idBytes
			}
			if modelBytes, _ := json.Marshal(model); modelBytes != nil {
				raw["model"] = modelBytes
			}
			// Extract usage for metrics.
			var chunk openaiStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err == nil && chunk.Usage != nil {
				usage.InputTokens = chunk.Usage.PromptTokens
				usage.OutputTokens = chunk.Usage.CompletionTokens
			}
			if rewritten, err := json.Marshal(raw); err == nil {
				data = string(rewritten)
			}
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	return usage
}

// forwardAsAnthropicStream reads OpenAI SSE chunks from inference server and
// translates them into Anthropic Messages SSE events on-the-fly, including
// tool_calls → tool_use translation.
func forwardAsAnthropicStream(w http.ResponseWriter, resp *http.Response, msgID, model string) streamUsage {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAnthropicError(w, http.StatusInternalServerError, "streaming not supported", "api_error")
		return streamUsage{}
	}
	defer resp.Body.Close()

	ctx := resp.Request.Context()
	logger, _ := logx.GetZapLogger(ctx)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	writeSSE(w, flusher, "message_start", anthropicMessageStart(msgID, model))

	var usage streamUsage
	stopReason := "end_turn"
	textBlockStarted := false
	// Track tool call state: anthropicIdx is the next content_block index.
	anthropicIdx := 0
	// openaiIdxToAnthropicIdx maps OpenAI tool_call index to Anthropic block index.
	openaiIdxToAnthropicIdx := map[int]int{}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			break
		}

		// Parse into a generic structure to access tool_calls which aren't
		// in the typed openaiStreamChunk.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(data), &raw); err != nil {
			logger.Warn(fmt.Sprintf("failed to parse chunk: %v", err))
			continue
		}

		// Extract usage.
		var chunk openaiStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err == nil && chunk.Usage != nil {
			usage.InputTokens = chunk.Usage.PromptTokens
			usage.OutputTokens = chunk.Usage.CompletionTokens
		}

		// Parse choices.
		var choices []struct {
			Index        int     `json:"index"`
			Delta        json.RawMessage `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		}
		if rawChoices, ok := raw["choices"]; ok {
			_ = json.Unmarshal(rawChoices, &choices)
		}

		for _, choice := range choices {
			// Parse delta.
			var delta struct {
				Content   string          `json:"content"`
				Role      string          `json:"role"`
				ToolCalls json.RawMessage `json:"tool_calls"`
			}
			_ = json.Unmarshal(choice.Delta, &delta)

			// Handle text content.
			if delta.Content != "" {
				if !textBlockStarted {
					writeSSE(w, flusher, "content_block_start", anthropicContentBlockStart(anthropicIdx))
					textBlockStarted = true
				}
				writeSSE(w, flusher, "content_block_delta", anthropicContentBlockDelta(anthropicIdx, delta.Content))
			}

			// Handle tool_calls.
			if len(delta.ToolCalls) > 0 {
				// Close text block if open before starting tool blocks.
				if textBlockStarted {
					writeSSE(w, flusher, "content_block_stop", anthropicContentBlockStop(anthropicIdx))
					anthropicIdx++
					textBlockStarted = false
				}

				var tcs []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				}
				_ = json.Unmarshal(delta.ToolCalls, &tcs)

				for _, tc := range tcs {
					aIdx, exists := openaiIdxToAnthropicIdx[tc.Index]
					if !exists {
						// New tool call: emit content_block_start.
						aIdx = anthropicIdx
						openaiIdxToAnthropicIdx[tc.Index] = aIdx
						anthropicIdx++
						writeSSE(w, flusher, "content_block_start", map[string]any{
							"type":  "content_block_start",
							"index": aIdx,
							"content_block": map[string]any{
								"type":  "tool_use",
								"id":    tc.ID,
								"name":  tc.Function.Name,
								"input": map[string]any{},
							},
						})
					}
					// Stream argument fragments.
					if tc.Function.Arguments != "" {
						writeSSE(w, flusher, "content_block_delta", map[string]any{
							"type":  "content_block_delta",
							"index": aIdx,
							"delta": map[string]string{
								"type":         "input_json_delta",
								"partial_json": tc.Function.Arguments,
							},
						})
					}
				}
			}

			if choice.FinishReason != nil {
				switch *choice.FinishReason {
				case "tool_calls":
					stopReason = "tool_use"
				case "length":
					stopReason = "max_tokens"
				}
			}
		}
	}

	// Close any open blocks.
	if textBlockStarted {
		writeSSE(w, flusher, "content_block_stop", anthropicContentBlockStop(anthropicIdx))
	}
	for _, aIdx := range openaiIdxToAnthropicIdx {
		writeSSE(w, flusher, "content_block_stop", anthropicContentBlockStop(aIdx))
	}
	// If no blocks were emitted at all, emit an empty text block.
	if !textBlockStarted && len(openaiIdxToAnthropicIdx) == 0 {
		writeSSE(w, flusher, "content_block_start", anthropicContentBlockStart(0))
		writeSSE(w, flusher, "content_block_stop", anthropicContentBlockStop(0))
	}

	writeSSE(w, flusher, "message_delta", anthropicMessageDelta(stopReason, usage.OutputTokens))
	writeSSE(w, flusher, "message_stop", anthropicMessageStop())

	return usage
}
