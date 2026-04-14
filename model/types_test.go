package model

import (
	"encoding/json"
	"testing"
)

func TestModelStructJSONTags(t *testing.T) {
	cases := []struct {
		name string
		in   any
		key  string
	}{
		{"entry", Entry{Identifier: "id", DateString: "x"}, "dateString"},
		{"treatment", Treatment{EventType: "Meal Bolus"}, "eventType"},
		{"device", DeviceStatus{CreatedAt: "x"}, "created_at"},
		{"food", Food{Name: "Apple"}, "name"},
		{"profile", Profile{DefaultProfile: "default"}, "defaultProfile"},
		{"setting", Setting{Key: "units"}, "key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}
			var out map[string]any
			if err := json.Unmarshal(data, &out); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if _, ok := out[tc.key]; !ok {
				t.Fatalf("expected json key %q in %s payload: %s", tc.key, tc.name, string(data))
			}
		})
	}
}
