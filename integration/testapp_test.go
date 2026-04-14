package integration

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "github.com/glycoview/nightscout-api/api/v1"
	v3 "github.com/glycoview/nightscout-api/api/v3"
	"github.com/glycoview/nightscout-api/config"
	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/internal/testsupport"
)

type testApp struct {
	t      *testing.T
	server *httptest.Server
	auth   *testsupport.AuthManager
	store  *testsupport.MemoryStore
	config config.Config
}

func newTestApp(t *testing.T, cfg config.Config) *testApp {
	t.Helper()

	cfg = cfg.WithDefaults()
	authManager := testsupport.NewAuthManager(cfg.APISecret, cfg.DefaultRoles)
	memStore := testsupport.NewMemoryStore()
	dep := deps.Dependencies{
		Config: cfg,
		Store:  memStore,
		Auth:   authManager,
	}

	mux := http.NewServeMux()
	mux.Handle("/api/v3/", http.StripPrefix("/api/v3", v3.NewNightscoutV3Router(dep)))
	mux.Handle("/api/", http.StripPrefix("/api", v1.NewNightscoutV1Router(dep)))

	app := &testApp{
		t:      t,
		server: httptest.NewServer(mux),
		auth:   authManager,
		store:  memStore,
		config: cfg,
	}
	t.Cleanup(app.close)
	return app
}

func (a *testApp) close() {
	if a.server != nil {
		a.server.Close()
	}
}

func (a *testApp) request(method, path string, body any, headers map[string]string) *http.Response {
	a.t.Helper()
	return a.requestWithClient(http.DefaultClient, method, path, body, headers)
}

func (a *testApp) requestNoRedirect(method, path string, body any, headers map[string]string) *http.Response {
	a.t.Helper()
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return a.requestWithClient(client, method, path, body, headers)
}

func (a *testApp) requestWithClient(client *http.Client, method, path string, body any, headers map[string]string) *http.Response {
	a.t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			a.t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, a.server.URL+path, reader)
	if err != nil {
		a.t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	res, err := client.Do(req)
	if err != nil {
		a.t.Fatalf("do request: %v", err)
	}
	return res
}

func decodeJSONBody[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	defer res.Body.Close()

	var value T
	if err := json.NewDecoder(res.Body).Decode(&value); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return value
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return string(data)
}

func apiSecretHash(secret string) string {
	sum := sha1.Sum([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func (a *testApp) issueCollectionJWTs(t *testing.T, baseName, collection string, actions ...string) map[string]string {
	t.Helper()

	result := make(map[string]string, len(actions))
	for _, action := range actions {
		roleName := fmt.Sprintf("%s-%s-%s", baseName, collection, action)
		permission := fmt.Sprintf("api:%s:%s", collection, action)
		if action == "all" {
			permission = fmt.Sprintf("api:%s:*", collection)
		}
		if err := a.auth.CreateRole(roleName, permission); err != nil {
			t.Fatalf("create role: %v", err)
		}
		subject := a.auth.CreateSubject(roleName, []string{roleName})
		token, err := a.auth.IssueJWT(subject.AccessToken)
		if err != nil {
			t.Fatalf("issue jwt: %v", err)
		}
		result[action] = token
	}
	return result
}

func mustStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected status: got %d want %d", got, want)
	}
}

func mustEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v want %v", msg, got, want)
	}
}

func number(t *testing.T, value any) float64 {
	t.Helper()
	num, ok := value.(float64)
	if !ok {
		t.Fatalf("expected JSON number, got %T", value)
	}
	return num
}

func text(t *testing.T, value any) string {
	t.Helper()
	s, ok := value.(string)
	if !ok {
		t.Fatalf("expected JSON string, got %T", value)
	}
	return s
}

func object(t *testing.T, value any) map[string]any {
	t.Helper()
	obj, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %T", value)
	}
	return obj
}

func list(t *testing.T, value any) []any {
	t.Helper()
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("expected list, got %T", value)
	}
	return items
}

func seedEntry(t *testing.T, app *testApp, when time.Time, sgv int, device string) string {
	t.Helper()
	record, _, err := app.store.Create(context.Background(), "entries", map[string]any{
		"type":       "sgv",
		"sgv":        sgv,
		"date":       when.UnixMilli(),
		"dateString": when.UTC().Format("2006-01-02T15:04:05"),
		"device":     device,
	}, "seed")
	if err != nil {
		t.Fatalf("seed entry: %v", err)
	}
	return record.Identifier()
}
