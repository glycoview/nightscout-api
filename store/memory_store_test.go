package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/glycoview/nightscout-api/internal/testsupport"
	"github.com/glycoview/nightscout-api/store"
)

func TestMemoryStoreCRUD(t *testing.T) {
	s := testsupport.NewMemoryStore()
	ctx := context.Background()
	doc := map[string]any{
		"date":      time.Now().UnixMilli(),
		"app":       "test",
		"device":    "pump",
		"eventType": "Meal Bolus",
	}

	record, created, err := s.Create(ctx, "treatments", doc, "subject-a")
	if err != nil || !created {
		t.Fatalf("Create failed: created=%v err=%v", created, err)
	}
	if _, duplicate, err := s.Create(ctx, "treatments", doc, "subject-a"); err != nil || duplicate {
		t.Fatalf("expected dedupe existing record")
	}

	got, err := s.Get(ctx, "treatments", record.Identifier())
	if err != nil || got.Identifier() != record.Identifier() {
		t.Fatalf("Get failed: %#v %v", got, err)
	}

	list, err := s.Search(ctx, "treatments", store.Query{SortField: "date", SortDesc: true})
	if err != nil || len(list) != 1 {
		t.Fatalf("Search failed: %v len=%d", err, len(list))
	}

	replaced, wasCreated, err := s.Replace(ctx, "treatments", record.Identifier(), map[string]any{
		"date":      doc["date"],
		"app":       "test",
		"device":    "pump",
		"eventType": "Meal Bolus",
		"carbs":     12,
	}, "subject-b")
	if err != nil || wasCreated || replaced.Subject != "subject-b" {
		t.Fatalf("Replace failed: created=%v err=%v record=%#v", wasCreated, err, replaced)
	}

	patched, err := s.Patch(ctx, "treatments", record.Identifier(), map[string]any{"insulin": 1}, "subject-c")
	if err != nil || patched.Data["modifiedBy"] != "subject-c" {
		t.Fatalf("Patch failed: %#v err=%v", patched, err)
	}

	history, err := s.History(ctx, "treatments", 0, 10)
	if err != nil || len(history) < 3 {
		t.Fatalf("History failed: %v len=%d", err, len(history))
	}
	lastModified, err := s.LastModified(ctx, []string{"treatments"})
	if err != nil || lastModified["treatments"] == 0 {
		t.Fatalf("LastModified failed: %v %#v", err, lastModified)
	}
}

func TestMemoryStoreDeleteModes(t *testing.T) {
	s := testsupport.NewMemoryStore()
	ctx := context.Background()
	record, _, err := s.Create(ctx, "entries", map[string]any{
		"date":       time.Now().UnixMilli(),
		"device":     "sensor",
		"dateString": "2024-01-01T00:00:00",
	}, "seed")
	if err != nil {
		t.Fatalf("seed Create failed: %v", err)
	}

	if err := s.Delete(ctx, "entries", record.Identifier(), false, "subject"); err != nil {
		t.Fatalf("soft Delete failed: %v", err)
	}
	if _, err := s.Get(ctx, "entries", record.Identifier()); err != store.ErrGone {
		t.Fatalf("expected ErrGone after soft delete, got %v", err)
	}

	record2, _, err := s.Create(ctx, "entries", map[string]any{
		"date":       time.Now().UnixMilli() + 1,
		"device":     "sensor-2",
		"dateString": "2024-01-01T00:05:00",
	}, "seed")
	if err != nil {
		t.Fatalf("seed Create2 failed: %v", err)
	}
	if err := s.Delete(ctx, "entries", record2.Identifier(), true, "subject"); err != nil {
		t.Fatalf("hard Delete failed: %v", err)
	}
	if _, err := s.Get(ctx, "entries", record2.Identifier()); err != store.ErrNotFound {
		t.Fatalf("expected ErrNotFound after hard delete, got %v", err)
	}
}

func TestMemoryStoreDeleteMatchingAndHelpers(t *testing.T) {
	s := testsupport.NewMemoryStore()
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, _, err := s.Create(ctx, "devicestatus", map[string]any{
			"date":       time.Now().UnixMilli() + int64(i),
			"created_at": "2024-01-01T00:00:00.000Z",
			"device":     "rig",
			"app":        "test",
		}, "seed"); err != nil {
			t.Fatalf("seed Create failed: %v", err)
		}
	}
	deleted, err := s.DeleteMatching(ctx, "devicestatus", store.Query{Filters: []store.Filter{{Field: "device", Op: "eq", Value: "rig"}}, SortField: "date", SortDesc: true}, false, "subject")
	if err != nil || deleted == 0 {
		t.Fatalf("DeleteMatching failed: deleted=%d err=%v", deleted, err)
	}
}
