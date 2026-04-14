package handlers

import (
	"net/http"

	"github.com/glycoview/nightscout-api/auth"
	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/httpx"
	"github.com/glycoview/nightscout-api/query"
	"github.com/glycoview/nightscout-api/store"
	"github.com/glycoview/nightscout-api/util"
)

// TreatmentsList lists treatments using the Nightscout v1 query format.
func TreatmentsList(dep deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), "created_at")
		records, err := dep.Store.Search(r.Context(), "treatments", query)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		util.WriteRecords(w, records, nil)
	}
}

// TreatmentsCreate creates one or more treatments.
func TreatmentsCreate(dep deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		var body any
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		switch typed := body.(type) {
		case map[string]any:
			if _, _, err := dep.Store.Create(r.Context(), "treatments", typed, identity.Name); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
				return
			}
		case []any:
			for _, item := range typed {
				doc, ok := item.(map[string]any)
				if !ok {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
					return
				}
				if _, _, err := dep.Store.Create(r.Context(), "treatments", doc, identity.Name); err != nil {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
					return
				}
			}
		default:
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		records, _ := dep.Store.Search(r.Context(), "treatments", store.DefaultQuery())
		util.WriteRecords(w, records, nil)
	}
}

// TreatmentsDelete deletes treatments matching the supplied v1 query.
func TreatmentsDelete(dep deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), "created_at")
		deleted, err := dep.Store.DeleteMatching(r.Context(), "treatments", query, true, identity.Name)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
	}
}
