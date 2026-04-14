package model

type Food struct {
	Identifier string         `json:"identifier,omitempty"`
	CreatedAt  string         `json:"created_at,omitempty"`
	Name       string         `json:"name,omitempty"`
	Category   string         `json:"category,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}
