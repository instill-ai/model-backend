package handler

import (
	"encoding/json"
	"net/http"
)

// openaiChatRequest holds only the fields we need to inspect from the incoming
// request.  The full body is forwarded to Ray Serve as-is so that we stay
// compatible with any new fields the OpenAI spec may add.
type openaiChatRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream,omitempty"`
}

// openaiModel is the OpenAI model object returned by GET /v1/models.
type openaiModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// openaiModelList is the response for GET /v1/models.
type openaiModelList struct {
	Object string        `json:"object"`
	Data   []openaiModel `json:"data"`
}

func writeOpenAIError(w http.ResponseWriter, status int, message, errType, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errType,
			"code":    code,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func writeOpenAIJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
