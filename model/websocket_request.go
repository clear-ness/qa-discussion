package model

import (
	"encoding/json"
	"io"
)

type WebSocketRequest struct {
	// Client-provided fields
	Seq    int64                  `json:"seq"`
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data"` // The metadata for an action.

	// Server-provided fields
	Session Session `json:"-"`
}

func (o *WebSocketRequest) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

func WebSocketRequestFromJson(data io.Reader) *WebSocketRequest {
	var o *WebSocketRequest
	json.NewDecoder(data).Decode(&o)
	return o
}
