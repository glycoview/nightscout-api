package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glycoview/nightscout-api/auth"
	"github.com/glycoview/nightscout-api/config"
	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/internal/testsupport"
	"github.com/glycoview/nightscout-api/model"
	"github.com/go-chi/chi/v5"
)

func testDepsV3() deps.Dependencies {
	return deps.Dependencies{
		Config: config.Config{APISecret: "secret", AppVersion: "1.2.3", DefaultRoles: []string{"denied"}, API3MaxLimit: 5}.WithDefaults(),
		Store:  testsupport.NewMemoryStore(),
		Auth:   testsupport.NewAuthManager("secret", []string{"denied"}),
	}
}

func withParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.RouteContext(req.Context())
	if routeCtx == nil {
		routeCtx = chi.NewRouteContext()
	}
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func decodeV3(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decodeV3: %v body=%s", err, rec.Body.String())
	}
	return out
}

func TestV3SimpleHandlersAndHelpers(t *testing.T) {
	dep := testDepsV3()
	mem := dep.Store.(*testsupport.MemoryStore)
	_, _, _ = mem.Create(context.Background(), "entries", map[string]any{
		"date":       time.Now().UnixMilli(),
		"dateString": "2024-01-01T00:00:00",
		"device":     "sensor",
	}, "seed")

	for _, tc := range []struct {
		name string
		h    http.HandlerFunc
	}{
		{"version", Version(dep)},
		{"search", Search(dep)},
		{"create", Create(dep)},
		{"read", Read(dep)},
		{"update", Update(dep)},
		{"patch", Patch(dep)},
		{"remove", Remove(dep)},
		{"history", History(dep)},
	} {
		if tc.h == nil {
			t.Fatalf("%s handler is nil", tc.name)
		}
	}

	rec := httptest.NewRecorder()
	Status(dep)(rec, httptest.NewRequest(http.MethodGet, "/", nil), &auth.Identity{Name: "reader"})
	if rec.Code != http.StatusOK {
		t.Fatalf("Status failed")
	}
	rec = httptest.NewRecorder()
	Test(dep)(rec, httptest.NewRequest(http.MethodGet, "/", nil), &auth.Identity{Name: "reader"})
	if rec.Code != http.StatusOK {
		t.Fatalf("Test endpoint failed")
	}
	rec = httptest.NewRecorder()
	LastModified(dep)(rec, httptest.NewRequest(http.MethodGet, "/", nil), &auth.Identity{Name: "reader"})
	if rec.Code != http.StatusOK {
		t.Fatalf("LastModified failed")
	}

	if !validCollection("entries") || validCollection("missing") {
		t.Fatalf("validCollection mismatch")
	}
	if err := validateCreateDoc("treatments", map[string]any{"date": int64(1), "app": "x"}); err == nil {
		t.Fatalf("expected validateCreateDoc to reject old date")
	}
	existing := model.Record{ID: "id", Data: map[string]any{"identifier": "id", "date": int64(1700000000000), "device": "pump", "app": "x", "eventType": "Meal Bolus"}, IsValid: true}
	if err := validateReplaceDoc("treatments", existing, map[string]any{"date": int64(1700000000001)}); err == nil {
		t.Fatalf("expected validateReplaceDoc immutable failure")
	}
	if err := validatePatchDoc(existing, map[string]any{"identifier": "bad"}); err == nil {
		t.Fatalf("expected validatePatchDoc immutable failure")
	}
	if _, err := normalizeV3Document("treatments", map[string]any{"date": "bad", "app": "x"}); err == nil {
		t.Fatalf("expected normalizeV3Document failure")
	}
	if authErrorMessage(auth.ErrBadToken) != "Bad access token or JWT" || authErrorMessage(auth.ErrMissingCredentials) != "Missing or bad access token or JWT" {
		t.Fatalf("authErrorMessage mismatch")
	}
	if _, err := toInt64Compat("7"); err != nil {
		t.Fatalf("toInt64Compat failed: %v", err)
	}
	if _, err := toInt64Compat(true); err == nil {
		t.Fatalf("expected toInt64Compat to reject bool")
	}
	selected := selectV3Fields(model.Record{ID: "id", Data: map[string]any{"identifier": "id", "_id": "id", "date": 1}, IsValid: true}, nil)
	if _, ok := selected["_id"]; ok {
		t.Fatalf("selectV3Fields should omit _id")
	}
}

func TestV3HandlerLifecycle(t *testing.T) {
	dep := testDepsV3()
	manager := dep.Auth.(*testsupport.AuthManager)
	_ = manager.CreateRole("all", "api:treatments:*", "api:entries:read")
	subject := manager.CreateSubject("writer", []string{"all"})
	token, _ := manager.IssueJWT(subject.AccessToken)
	headers := map[string]string{"Authorization": "Bearer " + token}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"date":1700000000000,"app":"x","device":"pump","eventType":"Meal Bolus"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req = withParam(req, "collection", "treatments")
	rec := httptest.NewRecorder()
	Create(dep).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Create failed: %d %s", rec.Code, rec.Body.String())
	}
	body := decodeV3(t, rec)
	identifier := body["identifier"].(string)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/?fields=_all", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req = withParam(req, "collection", "treatments")
	req = withParam(req, "identifier", identifier)
	Read(dep).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Read failed: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"date":1700000000000,"app":"x","device":"pump","eventType":"Meal Bolus","carbs":10}`))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req = withParam(req, "collection", "treatments")
	req = withParam(req, "identifier", identifier)
	Update(dep).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Update failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(`{"carbs":11}`))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req = withParam(req, "collection", "treatments")
	req = withParam(req, "identifier", identifier)
	Patch(dep).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Patch failed: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/?limit=2&sort=date", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req = withParam(req, "collection", "treatments")
	Search(dep).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Search failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/?limit=2", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req = withParam(req, "collection", "treatments")
	req = withParam(req, "since", "0")
	History(dep).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("History failed: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req = withParam(req, "collection", "treatments")
	req = withParam(req, "identifier", identifier)
	Remove(dep).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Remove failed: %d", rec.Code)
	}

	if _, found, err := findCreateDuplicate(httptest.NewRequest(http.MethodGet, "/", nil), dep, "treatments", map[string]any{
		"identifier": identifier,
	}); err != nil || !found {
		t.Fatalf("findCreateDuplicate unexpected result: found=%v err=%v", found, err)
	}
}
