package integration

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/glycoview/nightscout-api/config"
)

func TestV1StatusVersionsAndVerifyAuth(t *testing.T) {
	app := newTestApp(t, config.Config{
		APISecret:    "this is my long pass phrase",
		DefaultRoles: []string{"readable"},
		Enable:       []string{"careportal", "rawbg"},
	})

	res := app.request(http.MethodGet, "/api/status.json", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	statusBody := decodeJSONBody[map[string]any](t, res)
	if statusBody["apiEnabled"] != true || statusBody["careportalEnabled"] != true {
		t.Fatalf("unexpected status payload: %#v", statusBody)
	}
	settings := object(t, statusBody["settings"])
	enable := list(t, settings["enable"])
	if len(enable) != 2 {
		t.Fatalf("expected 2 enabled features, got %d", len(enable))
	}

	res = app.request(http.MethodGet, "/api/status.html", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	mustEqual(t, res.Header.Get("Content-Type"), "text/html", "status.html content type")
	_ = readBody(t, res)

	res = app.requestNoRedirect(http.MethodGet, "/api/status.svg", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusFound)
	_ = readBody(t, res)

	res = app.request(http.MethodGet, "/api/status.txt", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	mustEqual(t, readBody(t, res), "STATUS OK", "status.txt body")

	res = app.request(http.MethodGet, "/api/status.js", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	mustEqual(t, res.Header.Get("Content-Type"), "application/javascript", "status.js content type")
	js := readBody(t, res)
	if !strings.HasPrefix(js, "this.serverSettings =") {
		t.Fatalf("unexpected status.js payload: %q", js)
	}

	res = app.requestNoRedirect(http.MethodGet, "/api/status.png", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusFound)
	mustEqual(t, res.Header.Get("Location"), "http://img.shields.io/badge/Nightscout-OK-green.png", "status.png redirect")
	_ = readBody(t, res)

	res = app.request(http.MethodGet, "/api/versions", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	versions := decodeJSONBody[[]map[string]string](t, res)
	if len(versions) < 3 {
		t.Fatalf("expected at least 3 versions, got %d", len(versions))
	}

	denied := newTestApp(t, config.Config{
		APISecret:    "this is my long pass phrase",
		DefaultRoles: []string{"denied"},
	})
	res = denied.request(http.MethodGet, "/api/verifyauth", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	verifyBody := decodeJSONBody[map[string]any](t, res)
	message := object(t, verifyBody["message"])
	mustEqual(t, text(t, message["message"]), "UNAUTHORIZED", "verifyauth unauthorized")

	res = denied.request(http.MethodGet, "/api/verifyauth", nil, map[string]string{"api-secret": apiSecretHash(denied.config.APISecret)})
	mustStatus(t, res.StatusCode, http.StatusOK)
	verifyBody = decodeJSONBody[map[string]any](t, res)
	message = object(t, verifyBody["message"])
	mustEqual(t, text(t, message["message"]), "OK", "verifyauth authorized")
}

func TestV1EntriesRoutes(t *testing.T) {
	app := newTestApp(t, config.Config{
		APISecret:    "this is my long pass phrase",
		DefaultRoles: []string{"readable"},
	})
	ids := seedHistoricalEntries(t, app)

	res := app.request(http.MethodGet, "/api/entries.json?find[dateString][$gte]=2014-07-19&count=30", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	entries := decodeJSONBody[[]map[string]any](t, res)
	if len(entries) != 30 {
		t.Fatalf("expected 30 entries, got %d", len(entries))
	}

	res = app.request(http.MethodGet, "/api/entries/sgv.json?find[dateString][$gte]=2014-07-19&find[dateString][$lte]=2014-07-20", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	entries = decodeJSONBody[[]map[string]any](t, res)
	if len(entries) != 10 {
		t.Fatalf("expected default 10 entries, got %d", len(entries))
	}
	if number(t, entries[0]["date"]) <= number(t, entries[1]["date"]) {
		t.Fatalf("expected descending entries order")
	}

	res = app.request(http.MethodGet, "/api/echo/entries/sgv.json?find[dateString][$gte]=2014-07-19", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	echoBody := decodeJSONBody[map[string]any](t, res)
	mustEqual(t, text(t, echoBody["storage"]), "entries", "echo storage")

	res = app.request(http.MethodGet, "/api/slice/entries/dateString/sgv/2014-07-19.json?count=20", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	sliced := decodeJSONBody[[]map[string]any](t, res)
	if len(sliced) != 20 {
		t.Fatalf("expected 20 sliced entries, got %d", len(sliced))
	}

	res = app.request(http.MethodGet, "/api/times/echo/2014-07/T09:.json?count=20&find[sgv][$gte]=100", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	timesEcho := decodeJSONBody[map[string]any](t, res)
	patterns := list(t, timesEcho["pattern"])
	if len(patterns) == 0 {
		t.Fatalf("expected non-empty times echo pattern")
	}

	res = app.request(http.MethodGet, "/api/times/2014-07/T09:.json", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	timed := decodeJSONBody[[]map[string]any](t, res)
	if len(timed) != 10 {
		t.Fatalf("expected default 10 modal time entries, got %d", len(timed))
	}

	res = app.request(http.MethodGet, "/api/entries/current.json", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	current := decodeJSONBody[[]map[string]any](t, res)
	if len(current) != 1 {
		t.Fatalf("expected one current entry, got %d", len(current))
	}

	res = app.request(http.MethodGet, "/api/entries/"+ids[0]+".json", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	byID := decodeJSONBody[[]map[string]any](t, res)
	if len(byID) != 1 || text(t, byID[0]["_id"]) != ids[0] {
		t.Fatalf("unexpected id lookup response: %#v", byID)
	}
}

func TestV1EntriesWriteAuthorizationAndPreview(t *testing.T) {
	app := newTestApp(t, config.Config{
		APISecret:    "this is my long pass phrase",
		DefaultRoles: []string{"readable"},
	})

	now := time.Now().UTC().Truncate(time.Second)
	payload := []map[string]any{{
		"type":       "sgv",
		"sgv":        100,
		"date":       now.UnixMilli(),
		"dateString": now.Format(time.RFC3339),
		"device":     "write-test",
	}}

	res := app.request(http.MethodPost, "/api/entries.json", payload, nil)
	mustStatus(t, res.StatusCode, http.StatusUnauthorized)
	unauthorized := decodeJSONBody[map[string]any](t, res)
	mustEqual(t, text(t, unauthorized["message"]), "Unauthorized", "unauthorized post message")

	res = app.request(http.MethodPost, "/api/entries/preview.json", payload, map[string]string{"api-secret": apiSecretHash(app.config.APISecret)})
	mustStatus(t, res.StatusCode, http.StatusCreated)
	preview := decodeJSONBody[[]map[string]any](t, res)
	if len(preview) != 1 {
		t.Fatalf("expected one preview entry, got %d", len(preview))
	}

	res = app.request(http.MethodPost, "/api/entries.json", payload, map[string]string{"api-secret": apiSecretHash(app.config.APISecret)})
	mustStatus(t, res.StatusCode, http.StatusOK)
	created := decodeJSONBody[[]map[string]any](t, res)
	if len(created) != 1 {
		t.Fatalf("expected one created entry, got %d", len(created))
	}

	datePath := url.QueryEscape(now.Format("2006-01-02"))
	res = app.request(http.MethodGet, "/api/slice/entries/dateString/sgv/"+datePath+".json", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	found := decodeJSONBody[[]map[string]any](t, res)
	if len(found) != 1 {
		t.Fatalf("expected one persisted entry, got %d", len(found))
	}

	res = app.request(http.MethodDelete, "/api/entries/sgv?find[dateString]="+url.QueryEscape(now.Format(time.RFC3339)), nil, map[string]string{"api-secret": apiSecretHash(app.config.APISecret)})
	mustStatus(t, res.StatusCode, http.StatusOK)
}

func TestV1TreatmentsAndDeviceStatus(t *testing.T) {
	app := newTestApp(t, config.Config{
		APISecret:    "this is my long pass phrase",
		DefaultRoles: []string{"readable"},
		Enable:       []string{"careportal", "api"},
	})

	res := app.request(http.MethodPost, "/api/treatments/", map[string]any{
		"eventType":   "Meal Bolus",
		"created_at":  "2024-01-02T03:04:05.000+02:00",
		"carbs":       "30",
		"insulin":     "2.00",
		"glucose":     100,
		"glucoseType": "Finger",
		"notes":       "<IMG SRC=\"javascript:alert('XSS');\">",
		"app":         "test",
		"device":      "pump",
	}, map[string]string{"api-secret": apiSecretHash(app.config.APISecret)})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("unexpected treatment create status: %d body=%s", res.StatusCode, readBody(t, res))
	}
	mustStatus(t, res.StatusCode, http.StatusOK)
	_ = decodeJSONBody[[]map[string]any](t, res)

	res = app.request(http.MethodGet, "/api/treatments?find[eventType]="+url.QueryEscape("Meal Bolus")+"&find[created_at][$gte]=2024-01-02", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	treatments := decodeJSONBody[[]map[string]any](t, res)
	if len(treatments) != 1 {
		t.Fatalf("expected one treatment, got %d", len(treatments))
	}
	treatment := treatments[0]
	mustEqual(t, text(t, treatment["notes"]), "<img>", "sanitized notes")
	mustEqual(t, number(t, treatment["carbs"]), 30, "numeric carbs")
	mustEqual(t, number(t, treatment["insulin"]), 2, "numeric insulin")
	mustEqual(t, text(t, treatment["created_at"]), "2024-01-02T01:04:05.000Z", "normalized created_at")
	mustEqual(t, number(t, treatment["utcOffset"]), 120, "stored utcOffset")

	res = app.request(http.MethodDelete, "/api/treatments/?find[carbs]=30&find[created_at][$gte]=2024-01-02", nil, map[string]string{"api-secret": apiSecretHash(app.config.APISecret)})
	mustStatus(t, res.StatusCode, http.StatusOK)

	res = app.request(http.MethodGet, "/api/treatments?find[carbs]=30&find[created_at][$gte]=2024-01-02", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	treatments = decodeJSONBody[[]map[string]any](t, res)
	if len(treatments) != 0 {
		t.Fatalf("expected treatments delete to remove records, got %d", len(treatments))
	}

	res = app.request(http.MethodPost, "/api/devicestatus/", map[string]any{
		"device":     "xdripjs://rigName",
		"created_at": "2018-12-16T01:00:52Z",
		"xdripjs": map[string]any{
			"state": 6,
		},
		"app": "test",
	}, map[string]string{"api-secret": apiSecretHash(app.config.APISecret)})
	mustStatus(t, res.StatusCode, http.StatusOK)

	res = app.request(http.MethodGet, "/api/devicestatus?find[created_at][$gte]=2018-12-16&find[created_at][$lte]=2018-12-17", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	deviceStatuses := decodeJSONBody[[]map[string]any](t, res)
	if len(deviceStatuses) != 1 {
		t.Fatalf("expected one devicestatus, got %d", len(deviceStatuses))
	}
	ds := deviceStatuses[0]
	xdrip := object(t, ds["xdripjs"])
	mustEqual(t, number(t, xdrip["state"]), 6, "devicestatus nested state")
	mustEqual(t, number(t, ds["utcOffset"]), 0, "devicestatus utcOffset")
}

func seedHistoricalEntries(t *testing.T, app *testApp) []string {
	t.Helper()

	ctx := context.Background()
	base := time.Date(2014, 7, 19, 11, 25, 0, 0, time.UTC)
	ids := make([]string, 0, 30)
	for i := 0; i < 30; i++ {
		when := base.Add(-5 * time.Minute * time.Duration(i))
		record, _, err := app.store.Create(ctx, "entries", map[string]any{
			"type":       "sgv",
			"sgv":        100,
			"date":       when.UnixMilli(),
			"dateString": when.Format("2006-01-02T15:04:05"),
			"device":     "seed-device",
		}, "seed")
		if err != nil {
			t.Fatalf("seed historical entry: %v", err)
		}
		ids = append(ids, record.Identifier())
	}
	return ids
}
