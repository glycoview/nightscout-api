package model

// Setting describes the public fields commonly used in Nightscout settings
// documents.
type Setting struct {
	Identifier string         `json:"identifier,omitempty"`
	Key        string         `json:"key,omitempty"`
	Value      any            `json:"value,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}
