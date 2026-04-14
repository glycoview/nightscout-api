package model

// Entry describes the public fields commonly used in Nightscout entry
// documents.
type Entry struct {
	Identifier string         `json:"identifier,omitempty"`
	Type       string         `json:"type,omitempty"`
	SGV        any            `json:"sgv,omitempty"`
	MBG        any            `json:"mbg,omitempty"`
	Date       int64          `json:"date,omitempty"`
	DateString string         `json:"dateString,omitempty"`
	UTCOffset  int            `json:"utcOffset,omitempty"`
	Device     string         `json:"device,omitempty"`
	Direction  string         `json:"direction,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}
