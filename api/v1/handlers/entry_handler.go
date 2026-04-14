package handlers

import (
	"net/http"

	"github.com/glycoview/nightscout-api/auth"
	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/httpx"
	"github.com/glycoview/nightscout-api/model"
	"github.com/glycoview/nightscout-api/query"
	"github.com/glycoview/nightscout-api/store"
	"github.com/glycoview/nightscout-api/util"
	"github.com/go-chi/chi/v5"
)

func EntriesCurrent(dep deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := store.DefaultQuery()
		query.Limit = 1
		records, err := dep.Store.Search(r.Context(), "entries", query)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		util.WriteRecords(w, records, nil)
	}
}

func EntriesList(dep deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), "date")
		records, err := dep.Store.Search(r.Context(), "entries", query)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		util.WriteRecords(w, records, nil)
	}
}

func EntriesSpec(dep deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		spec := chi.URLParam(r, "spec")
		switch spec {
		case "current":
			EntriesCurrent(dep)(w, r, identity)
			return
		case "sgv", "mbg":
			query := query.ParseV1(r.URL.Query(), "date")
			query.Filters = append(query.Filters, store.Filter{Field: "type", Op: "eq", Value: spec})
			records, err := dep.Store.Search(r.Context(), "entries", query)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			util.WriteRecords(w, records, nil)
			return
		default:
			record, err := dep.Store.Get(r.Context(), "entries", spec)
			if err != nil {
				status, body := httpx.RequireRecord(err)
				httpx.WriteJSON(w, status, body)
				return
			}
			util.WriteRecords(w, []model.Record{record}, nil)
		}
	}
}

func EntriesCreate(dep deps.Dependencies, persist bool) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		var body any
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		preview := make([]map[string]any, 0)
		apply := func(doc map[string]any) error {
			doc = model.CloneMap(doc)
			if dateString, ok := model.StringField(doc, "dateString"); ok && dateString != "" {
				normalized, offset, err := model.ToUTCString(dateString)
				if err == nil {
					doc["dateString"] = normalized
					doc["utcOffset"] = offset
				}
			}
			if persist {
				record, _, err := dep.Store.Create(r.Context(), "entries", doc, identity.Name)
				if err == nil {
					preview = append(preview, record.ToMap(false))
				}
				return err
			}
			if _, ok := model.StringField(doc, "_id"); !ok {
				doc["_id"] = store.GenerateIdentifier()
			}
			if _, ok := model.StringField(doc, "identifier"); !ok {
				doc["identifier"] = doc["_id"]
			}
			preview = append(preview, doc)
			return nil
		}
		switch typed := body.(type) {
		case map[string]any:
			if err := apply(typed); err != nil {
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
				if err := apply(doc); err != nil {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
					return
				}
			}
		default:
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		status := http.StatusOK
		if !persist {
			status = http.StatusCreated
		}
		httpx.WriteJSON(w, status, preview)
	}
}

func EntriesDelete(dep deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), "date")
		if spec := chi.URLParam(r, "spec"); spec != "" && spec != "json" {
			query.Filters = append(query.Filters, store.Filter{Field: "type", Op: "eq", Value: spec})
		}
		deleted, err := dep.Store.DeleteMatching(r.Context(), "entries", query, true, identity.Name)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
	}
}
