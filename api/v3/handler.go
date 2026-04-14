package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/glycoview/nightscout-api/auth"
	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/httpx"
	"github.com/glycoview/nightscout-api/model"
	"github.com/glycoview/nightscout-api/query"
	"github.com/glycoview/nightscout-api/store"
	"github.com/go-chi/chi/v5"
)

// Version reports the application and API version metadata for API v3.
func Version(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"status": http.StatusOK,
			"result": map[string]any{
				"version":    dep.Config.AppVersion,
				"apiVersion": "3.0.4",
				"srvDate":    time.Now().UnixMilli(),
			},
		})
	}
}

// Status returns a lightweight authenticated health payload for API v3.
func Status(_ deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"status": http.StatusOK,
			"result": map[string]any{
				"srvDate": time.Now().UnixMilli(),
			},
		})
	}
}

// Test is a minimal authenticated endpoint used for API v3 connectivity checks.
func Test(_ deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": http.StatusOK, "result": "ok"})
	}
}

// LastModified returns the newest modification timestamp for each known
// collection.
func LastModified(dep deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		collections := []string{"entries", "treatments", "devicestatus", "profile", "food", "settings"}
		lastModified, err := dep.Store.LastModified(r.Context(), collections)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"status": http.StatusOK,
			"result": map[string]any{
				"srvDate":     time.Now().UnixMilli(),
				"collections": lastModified,
			},
		})
	}
}

// Search implements collection search for API v3.
func Search(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		if !validCollection(collection) {
			httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"status": http.StatusNotFound})
			return
		}
		permission := fmt.Sprintf("api:%s:read", collection)
		dep.Auth.Require(permission, false, func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
			parsed, err := query.ParseV3(r.URL.Query())
			if err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
				return
			}
			maxLimit := dep.Config.API3MaxLimit
			if maxLimit <= 0 {
				maxLimit = 1000
			}
			if parsed.Limit > maxLimit {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": "Parameter limit out of tolerance"})
				return
			}
			if parsed.Limit == 0 {
				parsed.Limit = maxLimit
			}
			records, err := dep.Store.Search(r.Context(), collection, parsed)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": http.StatusInternalServerError, "message": err.Error()})
				return
			}
			result := make([]map[string]any, 0, len(records))
			for _, record := range records {
				result = append(result, selectV3Fields(record, parsed.Fields))
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": http.StatusOK, "result": result})
		}).ServeHTTP(w, r)
	}
}

// Create creates a document in an API v3 collection.
func Create(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		if !validCollection(collection) {
			httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"status": http.StatusNotFound})
			return
		}
		identity, err := dep.Auth.AuthenticateExplicit(r)
		if err != nil {
			httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{"status": http.StatusUnauthorized, "message": authErrorMessage(err)})
			return
		}
		var doc map[string]any
		if err := httpx.ReadJSON(r, &doc); err != nil || len(doc) == 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": "Bad or missing request body"})
			return
		}
		doc, err = normalizeV3Document(collection, doc)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
			return
		}
		duplicate, found, err := findCreateDuplicate(r, dep, collection, doc)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
		permission := fmt.Sprintf("api:%s:create", collection)
		if found {
			permission = fmt.Sprintf("api:%s:update", collection)
		}
		if !dep.Auth.HasPermission(*identity, permission) {
			httpx.WriteJSON(w, http.StatusForbidden, map[string]any{"status": http.StatusForbidden, "message": fmt.Sprintf("Missing permission %s", permission)})
			return
		}
		record, created, err := dep.Store.Create(r.Context(), collection, doc, identity.Name)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
			return
		}
		status := http.StatusCreated
		body := map[string]any{
			"status":       status,
			"identifier":   record.Identifier(),
			"lastModified": record.SrvModified,
		}
		if !created {
			status = http.StatusOK
			body["status"] = status
			body["isDeduplication"] = true
			if found && duplicate.Identifier() != "" && duplicate.Identifier() != record.Identifier() {
				body["deduplicatedIdentifier"] = duplicate.Identifier()
			}
		}
		httpx.LastModifiedHeader(w, record.SrvModified)
		w.Header().Set("Location", fmt.Sprintf("/api/v3/%s/%s", model.NormalizeCollection(collection), record.Identifier()))
		httpx.WriteJSON(w, status, body)
	}
}

