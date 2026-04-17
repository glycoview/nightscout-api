package handlers

import (
	"net/http"
	"time"

	"github.com/glycoview/nightscout-api/auth"
	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/httpx"
	"github.com/glycoview/nightscout-api/model"
	"github.com/glycoview/nightscout-api/query"
	"github.com/glycoview/nightscout-api/util"
)

// ensureV1Timestamp fills in date/created_at when missing. Nightscout v1
// clients (Trio, xDrip, etc.) routinely POST treatments, devicestatus, and
// food without a timestamp and expect the server to default to now.
func ensureV1Timestamp(doc map[string]any) {
	if doc == nil {
		return
	}
	hasDate := false
	if _, ok := doc["date"]; ok && doc["date"] != nil {
		hasDate = true
	}
	if value, ok := model.StringField(doc, "created_at"); ok && value != "" {
		hasDate = true
	}
	if hasDate {
		return
	}
	now := time.Now().UTC()
	doc["created_at"] = now.Format("2006-01-02T15:04:05.000Z")
	doc["date"] = now.UnixMilli()
}

// GenericCollectionList lists records for a collection that follows the common
// Nightscout list conventions.
func GenericCollectionList(dep deps.Dependencies, collection, defaultDateField string) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), defaultDateField)
		records, err := dep.Store.Search(r.Context(), collection, query)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		util.WriteRecords(w, records, nil)
	}
}

// GenericCollectionCreate creates one or more records in a generic collection.
func GenericCollectionCreate(dep deps.Dependencies, collection string) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		var body any
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		var created []model.Record
		switch typed := body.(type) {
		case map[string]any:
			ensureV1Timestamp(typed)
			record, _, err := dep.Store.Create(r.Context(), collection, typed, identity.Name)
			if err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
				return
			}
			created = append(created, record)
		case []any:
			for _, item := range typed {
				doc, ok := item.(map[string]any)
				if !ok {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
					return
				}
				ensureV1Timestamp(doc)
				record, _, err := dep.Store.Create(r.Context(), collection, doc, identity.Name)
				if err != nil {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
					return
				}
				created = append(created, record)
			}
		default:
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		util.WriteRecords(w, created, nil)
	}
}

// GenericCollectionDelete deletes records from a generic collection using the
// v1 query syntax.
func GenericCollectionDelete(dep deps.Dependencies, collection, defaultDateField string) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), defaultDateField)
		deleted, err := dep.Store.DeleteMatching(r.Context(), collection, query, true, identity.Name)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
	}
}
