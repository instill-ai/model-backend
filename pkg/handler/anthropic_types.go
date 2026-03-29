package handler

import (
	"encoding/json"
	"strings"
)

// anthropicRequest is the Anthropic Messages API request format.
// Only the fields needed for translation are typed; the rest pass through.
type anthropicRequest struct {
	Model       string           `json:"model"`
	Messages    []anthropicMsg   `json:"messages"`
	System      json.RawMessage  `json:"system,omitempty"`
	MaxTokens   int              `json:"max_tokens"`
	Stream      bool             `json:"stream,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	TopP        *float64         `json:"top_p,omitempty"`
	StopSeqs    []string         `json:"stop_sequences,omitempty"`
	Metadata    *json.RawMessage `json:"metadata,omitempty"`
}

// SystemText extracts a plain text system prompt from the System field,
// which can be either a JSON string or an array of content blocks.
func (r anthropicRequest) SystemText() string {
	if len(r.System) == 0 || string(r.System) == "null" {
		return ""
	}

	var s string
	if err := json.Unmarshal(r.System, &s); err == nil {
		return s
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(r.System, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}

	return ""
}

type anthropicMsg struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// anthropicResponse is the Anthropic non-streaming response.
type anthropicResponse struct {
	ID           string               `json:"id"`
	Type         string               `json:"type"`
	Role         string               `json:"role"`
	Content      []anthropicContent   `json:"content"`
	Model        string               `json:"model"`
	StopReason   *string              `json:"stop_reason"`
	StopSequence *string              `json:"stop_sequence"`
	Usage        anthropicUsage       `json:"usage"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Anthropic SSE event helpers

func anthropicMessageStart(id, model string) map[string]any {
	return map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":      id,
			"type":    "message",
			"role":    "assistant",
			"content": []any{},
			"model":   model,
			"usage":   map[string]int{"input_tokens": 0, "output_tokens": 0},
		},
	}
}

func anthropicContentBlockStart(idx int) map[string]any {
	return map[string]any{
		"type":          "content_block_start",
		"index":         idx,
		"content_block": map[string]string{"type": "text", "text": ""},
	}
}

func anthropicContentBlockDelta(idx int, text string) map[string]any {
	return map[string]any{
		"type":  "content_block_delta",
		"index": idx,
		"delta": map[string]string{"type": "text_delta", "text": text},
	}
}

func anthropicContentBlockStop(idx int) map[string]any {
	return map[string]any{
		"type":  "content_block_stop",
		"index": idx,
	}
}

func anthropicMessageDelta(stopReason string, outputTokens int) map[string]any {
	return map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": map[string]int{"output_tokens": outputTokens},
	}
}

func anthropicMessageStop() map[string]any {
	return map[string]any{"type": "message_stop"}
}