// Read returns a single document from an API v3 collection.
func Read(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		if !validCollection(collection) {
			httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"status": http.StatusNotFound})
			return
		}
		permission := fmt.Sprintf("api:%s:read", collection)
		dep.Auth.Require(permission, false, func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
			record, err := dep.Store.Get(r.Context(), collection, chi.URLParam(r, "identifier"))
			if err != nil {
				status, body := httpx.RequireRecord(err)
				httpx.WriteJSON(w, status, body)
				return
			}
			if modifiedSince, ok := httpx.ParseIfModifiedSince(r); ok && time.UnixMilli(record.SrvModified).Before(modifiedSince.Add(time.Second)) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			httpx.LastModifiedHeader(w, record.SrvModified)
			parsed, _ := query.ParseV3(r.URL.Query())
			fields := parsed.Fields
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": http.StatusOK, "result": selectV3Fields(record, fields)})
		}).ServeHTTP(w, r)
	}
}

// Update replaces a document in an API v3 collection and supports upsert
// semantics when allowed by the caller's permissions.
func Update(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		if !validCollection(collection) {
			httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"status": http.StatusNotFound})
			return
		}
		permission := fmt.Sprintf("api:%s:update", collection)
		dep.Auth.Require(permission, false, func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
			var doc map[string]any
			if err := httpx.ReadJSON(r, &doc); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
				return
			}
			identifier := chi.URLParam(r, "identifier")
			existing, err := dep.Store.Get(r.Context(), collection, identifier)
			if err == store.ErrGone {
				httpx.WriteJSON(w, http.StatusGone, map[string]any{"status": http.StatusGone})
				return
			}
			if err != nil && err != store.ErrNotFound {
				status, body := httpx.RequireRecord(err)
				httpx.WriteJSON(w, status, body)
				return
			}
			if err == store.ErrNotFound && !dep.Auth.HasPermission(*identity, fmt.Sprintf("api:%s:create", collection)) {
				httpx.WriteJSON(w, http.StatusForbidden, map[string]any{"status": http.StatusForbidden, "message": fmt.Sprintf("Missing permission api:%s:create", collection)})
				return
			}
			if err == nil {
				if unmodifiedSince, ok := httpx.ParseIfUnmodifiedSince(r); ok && time.UnixMilli(existing.SrvModified).After(unmodifiedSince) {
					httpx.WriteJSON(w, http.StatusPreconditionFailed, map[string]any{"status": http.StatusPreconditionFailed})
					return
				}
				if err := validateReplaceDoc(collection, existing, doc); err != nil {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
					return
				}
			} else {
				if err := validateCreateDoc(collection, doc); err != nil {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
					return
				}
			}
			record, created, err := dep.Store.Replace(r.Context(), collection, identifier, doc, identity.Name)
			if err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
				return
			}
			httpx.LastModifiedHeader(w, record.SrvModified)
			status := http.StatusOK
			body := map[string]any{"status": http.StatusOK, "result": selectV3Fields(record, nil)}
			if created {
				status = http.StatusCreated
				body["status"] = status
				body["identifier"] = record.Identifier()
				body["lastModified"] = record.SrvModified
			}
			httpx.WriteJSON(w, status, body)
		}).ServeHTTP(w, r)
	}
}

// Patch applies a partial update to an API v3 document.
func Patch(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		if !validCollection(collection) {
			httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"status": http.StatusNotFound})
			return
		}
		permission := fmt.Sprintf("api:%s:update", collection)
		dep.Auth.Require(permission, false, func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
			var doc map[string]any
			if err := httpx.ReadJSON(r, &doc); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
				return
			}
			existing, err := dep.Store.Get(r.Context(), collection, chi.URLParam(r, "identifier"))
			if err != nil {
				status, body := httpx.RequireRecord(err)
				httpx.WriteJSON(w, status, body)
				return
			}
			if unmodifiedSince, ok := httpx.ParseIfUnmodifiedSince(r); ok && time.UnixMilli(existing.SrvModified).After(unmodifiedSince) {
				httpx.WriteJSON(w, http.StatusPreconditionFailed, map[string]any{"status": http.StatusPreconditionFailed})
				return
			}
			if err := validatePatchDoc(existing, doc); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
				return
			}
			record, err := dep.Store.Patch(r.Context(), collection, chi.URLParam(r, "identifier"), doc, identity.Name)
			if err != nil {
				status, body := httpx.RequireRecord(err)
				httpx.WriteJSON(w, status, body)
				return
			}
			httpx.LastModifiedHeader(w, record.SrvModified)
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": http.StatusOK, "result": selectV3Fields(record, nil)})
		}).ServeHTTP(w, r)
	}
}

