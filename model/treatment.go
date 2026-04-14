package model

type Treatment struct {
	Identifier  string         `json:"identifier,omitempty"`
	EventType   string         `json:"eventType,omitempty"`
	CreatedAt   string         `json:"created_at,omitempty"`
	Date        int64          `json:"date,omitempty"`
	UTCOffset   int            `json:"utcOffset,omitempty"`
	Insulin     any            `json:"insulin,omitempty"`
	Carbs       any            `json:"carbs,omitempty"`
	Glucose     any            `json:"glucose,omitempty"`
	GlucoseType string         `json:"glucoseType,omitempty"`
	Notes       string         `json:"notes,omitempty"`
	EnteredBy   string         `json:"enteredBy,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}
