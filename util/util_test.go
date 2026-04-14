package util

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glycoview/nightscout-api/model"
	"github.com/glycoview/nightscout-api/store"
	"github.com/go-chi/chi/v5"
)

func TestStripImplicitDateFilters(t *testing.T) {
	filters := []store.Filter{
		{Field: "date", Op: "gte", Value: "2024-01-01T00:00:00Z"},
		{Field: "created_at", Op: "gte", Value: "2024-01-01T00:00:00Z"},
		{Field: "sgv", Op: "gte", Value: "100"},
	}
	out := StripImplicitDateFilters(filters)
	if len(out) != 1 || out[0].Field != "sgv" {
		t.Fatalf("StripImplicitDateFilters mismatch: %#v", out)
	}
}

func TestWriteRecords(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteRecords(rec, []model.Record{{ID: "1", Data: map[string]any{"identifier": "1", "device": "pump"}, IsValid: true}}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}
}

func TestParseTimesWildcard(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/times/2014-07/T09:.json", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("*", "2014-07/T09:.json")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	req = req.WithContext(ctx)

	prefix, expr, ok := ParseTimesWildcard(req)
	if !ok || prefix != "2014-07" || expr != "T09:" {
		t.Fatalf("ParseTimesWildcard mismatch: %q %q %v", prefix, expr, ok)
	}
}
