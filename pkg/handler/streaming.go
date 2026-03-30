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

type llamaServerRequest struct {
	Model         string           `json:"model"`
	Messages      []llamaServerMsg `json:"messages"`
	MaxTokens     *int             `json:"max_tokens,omitempty"`
	Temperature   *float64         `json:"temperature,omitempty"`
	TopP          *float64         `json:"top_p,omitempty"`
	Stream        bool             `json:"stream"`
	Seed          *int             `json:"seed,omitempty"`
	StreamOptions *streamOptions   `json:"stream_options,omitempty"`
}

type llamaServerMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type streamUsage struct {
	InputTokens  int
	OutputTokens int
}

func openaiToLlamaRequest(chatReq openaiChatRequest) llamaServerRequest {
	msgs := make([]llamaServerMsg, 0, len(chatReq.Messages))
	for _, m := range chatReq.Messages {
		msgs = append(msgs, llamaServerMsg{Role: m.Role, Content: flattenContent(m.Content)})
	}
	return llamaServerRequest{
		Model:         "default",
		Messages:      msgs,
		MaxTokens:     chatReq.MaxTokens,
		Temperature:   chatReq.Temperature,
		TopP:          chatReq.TopP,
		Stream:        true,
		Seed:          chatReq.Seed,
		StreamOptions: &streamOptions{IncludeUsage: true},
	}
}

func anthropicToLlamaRequest(antReq anthropicRequest) llamaServerRequest {
	msgs := make([]llamaServerMsg, 0, len(antReq.Messages)+1)
	if sysText := antReq.SystemText(); sysText != "" {
		msgs = append(msgs, llamaServerMsg{Role: "system", Content: sysText})
	}
	for _, m := range antReq.Messages {
		msgs = append(msgs, llamaServerMsg{Role: m.Role, Content: flattenContent(m.Content)})
	}
	maxTokens := antReq.MaxTokens
	return llamaServerRequest{
		Model:         "default",
		Messages:      msgs,
		MaxTokens:     &maxTokens,
		Temperature:   antReq.Temperature,
		TopP:          antReq.TopP,
		Stream:        true,
		StreamOptions: &streamOptions{IncludeUsage: true},
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

// doLlamaStream sends a streaming POST to the llama-server's OpenAI-compatible
// endpoint and returns the raw HTTP response for SSE consumption.
func doLlamaStream(ctx context.Context, llamaBaseURL string, req llamaServerRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimSuffix(llamaBaseURL, "/") + "/chat/completions"
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
		return nil, fmt.Errorf("llama-server returned %d: %s", resp.StatusCode, string(errBody))
	}

	return resp, nil
}

// forwardOpenAIStream reads SSE chunks from llama-server and forwards them to
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

		var chunk openaiStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err == nil {
			chunk.ID = "chatcmpl-" + chatID
			chunk.Model = model
			if chunk.Usage != nil {
				usage.InputTokens = chunk.Usage.PromptTokens
				usage.OutputTokens = chunk.Usage.CompletionTokens
			}
			if rewritten, err := json.Marshal(chunk); err == nil {
				data = string(rewritten)
			}
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	return usage
}

// forwardAsAnthropicStream reads OpenAI SSE chunks from llama-server and
// translates them into Anthropic Messages SSE events on-the-fly.
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
	writeSSE(w, flusher, "content_block_start", anthropicContentBlockStart(0))

	var usage streamUsage
	stopReason := "end_turn"

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

		var chunk openaiStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			logger.Warn(fmt.Sprintf("failed to parse llama-server chunk: %v", err))
			continue
		}

		if chunk.Usage != nil {
			usage.InputTokens = chunk.Usage.PromptTokens
			usage.OutputTokens = chunk.Usage.CompletionTokens
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				writeSSE(w, flusher, "content_block_delta", anthropicContentBlockDelta(0, choice.Delta.Content))
			}
			if choice.FinishReason != nil {
				if *choice.FinishReason == "length" {
					stopReason = "max_tokens"
				}
			}
		}
	}

	writeSSE(w, flusher, "content_block_stop", anthropicContentBlockStop(0))
	writeSSE(w, flusher, "message_delta", anthropicMessageDelta(stopReason, usage.OutputTokens))
	writeSSE(w, flusher, "message_stop", anthropicMessageStop())

	return usage
}
