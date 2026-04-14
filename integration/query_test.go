package integration

import (
	"net/url"
	"testing"
	"time"

	"github.com/glycoview/nightscout-api/query"
)

func TestParseV1DefaultDateWindow(t *testing.T) {
	before := time.Now().UTC().Add(-97 * time.Hour)
	after := time.Now().UTC().Add(-95 * time.Hour)

	parsed := query.ParseV1(url.Values{}, "date")
	if len(parsed.Filters) != 1 {
		t.Fatalf("expected one implicit filter, got %d", len(parsed.Filters))
	}

	filter := parsed.Filters[0]
	mustEqual(t, filter.Field, "date", "implicit date field")
	mustEqual(t, filter.Op, "gte", "implicit date op")

	got, err := time.Parse(time.RFC3339, filter.Value)
	if err != nil {
		t.Fatalf("parse implicit date: %v", err)
	}
	if !got.After(before) || !got.Before(after) {
		t.Fatalf("expected implicit date around 4 days ago, got %s", got)
	}
}

func TestParseV1DoesNotAddDateFilterWhenIdentifierPresent(t *testing.T) {
	values := url.Values{}
	values.Set("find[_id]", "1234")

	parsed := query.ParseV1(values, "date")
	if len(parsed.Filters) != 1 {
		t.Fatalf("expected only explicit id filter, got %d filters", len(parsed.Filters))
	}
	mustEqual(t, parsed.Filters[0].Field, "_id", "filter field")
}

func TestParseV3LeavesLimitUnsetByDefault(t *testing.T) {
	parsed, err := query.ParseV3(url.Values{})
	if err != nil {
		t.Fatalf("parse v3 query: %v", err)
	}
	if parsed.Limit != 0 {
		t.Fatalf("expected default limit 0 for API3, got %d", parsed.Limit)
	}
}
