package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRecordHelpers(t *testing.T) {
	deletedAt := time.Now().UnixMilli()
	record := Record{
		ID:          "fallback-id",
		Data:        map[string]any{"a": 1, "_id": "data-id", "nested": map[string]any{"x": "y"}},
		SrvCreated:  1,
		SrvModified: 2,
		Subject:     "tester",
		IsValid:     true,
		DeletedAt:   &deletedAt,
	}

	clone := record.Clone()
	clone.Data["a"] = 2
	if record.Data["a"] == 2 {
		t.Fatalf("Clone should deep-copy data")
	}
	if record.Identifier() != "data-id" {
		t.Fatalf("unexpected identifier: %s", record.Identifier())
	}
	withData := record.WithData(map[string]any{"b": 2})
	if _, ok := withData.Data["b"]; !ok {
		t.Fatalf("WithData should replace data")
	}
	full := record.ToMap(true)
	if full["srvCreated"] != int64(1) || full["deletedAt"] != deletedAt {
		t.Fatalf("ToMap(includeMeta) mismatch: %#v", full)
	}
}

func TestMapAndFieldHelpers(t *testing.T) {
	src := map[string]any{
		"stringer": json.Number("12"),
		"child":    map[string]any{"value": "x"},
		"list":     []any{map[string]any{"a": "b"}},
	}
	cloned := CloneMap(src)
	merged := Merge(cloned, map[string]any{"extra": true})
	if merged["extra"] != true {
		t.Fatalf("Merge failed")
	}

	if got, ok := StringField(map[string]any{"v": json.Number("5")}, "v"); !ok || got != "5" {
		t.Fatalf("StringField mismatch")
	}
	if got, ok := Int64Field(map[string]any{"v": "7"}, "v"); !ok || got != 7 {
		t.Fatalf("Int64Field mismatch")
	}
	if got, ok := BoolField(map[string]any{"v": "true"}, "v"); !ok || !got {
		t.Fatalf("BoolField mismatch")
	}
}

func TestTimeAndPathHelpers(t *testing.T) {
	got, offset, err := ToUTCString("2024-01-02T03:04:05.000+02:00")
	if err != nil || got != "2024-01-02T01:04:05.000Z" || offset != 120 {
		t.Fatalf("ToUTCString mismatch: got=%s offset=%d err=%v", got, offset, err)
	}
	if _, _, err := ToUTCString("2024-01-02T03:04:05+0200"); err != nil {
		t.Fatalf("ToUTCString alt layout failed: %v", err)
	}
	if NormalizeCollection("device_status") != "devicestatus" || NormalizeCollection("foods") != "food" {
		t.Fatalf("NormalizeCollection mismatch")
	}

	data := map[string]any{
		"a": map[string]any{"b": "c"},
		"d": []any{},
		"e": map[string]any{"f": map[string]any{"g": 1}},
	}
	if PathValue(data, "a.b") != "c" || PathValue(map[string]any{"arr": map[string]any{"0": "x"}}, "arr[0]") != "x" {
		t.Fatalf("PathValue mismatch")
	}
	if len(splitPath("x[y]")) != 2 {
		t.Fatalf("splitPath mismatch")
	}
}
