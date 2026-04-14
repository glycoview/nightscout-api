package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glycoview/nightscout-api/store"
)

func TestWriteAndReadJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusCreated, map[string]any{"ok": true})
	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("unexpected content type: %s", rec.Header().Get("Content-Type"))
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"num":1}`))
	var dst map[string]any
	if err := ReadJSON(req, &dst); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}
}

func TestRequireRecord(t *testing.T) {
	status, _ := RequireRecord(store.ErrGone)
	if status != http.StatusGone {
		t.Fatalf("expected 410, got %d", status)
	}
	status, _ = RequireRecord(store.ErrNotFound)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
	status, body := RequireRecord(errors.New("boom"))
	if status != http.StatusInternalServerError || body["message"] != "boom" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestConditionalHeadersAndParsing(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-Modified-Since", now.Format(http.TimeFormat))
	if _, ok := ParseIfModifiedSince(req); !ok {
		t.Fatalf("expected valid If-Modified-Since")
	}
	req.Header.Set("If-Unmodified-Since", now.Format(http.TimeFormat))
	if _, ok := ParseIfUnmodifiedSince(req); !ok {
		t.Fatalf("expected valid If-Unmodified-Since")
	}

	rec := httptest.NewRecorder()
	LastModifiedHeader(rec, now.UnixMilli())
	if rec.Header().Get("Last-Modified") == "" {
		t.Fatalf("expected Last-Modified header")
	}
	rec = httptest.NewRecorder()
	LastModifiedHeader(rec, 0)
	if rec.Header().Get("Last-Modified") != "" {
		t.Fatalf("expected no Last-Modified header for zero millis")
	}

	if ParsePositiveInt("12", 5) != 12 || ParsePositiveInt("-1", 5) != 5 {
		t.Fatalf("ParsePositiveInt mismatch")
	}
}
