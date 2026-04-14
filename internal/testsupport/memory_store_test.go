package testsupport

import (
	"context"
	"testing"
	"time"

	"github.com/glycoview/nightscout-api/store"
)

func TestMemoryStoreHelpers(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	record, _, err := s.Create(ctx, "entries", map[string]any{
		"date":       time.Now().UnixMilli(),
		"dateString": "2024-01-01T00:00:00",
		"device":     "sensor",
	}, "seed")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if recordIdentifier(map[string]any{"identifier": "x"}) != "x" {
		t.Fatalf("recordIdentifier mismatch")
	}
	if withDefaultQuery(store.Query{}).Limit == 0 {
		t.Fatalf("withDefaultQuery mismatch")
	}
	if _, err := s.Get(ctx, "entries", record.Identifier()); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
}
