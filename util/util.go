package util

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/glycoview/nightscout-api/httpx"
	"github.com/glycoview/nightscout-api/model"
	"github.com/glycoview/nightscout-api/store"
	"github.com/go-chi/chi/v5"
)

// StripImplicitDateFilters removes the default rolling-window filters injected
// by ParseV1 so slice/times style endpoints can search across larger ranges.
func StripImplicitDateFilters(filters []store.Filter) []store.Filter {
	out := make([]store.Filter, 0, len(filters))
	for _, filter := range filters {
		if filter.Field == "date" && filter.Op == "gte" && strings.Contains(filter.Value, "T") {
			continue
		}
		if filter.Field == "created_at" && filter.Op == "gte" && strings.Contains(filter.Value, "T") {
			continue
		}
		out = append(out, filter)
	}
	return out
}

// WriteRecords serializes records using the public Nightscout response shape.
func WriteRecords(w http.ResponseWriter, records []model.Record, fields []string) {
	response := make([]map[string]any, 0, len(records))
	for _, record := range records {
		if len(fields) == 0 {
			response = append(response, record.ToMap(false))
		} else {
			response = append(response, store.SelectFields(record, fields))
		}
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

// ParseTimesWildcard extracts the `prefix/expr` wildcard payload used by the
// v1 /times endpoints.
func ParseTimesWildcard(r *http.Request) (prefix, expr string, ok bool) {
	wild := chi.URLParam(r, "*")
	parts := strings.SplitN(wild, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	prefix = parts[0]
	expr = strings.TrimSuffix(parts[1], ".json")
	if decoded, err := url.PathUnescape(prefix); err == nil {
		prefix = decoded
	}
	if decoded, err := url.PathUnescape(expr); err == nil {
		expr = decoded
	}
	if prefix == "" || expr == "" {
		return "", "", false
	}
	return prefix, expr, true
}
