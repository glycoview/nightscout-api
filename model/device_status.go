package model

// DeviceStatus describes the public fields commonly used in Nightscout device
// status documents.
type DeviceStatus struct {
	Identifier string         `json:"identifier,omitempty"`
	CreatedAt  string         `json:"created_at,omitempty"`
	Date       int64          `json:"date,omitempty"`
	UTCOffset  int            `json:"utcOffset,omitempty"`
	Device     string         `json:"device,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}
