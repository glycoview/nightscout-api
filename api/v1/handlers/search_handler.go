package handlers

import (
	"net/http"

	"github.com/glycoview/nightscout-api/auth"
	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/httpx"
	"github.com/glycoview/nightscout-api/query"
	"github.com/glycoview/nightscout-api/util"
	"github.com/go-chi/chi/v5"
)

func Versions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, []map[string]string{
			{"url": "/api/v1", "version": "1.0.0"},
			{"url": "/api/v2", "version": "2.0.0"},
			{"url": "/api/v3", "version": "3.0.4"},
		})
	}
}

func EchoRoute(_ deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		httpx.WriteJSON(w, http.StatusOK, query.EchoV1(chi.URLParam(r, "collection"), r.URL.Query()))
	}
}

func SliceRoute(dep deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		collection := chi.URLParam(r, "collection")
		q := query.ParseV1(r.URL.Query(), "date")
		q.Filters = util.StripImplicitDateFilters(q.Filters)
		q.Limit = max(q.Limit, 1000)
		records, err := dep.Store.Search(r.Context(), collection, q)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		filtered := query.Slice(records, chi.URLParam(r, "field"), chi.URLParam(r, "type"), chi.URLParam(r, "prefix"), q.Limit)
		util.WriteRecords(w, filtered, nil)
	}
}

func TimesEchoRoute(_ deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		prefix, expr, ok := util.ParseTimesWildcard(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, patterns := query.Times(nil, prefix, expr, 0)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"req":     map[string]any{"query": r.URL.Query()},
			"pattern": patterns,
		})
	}
}

func TimesRoute(dep deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		prefix, expr, ok := util.ParseTimesWildcard(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		q := query.ParseV1(r.URL.Query(), "date")
		q.Filters = util.StripImplicitDateFilters(q.Filters)
		q.Limit = max(q.Limit, 1000)
		records, err := dep.Store.Search(r.Context(), "entries", q)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		filtered, _ := query.Times(records, prefix, expr, q.Limit)
		util.WriteRecords(w, filtered, nil)
	}
}
