package query

import (
	"net/url"
	"testing"
	"time"

	"github.com/glycoview/nightscout-api/model"
	"github.com/glycoview/nightscout-api/store"
)

func sampleRecord(date int64, dateString string, extra map[string]any) model.Record {
	data := map[string]any{
		"date":       date,
		"dateString": dateString,
		"type":       "sgv",
	}
	for k, v := range extra {
		data[k] = v
	}
	return model.Record{ID: "id", Data: data, IsValid: true}
}

func TestParseV1AndEchoV1(t *testing.T) {
	values := url.Values{}
	values.Set("count", "5")
	values.Set("find[sgv][$gte]", "120")
	values.Set("find", `{"device":{"$eq":"pump"}}`)

	q := ParseV1(values, "date")
	if q.Limit != 5 || len(q.Filters) < 2 {
		t.Fatalf("unexpected ParseV1 result: %#v", q)
	}

	echo := EchoV1("entries", values)
	if echo["storage"] != "entries" {
		t.Fatalf("unexpected echo payload: %#v", echo)
	}
}

func TestParseV3(t *testing.T) {
	values := url.Values{
		"limit":      []string{"3"},
		"skip":       []string{"1"},
		"sort$desc":  []string{"date"},
		"fields":     []string{"date,device"},
		"device$eq":  []string{"pump"},
		"created_at": []string{"2024-01-01"},
	}
	q, err := ParseV3(values)
	if err != nil {
		t.Fatalf("ParseV3 failed: %v", err)
	}
	if q.Limit != 3 || q.Skip != 1 || q.SortField != "date" || !q.SortDesc || len(q.Fields) != 2 || len(q.Filters) != 2 {
		t.Fatalf("unexpected ParseV3 result: %#v", q)
	}
	if _, err := ParseV3(url.Values{"sort": []string{"date"}, "sort$desc": []string{"x"}}); err == nil {
		t.Fatalf("expected ParseV3 to reject conflicting sort params")
	}
}

func TestSliceAndTimes(t *testing.T) {
	now := time.Now().UnixMilli()
	records := []model.Record{
		sampleRecord(now, "2014-07-19T09:00:00", map[string]any{"device": "a"}),
		sampleRecord(now-1, "2014-07-19T09:05:00", map[string]any{"device": "b"}),
		sampleRecord(now-2, "2014-07-19T10:00:00", map[string]any{"type": "mbg"}),
	}

	sliced := Slice(records, "dateString", "sgv", "2014-07-19", 2)
	if len(sliced) != 2 {
		t.Fatalf("expected 2 sliced records, got %d", len(sliced))
	}

	timed, patterns := Times(records, "2014-07", "T09:", 10)
	if len(timed) != 2 || len(patterns) == 0 {
		t.Fatalf("unexpected Times result: len=%d patterns=%v", len(timed), patterns)
	}
}

func TestExpandAndParseHelpers(t *testing.T) {
	if got := ExpandBraces("20{14..15}"); len(got) != 2 || got[0] != "2014" {
		t.Fatalf("ExpandBraces mismatch: %v", got)
	}
	field, op := parseBracketFilter("find[sgv][$gte]")
	if field != "sgv" || op != "gte" {
		t.Fatalf("parseBracketFilter mismatch")
	}
	field, op = parseV3Filter("date$lte")
	if field != "date" || op != "lte" {
		t.Fatalf("parseV3Filter mismatch")
	}
	filters := parseMongoFindJSON(`{"sgv":{"$gte":120}}`)
	if len(filters) != 1 || filters[0].Op != "gte" {
		t.Fatalf("parseMongoFindJSON mismatch: %#v", filters)
	}
	if !isReservedV3Key("limit") || isReservedV3Key("sgv") {
		t.Fatalf("isReservedV3Key mismatch")
	}
	if !hasDateFilter([]store.Filter{{Field: "created_at"}}) || !hasIDFilter([]store.Filter{{Field: "_id"}}) {
		t.Fatalf("date/id helper mismatch")
	}
	if got := expandBody("01..03"); len(got) != 3 || got[2] != "03" {
		t.Fatalf("expandBody mismatch: %v", got)
	}
	if !matchesAnyPrefix("2014-07-19T09:00", []string{"2014-07"}) {
		t.Fatalf("matchesAnyPrefix mismatch")
	}
	if !prefixPatternToRegexp("2014-07").MatchString("2014-07-19") {
		t.Fatalf("prefixPatternToRegexp mismatch")
	}
	if max(2, 5) != 5 {
		t.Fatalf("max mismatch")
	}
}