// Remove soft-deletes or permanently deletes an API v3 document.
func Remove(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		if !validCollection(collection) {
			httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"status": http.StatusNotFound})
			return
		}
		permission := fmt.Sprintf("api:%s:delete", collection)
		dep.Auth.Require(permission, false, func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
			permanent := r.URL.Query().Get("permanent") == "true"
			err := dep.Store.Delete(r.Context(), collection, chi.URLParam(r, "identifier"), permanent, identity.Name)
			if err != nil {
				status, body := httpx.RequireRecord(err)
				httpx.WriteJSON(w, status, body)
				return
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": http.StatusOK})
		}).ServeHTTP(w, r)
	}
}

// History returns document revisions modified since the supplied timestamp.
func History(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection := chi.URLParam(r, "collection")
		if !validCollection(collection) {
			httpx.WriteJSON(w, http.StatusNotFound, map[string]any{"status": http.StatusNotFound})
			return
		}
		permission := fmt.Sprintf("api:%s:read", collection)
		dep.Auth.Require(permission, false, func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
			since, err := strconv.ParseInt(chi.URLParam(r, "since"), 10, 64)
			if err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
				return
			}
			limit := httpx.ParsePositiveInt(r.URL.Query().Get("limit"), 1000)
			records, err := dep.Store.History(r.Context(), collection, since, limit)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": http.StatusInternalServerError, "message": err.Error()})
				return
			}
			result := make([]map[string]any, 0, len(records))
			for _, record := range records {
				result = append(result, selectV3Fields(record, nil))
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": http.StatusOK, "result": result})
		}).ServeHTTP(w, r)
	}
}

func validCollection(collection string) bool {
	switch model.NormalizeCollection(collection) {
	case "entries", "treatments", "devicestatus", "profile", "food", "settings":
		return true
	default:
		return false
	}
}

func validateCreateDoc(collection string, doc map[string]any) error {
	collection = model.NormalizeCollection(collection)
	if _, ok := model.Int64Field(doc, "date"); !ok && collection != "settings" {
		return fmt.Errorf("Bad or missing date field")
	}
	if date, ok := model.Int64Field(doc, "date"); ok && date <= 100000000000 {
		return fmt.Errorf("Bad or missing date field")
	}
	if utcOffset, exists := doc["utcOffset"]; exists {
		if value, err := toInt64Compat(utcOffset); err != nil || value < -1440 || value > 1440 {
			return fmt.Errorf("Bad or missing utcOffset field")
		}
	}
	if collection != "settings" {
		if app, ok := model.StringField(doc, "app"); !ok || strings.TrimSpace(app) == "" {
			return fmt.Errorf("Bad or missing app field")
		}
	}
	return nil
}

func validateReplaceDoc(_ string, existing model.Record, doc map[string]any) error {
	immutable := []string{"date", "utcOffset", "eventType", "device", "app", "srvCreated", "subject", "srvModified", "modifiedBy", "isValid"}
	for _, field := range immutable {
		if proposed, exists := doc[field]; exists {
			current := model.PathValue(existing.ToMap(true), field)
			if field == "identifier" {
				continue
			}
			if fmt.Sprint(proposed) != fmt.Sprint(current) {
				return fmt.Errorf("Field %s cannot be modified by the client", field)
			}
		}
	}
	if identifier, ok := model.StringField(doc, "identifier"); ok && identifier != "" && identifier != existing.Identifier() {
		delete(doc, "identifier")
	}
	return nil
}

func validatePatchDoc(existing model.Record, doc map[string]any) error {
	immutable := []string{"identifier", "date", "utcOffset", "eventType", "device", "app", "srvCreated", "subject", "srvModified", "modifiedBy", "isValid"}
	for _, field := range immutable {
		if proposed, exists := doc[field]; exists {
			current := model.PathValue(existing.ToMap(true), field)
			if fmt.Sprint(proposed) != fmt.Sprint(current) {
				return fmt.Errorf("Field %s cannot be modified by the client", field)
			}
		}
	}
	return nil
}

