package model

// Profile describes the public fields commonly used in Nightscout profile
// documents.
type Profile struct {
	Identifier     string         `json:"identifier,omitempty"`
	CreatedAt      string         `json:"created_at,omitempty"`
	DefaultProfile string         `json:"defaultProfile,omitempty"`
	StartDate      string         `json:"startDate,omitempty"`
	Store          map[string]any `json:"store,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
}
