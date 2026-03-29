package handler

import (
	"encoding/json"
	"testing"
)

func TestSystemText_String(t *testing.T) {
	raw := json.RawMessage(`"You are a helpful assistant."`)
	r := anthropicRequest{System: raw}
	got := r.SystemText()
	want := "You are a helpful assistant."
	if got != want {
		t.Errorf("SystemText() = %q, want %q", got, want)
	}
}

func TestSystemText_ArraySingle(t *testing.T) {
	raw := json.RawMessage(`[{"type":"text","text":"You are a helpful assistant."}]`)
	r := anthropicRequest{System: raw}
	got := r.SystemText()
	want := "You are a helpful assistant."
	if got != want {
		t.Errorf("SystemText() = %q, want %q", got, want)
	}
}

func TestSystemText_ArrayMultiple(t *testing.T) {
	raw := json.RawMessage(`[{"type":"text","text":"You are a helpful assistant."},{"type":"text","text":"Always be concise."}]`)
	r := anthropicRequest{System: raw}
	got := r.SystemText()
	want := "You are a helpful assistant.\nAlways be concise."
	if got != want {
		t.Errorf("SystemText() = %q, want %q", got, want)
	}
}

func TestSystemText_Empty(t *testing.T) {
	r := anthropicRequest{}
	if got := r.SystemText(); got != "" {
		t.Errorf("SystemText() = %q, want empty", got)
	}
}

func TestSystemText_Null(t *testing.T) {
	raw := json.RawMessage(`null`)
	r := anthropicRequest{System: raw}
	if got := r.SystemText(); got != "" {
		t.Errorf("SystemText() = %q, want empty", got)
	}
}

func TestConvertAnthropicContent_SimpleString(t *testing.T) {
	raw := json.RawMessage(`"Hello world"`)
	parts, err := convertAnthropicContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
	m := parts[0].(map[string]any)
	if m["type"] != "text" || m["text"] != "Hello world" {
		t.Errorf("got %v, want text='Hello world'", m)
	}
}

func TestConvertAnthropicContent_SingleBlock(t *testing.T) {
	raw := json.RawMessage(`[{"type":"text","text":"Hello world"}]`)
	parts, err := convertAnthropicContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
	m := parts[0].(map[string]any)
	if m["text"] != "Hello world" {
		t.Errorf("got text=%q, want 'Hello world'", m["text"])
	}
}

func TestConvertAnthropicContent_MultipleTextBlocks(t *testing.T) {
	raw := json.RawMessage(`[{"type":"text","text":"Context: This is context"},{"type":"text","text":"Question: What is 1+1?"}]`)
	parts, err := convertAnthropicContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 merged part, got %d", len(parts))
	}
	m := parts[0].(map[string]any)
	want := "Context: This is context\nQuestion: What is 1+1?"
	if m["text"] != want {
		t.Errorf("got text=%q, want %q", m["text"], want)
	}
}

func TestConvertAnthropicContent_EmptyRaw(t *testing.T) {
	parts, err := convertAnthropicContent(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
	m := parts[0].(map[string]any)
	if m["text"] != "" {
		t.Errorf("got text=%q, want empty", m["text"])
	}
}

func TestConvertAnthropicContent_NullRaw(t *testing.T) {
	raw := json.RawMessage(`null`)
	parts, err := convertAnthropicContent(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
}

func TestAnthropicToInstillTaskInput_ClaudeCodePayload(t *testing.T) {
	payload := `{
		"model": "instill/glm-5",
		"max_tokens": 4096,
		"stream": true,
		"system": [
			{"type": "text", "text": "System prompt part 1."},
			{"type": "text", "text": "System prompt part 2."}
		],
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "Context block"},
					{"type": "text", "text": "Actual question"}
				]
			}
		]
	}`

	var antReq anthropicRequest
	if err := json.Unmarshal([]byte(payload), &antReq); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	taskInput, err := anthropicToInstillTaskInput(antReq, "glm-5")
	if err != nil {
		t.Fatalf("anthropicToInstillTaskInput: %v", err)
	}

	data := taskInput.Fields["data"].GetStructValue()
	msgs := data.Fields["messages"].GetListValue().Values
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (system + user), got %d", len(msgs))
	}

	sysMsg := msgs[0].GetStructValue()
	if sysMsg.Fields["role"].GetStringValue() != "system" {
		t.Error("first message role should be system")
	}
	sysContent := sysMsg.Fields["content"].GetListValue().Values
	if len(sysContent) != 1 {
		t.Fatalf("system message should have 1 content block, got %d", len(sysContent))
	}
	sysText := sysContent[0].GetStructValue().Fields["text"].GetStringValue()
	if sysText != "System prompt part 1.\nSystem prompt part 2." {
		t.Errorf("system text = %q", sysText)
	}

	userMsg := msgs[1].GetStructValue()
	if userMsg.Fields["role"].GetStringValue() != "user" {
		t.Error("second message role should be user")
	}
	userContent := userMsg.Fields["content"].GetListValue().Values
	if len(userContent) != 1 {
		t.Fatalf("user message should have 1 content block (merged), got %d", len(userContent))
	}
	userText := userContent[0].GetStructValue().Fields["text"].GetStringValue()
	if userText != "Context block\nActual question" {
		t.Errorf("user text = %q", userText)
	}
}

func TestFullRequestUnmarshal_ClaudeCodeStyle(t *testing.T) {
	// Simulate what Claude Code actually sends.
	payload := `{
		"model": "instill/glm-5",
		"max_tokens": 4096,
		"stream": true,
		"system": [
			{"type": "text", "text": "You are an interactive AI assistant."},
			{"type": "text", "text": "IMPORTANT: Follow conventions."}
		],
		"messages": [
			{
				"role": "user",
				"content": [
					{"type": "text", "text": "<context>\nHere is some codebase context.\n</context>"},
					{"type": "text", "text": "Please explain how authentication works."}
				]
			}
		]
	}`

	var antReq anthropicRequest
	if err := json.Unmarshal([]byte(payload), &antReq); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	sysText := antReq.SystemText()
	wantSys := "You are an interactive AI assistant.\nIMPORTANT: Follow conventions."
	if sysText != wantSys {
		t.Errorf("SystemText() = %q, want %q", sysText, wantSys)
	}

	if len(antReq.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(antReq.Messages))
	}

	parts, err := convertAnthropicContent(antReq.Messages[0].Content)
	if err != nil {
		t.Fatalf("convertAnthropicContent error: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 merged part, got %d", len(parts))
	}
	m := parts[0].(map[string]any)
	wantContent := "<context>\nHere is some codebase context.\n</context>\nPlease explain how authentication works."
	if m["text"] != wantContent {
		t.Errorf("merged text = %q, want %q", m["text"], wantContent)
	}
}
