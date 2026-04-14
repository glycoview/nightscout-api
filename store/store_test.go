package store

import (
	"testing"
	"time"

	"github.com/glycoview/nightscout-api/model"
)

func TestNormalizeDataAndHelpers(t *testing.T) {
	clean, err := NormalizeData("treatments", map[string]any{
		"created_at": "2024-01-02T03:04:05.000+02:00",
		"carbs":      "30",
		"insulin":    "2.00",
		"glucose":    "100",
		"notes":      "<IMG SRC='x'>",
		"device":     "pump",
		"app":        "test",
		"eventType":  "Meal Bolus",
	})
	if err != nil {
		t.Fatalf("NormalizeData failed: %v", err)
	}
	if clean["created_at"] != "2024-01-02T01:04:05.000Z" || clean["utcOffset"] != 120 || clean["notes"] != "<img>" {
		t.Fatalf("NormalizeData mismatch: %#v", clean)
	}
	if clean["carbs"] != int64(30) || clean["insulin"] != float64(2) && clean["insulin"] != int64(2) {
		t.Fatalf("NormalizeData should normalize numeric values: %#v", clean)
	}

	if _, err := NormalizeData("entries", map[string]any{"date": "abc"}); err == nil {
		t.Fatalf("expected bad date error")
	}
	if CalculateIdentifier(map[string]any{"device": "pump", "date": int64(1)}) == "" {
		t.Fatalf("expected calculated identifier")
	}
	if GenerateIdentifier() == "" {
		t.Fatalf("expected generated identifier")
	}
	if DedupeKey("entries", map[string]any{"date": int64(1), "type": "sgv"}) == "" {
		t.Fatalf("expected dedupe key")
	}
}

func TestApplyQueryAndFieldSelection(t *testing.T) {
	records := []model.Record{
		{ID: "1", Data: map[string]any{"identifier": "1", "date": int64(2), "sgv": 100, "device": "a"}, IsValid: true},
		{ID: "2", Data: map[string]any{"identifier": "2", "date": int64(1), "sgv": 80, "device": "b"}, IsValid: false},
	}
	q := Query{
		Filters:   []Filter{{Field: "sgv", Op: "gte", Value: "90"}},
		SortField: "date",
		SortDesc:  true,
		Limit:     10,
	}
	filtered := ApplyQuery(records, q)
	if len(filtered) != 1 || filtered[0].Identifier() != "1" {
		t.Fatalf("ApplyQuery mismatch: %#v", filtered)
	}
	selected := SelectFields(records[0], []string{"device"})
	if len(selected) != 1 || selected["device"] != "a" {
		t.Fatalf("SelectFields mismatch: %#v", selected)
	}
}

func TestFilterAndCompareHelpers(t *testing.T) {
	record := model.Record{Data: map[string]any{"sgv": 100, "device": "pump", "created_at": "2024-01-02T00:00:00.000Z"}, IsValid: true}
	if !matchesAll(record, []Filter{{Field: "sgv", Op: "gte", Value: "90"}}) {
		t.Fatalf("matchesAll mismatch")
	}
	for _, tc := range []struct {
		value  any
		filter Filter
		want   bool
	}{
		{100, Filter{Op: "eq", Value: "100"}, true},
		{100, Filter{Op: "ne", Value: "80"}, true},
		{100, Filter{Op: "in", Value: "10|100"}, true},
		{"pump", Filter{Op: "re", Value: "^pu"}, true},
		{"pump", Filter{Op: "nin", Value: "pen|pod"}, true},
		{"2024-01-02T00:00:00.000Z", Filter{Op: "gte", Value: "2024-01-01"}, true},
		{"pump", Filter{Op: "exists", Value: "true"}, true},
	} {
		if got := matchFilter(tc.value, tc.filter); got != tc.want {
			t.Fatalf("matchFilter mismatch for %#v %#v: got %v", tc.value, tc.filter, got)
		}
	}
	if !CompareValues(int64(2), int64(1), true) || !CompareValues("b", "a", true) {
		t.Fatalf("CompareValues mismatch")
	}
}

func TestConversionHelpers(t *testing.T) {
	if _, err := toFloat64("1.5"); err != nil {
		t.Fatalf("toFloat64 failed: %v", err)
	}
	if _, err := toInt64("7"); err != nil {
		t.Fatalf("toInt64 failed: %v", err)
	}
	if _, err := toTime("2024-01-02"); err != nil {
		t.Fatalf("toTime failed: %v", err)
	}
	if _, err := toInt64(int32(7)); err != nil {
		t.Fatalf("toInt64 int32 failed: %v", err)
	}
	if !isRegexLiteral("/abc/i") {
		t.Fatalf("isRegexLiteral mismatch")
	}
	if regex, err := compileRegexFilter("/abc/i"); err != nil || !regex.MatchString("ABC") {
		t.Fatalf("compileRegexFilter mismatch: %v", err)
	}
	if !inStringSet("a", "a|b") || !inNumericSet(2, "1|2") {
		t.Fatalf("set helper mismatch")
	}
	if sanitizeNotes("<IMG src='x'>") != "<img>" {
		t.Fatalf("sanitizeNotes mismatch")
	}
	if millis, offset, err := parseDateValue("2024-01-02T03:04:05.000+02:00"); err != nil || millis == 0 || *offset != 120 {
		t.Fatalf("parseDateValue mismatch: %d %v %v", millis, offset, err)
	}
	if _, _, err := parseDateValue(int64(1)); err != nil {
		t.Fatalf("parseDateValue int64 failed: %v", err)
	}
}

func TestDefaultQuery(t *testing.T) {
	q := DefaultQuery()
	if q.Limit != 10 || q.SortField != "date" || !q.SortDesc {
		t.Fatalf("DefaultQuery mismatch: %#v", q)
	}
}

func TestToTimeWithMillis(t *testing.T) {
	now := time.Now().UnixMilli()
	got, err := toTime(now)
	if err != nil || got.UnixMilli() != now {
		t.Fatalf("toTime millis mismatch: %v %v", got, err)
	}
}
