package model

type Setting struct {
	Identifier string         `json:"identifier,omitempty"`
	Key        string         `json:"key,omitempty"`
	Value      any            `json:"value,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}