func normalizeV3Document(collection string, doc map[string]any) (map[string]any, error) {
	clean, err := store.NormalizeData(collection, doc)
	if err != nil {
		switch err.Error() {
		case "bad or missing date field":
			return nil, fmt.Errorf("Bad or missing date field")
		case "bad or missing utcOffset field":
			return nil, fmt.Errorf("Bad or missing utcOffset field")
		case "bad created_at field":
			return nil, fmt.Errorf("Bad created_at field")
		default:
			return nil, err
		}
	}
	if identifier, ok := model.StringField(clean, "identifier"); !ok || strings.TrimSpace(identifier) == "" {
		if calculated := store.CalculateIdentifier(clean); calculated != "" {
			clean["identifier"] = calculated
			clean["_id"] = calculated
		}
	}
	if err := validateCreateDoc(collection, clean); err != nil {
		return nil, err
	}
	return clean, nil
}

func findCreateDuplicate(r *http.Request, dep deps.Dependencies, collection string, doc map[string]any) (model.Record, bool, error) {
	if identifier, ok := model.StringField(doc, "identifier"); ok && strings.TrimSpace(identifier) != "" {
		record, found, err := searchDuplicateByFilters(r, dep, collection, []store.Filter{{Field: "identifier", Op: "eq", Value: identifier}})
		if err != nil || found {
			return record, found, err
		}
	}
	var filters []store.Filter
	switch model.NormalizeCollection(collection) {
	case "entries":
		date, _ := model.Int64Field(doc, "date")
		kind, _ := model.StringField(doc, "type")
		filters = append(filters,
			store.Filter{Field: "date", Op: "eq", Value: strconv.FormatInt(date, 10)},
			store.Filter{Field: "type", Op: "eq", Value: kind},
		)
	case "treatments":
		createdAt, _ := model.StringField(doc, "created_at")
		eventType, _ := model.StringField(doc, "eventType")
		filters = append(filters,
			store.Filter{Field: "created_at", Op: "eq", Value: createdAt},
			store.Filter{Field: "eventType", Op: "eq", Value: eventType},
		)
	case "devicestatus":
		createdAt, _ := model.StringField(doc, "created_at")
		device, _ := model.StringField(doc, "device")
		filters = append(filters,
			store.Filter{Field: "created_at", Op: "eq", Value: createdAt},
			store.Filter{Field: "device", Op: "eq", Value: device},
		)
	case "food", "profile":
		createdAt, _ := model.StringField(doc, "created_at")
		filters = append(filters, store.Filter{Field: "created_at", Op: "eq", Value: createdAt})
	}
	if len(filters) == 0 {
		return model.Record{}, false, nil
	}
	return searchDuplicateByFilters(r, dep, collection, filters)
}

func searchDuplicateByFilters(r *http.Request, dep deps.Dependencies, collection string, filters []store.Filter) (model.Record, bool, error) {
	records, err := dep.Store.Search(r.Context(), collection, store.Query{
		Filters:        filters,
		Limit:          1,
		SortField:      "date",
		SortDesc:       true,
		IncludeDeleted: true,
	})
	if err != nil {
		return model.Record{}, false, err
	}
	if len(records) == 0 {
		return model.Record{}, false, nil
	}
	return records[0], true, nil
}

func authErrorMessage(err error) string {
	switch {
	case err == auth.ErrBadToken, err == auth.ErrBadJWT:
		return "Bad access token or JWT"
	default:
		return "Missing or bad access token or JWT"
	}
}

func selectV3Fields(record model.Record, fields []string) map[string]any {
	selected := store.SelectFields(record, fields)
	delete(selected, "_id")
	return selected
}

func toInt64Compat(value any) (int64, error) {
	switch typed := value.(type) {
	case int:
		return int64(typed), nil
	case int64:
		return typed, nil
	case float64:
		return int64(typed), nil
	case json.Number:
		return typed.Int64()
	case string:
		return strconv.ParseInt(typed, 10, 64)
	default:
		return 0, fmt.Errorf("not int")
	}
}
