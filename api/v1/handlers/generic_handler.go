package handlers

import (
	"net/http"

	"github.com/glycoview/nightscout-api/auth"
	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/httpx"
	"github.com/glycoview/nightscout-api/model"
	"github.com/glycoview/nightscout-api/query"
	"github.com/glycoview/nightscout-api/util"
)

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
