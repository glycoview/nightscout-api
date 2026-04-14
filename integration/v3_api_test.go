package integration

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/glycoview/nightscout-api/config"
)

func TestV3VersionAndCreate(t *testing.T) {
	app := newTestApp(t, config.Config{
		APISecret:    "this is my long pass phrase",
		AppVersion:   "1.2.3",
		DefaultRoles: []string{"denied"},
	})
	jwts := app.issueCollectionJWTs(t, "api3-create", "treatments", "create", "read")

	res := app.request(http.MethodGet, "/api/v3/version", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusOK)
	versionBody := decodeJSONBody[map[string]any](t, res)
	result := object(t, versionBody["result"])
	mustEqual(t, text(t, result["version"]), "1.2.3", "api version app version")
	mustEqual(t, text(t, result["apiVersion"]), "3.0.4", "api version")

	valid := map[string]any{
		"date":      time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC).UnixMilli(),
		"app":       "test-app",
		"device":    "device-api3-create",
		"eventType": "Correction Bolus",
		"insulin":   0.3,
	}

	res = app.request(http.MethodPost, "/api/v3/treatments", valid, nil)
	mustStatus(t, res.StatusCode, http.StatusUnauthorized)
	unauthorized := decodeJSONBody[map[string]any](t, res)
	mustEqual(t, text(t, unauthorized["message"]), "Missing or bad access token or JWT", "create requires auth")

	res = app.request(http.MethodPost, "/api/v3/NOT_EXIST", valid, map[string]string{"Authorization": "Bearer " + jwts["create"]})
	mustStatus(t, res.StatusCode, http.StatusNotFound)

	res = app.request(http.MethodPost, "/api/v3/treatments", valid, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusForbidden)
	forbidden := decodeJSONBody[map[string]any](t, res)
	mustEqual(t, text(t, forbidden["message"]), "Missing permission api:treatments:create", "create permission")

	res = app.request(http.MethodPost, "/api/v3/treatments", map[string]any{}, map[string]string{"Authorization": "Bearer " + jwts["create"]})
	mustStatus(t, res.StatusCode, http.StatusBadRequest)

	res = app.request(http.MethodPost, "/api/v3/treatments", valid, map[string]string{"Authorization": "Bearer " + jwts["create"]})
	mustStatus(t, res.StatusCode, http.StatusCreated)
	created := decodeJSONBody[map[string]any](t, res)
	identifier := text(t, created["identifier"])
	if !strings.HasPrefix(res.Header.Get("Location"), "/api/v3/treatments/") {
		t.Fatalf("unexpected create location: %s", res.Header.Get("Location"))
	}
	if res.Header.Get("Last-Modified") == "" {
		t.Fatalf("expected Last-Modified header")
	}

	res = app.request(http.MethodGet, "/api/v3/treatments/"+identifier, nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusOK)
	read := decodeJSONBody[map[string]any](t, res)
	doc := object(t, read["result"])
	mustEqual(t, text(t, doc["identifier"]), identifier, "created identifier roundtrip")

	res = app.request(http.MethodPost, "/api/v3/treatments", map[string]any{
		"date":      "2019-06-10T08:07:08,576+02:00",
		"app":       "test-app",
		"device":    "device-api3-create-zoned",
		"eventType": "Correction Bolus",
		"insulin":   0.3,
	}, map[string]string{"Authorization": "Bearer " + jwts["create"]})
	mustStatus(t, res.StatusCode, http.StatusCreated)
	zoned := decodeJSONBody[map[string]any](t, res)

	res = app.request(http.MethodGet, "/api/v3/treatments/"+text(t, zoned["identifier"]), nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusOK)
	zonedRead := decodeJSONBody[map[string]any](t, res)
	zonedDoc := object(t, zonedRead["result"])
	mustEqual(t, number(t, zonedDoc["date"]), 1560146828576, "normalized date")
	mustEqual(t, number(t, zonedDoc["utcOffset"]), 120, "normalized utcOffset")

	for _, tc := range []struct {
		name string
		doc  map[string]any
		msg  string
	}{
		{
			name: "missing-date",
			doc: map[string]any{
				"app":       "test-app",
				"device":    "device-missing-date",
				"eventType": "Correction Bolus",
			},
			msg: "Bad or missing date field",
		},
		{
			name: "bad-offset",
			doc: map[string]any{
				"date":      valid["date"],
				"app":       "test-app",
				"device":    "device-bad-offset",
				"eventType": "Correction Bolus",
				"utcOffset": -5000,
			},
			msg: "Bad or missing utcOffset field",
		},
		{
			name: "missing-app",
			doc: map[string]any{
				"date":      valid["date"],
				"device":    "device-missing-app",
				"eventType": "Correction Bolus",
			},
			msg: "Bad or missing app field",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := app.request(http.MethodPost, "/api/v3/treatments", tc.doc, map[string]string{"Authorization": "Bearer " + jwts["create"]})
			mustStatus(t, res.StatusCode, http.StatusBadRequest)
			body := decodeJSONBody[map[string]any](t, res)
			mustEqual(t, text(t, body["message"]), tc.msg, tc.name)
		})
	}
}

func TestV3ReadAndDeleteLifecycle(t *testing.T) {
	app := newTestApp(t, config.Config{
		APISecret:    "this is my long pass phrase",
		DefaultRoles: []string{"denied"},
	})
	jwts := app.issueCollectionJWTs(t, "api3-read", "devicestatus", "create", "read", "delete")

	valid := map[string]any{
		"date":            time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC).UnixMilli(),
		"app":             "test-app",
		"device":          "device-api3-read",
		"uploaderBattery": 58,
	}

	res := app.request(http.MethodGet, "/api/v3/devicestatus/FAKE_IDENTIFIER", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusUnauthorized)

	res = app.request(http.MethodPost, "/api/v3/devicestatus", valid, map[string]string{"Authorization": "Bearer " + jwts["create"]})
	mustStatus(t, res.StatusCode, http.StatusCreated)
	created := decodeJSONBody[map[string]any](t, res)
	identifier := text(t, created["identifier"])

	res = app.request(http.MethodGet, "/api/v3/devicestatus/"+identifier+"?fields=date,device,subject", nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusOK)
	selected := decodeJSONBody[map[string]any](t, res)
	selectedResult := object(t, selected["result"])
	if len(selectedResult) != 3 {
		t.Fatalf("expected only selected fields, got %#v", selectedResult)
	}

	res = app.request(http.MethodGet, "/api/v3/devicestatus/"+identifier+"?fields=_all", nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusOK)
	allFields := decodeJSONBody[map[string]any](t, res)
	full := object(t, allFields["result"])
	if _, ok := full["_id"]; ok {
		t.Fatalf("expected API3 read to omit _id")
	}
	if _, ok := full["srvModified"]; !ok {
		t.Fatalf("expected srvModified in full read")
	}

	res = app.request(http.MethodGet, "/api/v3/devicestatus/"+identifier, nil, map[string]string{
		"Authorization":     "Bearer " + jwts["read"],
		"If-Modified-Since": time.Now().UTC().Add(time.Second).Format(http.TimeFormat),
	})
	mustStatus(t, res.StatusCode, http.StatusNotModified)
	_ = readBody(t, res)

	res = app.request(http.MethodDelete, "/api/v3/devicestatus/"+identifier, nil, map[string]string{"Authorization": "Bearer " + jwts["delete"]})
	mustStatus(t, res.StatusCode, http.StatusOK)

	res = app.request(http.MethodGet, "/api/v3/devicestatus/"+identifier, nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusGone)

	res = app.request(http.MethodPost, "/api/v3/devicestatus", map[string]any{
		"date":            time.Date(2024, 2, 3, 4, 6, 6, 0, time.UTC).UnixMilli(),
		"app":             "test-app",
		"device":          "device-api3-read-hard-delete",
		"uploaderBattery": 59,
	}, map[string]string{"Authorization": "Bearer " + jwts["create"]})
	mustStatus(t, res.StatusCode, http.StatusCreated)
	created = decodeJSONBody[map[string]any](t, res)
	identifier = text(t, created["identifier"])

	res = app.request(http.MethodDelete, "/api/v3/devicestatus/"+identifier+"?permanent=true", nil, map[string]string{"Authorization": "Bearer " + jwts["delete"]})
	mustStatus(t, res.StatusCode, http.StatusOK)

	res = app.request(http.MethodGet, "/api/v3/devicestatus/"+identifier, nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusNotFound)
}

func TestV3Search(t *testing.T) {
	app := newTestApp(t, config.Config{
		APISecret:    "this is my long pass phrase",
		DefaultRoles: []string{"denied"},
		API3MaxLimit: 5,
	})
	jwts := app.issueCollectionJWTs(t, "api3-search", "entries", "all", "read")

	ctx := context.Background()
	for i := 0; i < 8; i++ {
		_, _, err := app.store.Create(ctx, "entries", map[string]any{
			"type":       "sgv",
			"sgv":        100 + i,
			"date":       time.Date(2024, 3, 1, 0, i, 0, 0, time.UTC).UnixMilli(),
			"dateString": time.Date(2024, 3, 1, 0, i, 0, 0, time.UTC).Format(time.RFC3339),
			"device":     "device-api3-search-" + strconv.Itoa(i),
			"app":        "test-app",
		}, "seed")
		if err != nil {
			t.Fatalf("seed entries for search: %v", err)
		}
	}

	res := app.request(http.MethodGet, "/api/v3/entries", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusUnauthorized)

	for _, tc := range []struct {
		path string
		msg  string
	}{
		{path: "/api/v3/entries?limit=INVALID", msg: "Parameter limit out of tolerance"},
		{path: "/api/v3/entries?skip=-1", msg: "Parameter skip out of tolerance"},
		{path: "/api/v3/entries?sort=date&sort$desc=created_at", msg: "Parameters sort and sort_desc cannot be combined"},
	} {
		res := app.request(http.MethodGet, tc.path, nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
		mustStatus(t, res.StatusCode, http.StatusBadRequest)
		body := decodeJSONBody[map[string]any](t, res)
		mustEqual(t, text(t, body["message"]), tc.msg, tc.path)
	}

	res = app.request(http.MethodGet, "/api/v3/entries", nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusOK)
	limitedDefault := decodeJSONBody[map[string]any](t, res)
	if len(list(t, limitedDefault["result"])) != 5 {
		t.Fatalf("expected API3 ceiling limit of 5 by default")
	}

	res = app.request(http.MethodGet, "/api/v3/entries?sort=date&limit=3", nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusOK)
	ascending := list(t, decodeJSONBody[map[string]any](t, res)["result"])
	if len(ascending) != 3 {
		t.Fatalf("expected three search results, got %d", len(ascending))
	}

	res = app.request(http.MethodGet, "/api/v3/entries?sort$desc=date&limit=3", nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusOK)
	descending := list(t, decodeJSONBody[map[string]any](t, res)["result"])
	if number(t, object(t, ascending[0])["date"]) >= number(t, object(t, ascending[1])["date"]) {
		t.Fatalf("expected ascending sort by date")
	}
	if number(t, object(t, descending[0])["date"]) <= number(t, object(t, descending[1])["date"]) {
		t.Fatalf("expected descending sort by date")
	}

	res = app.request(http.MethodGet, "/api/v3/entries?sort=date&skip=2&limit=2", nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusOK)
	skipped := list(t, decodeJSONBody[map[string]any](t, res)["result"])
	if len(skipped) != 2 {
		t.Fatalf("expected two skipped results, got %d", len(skipped))
	}

	res = app.request(http.MethodGet, "/api/v3/entries?fields=date,app,subject&limit=2", nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusOK)
	projected := list(t, decodeJSONBody[map[string]any](t, res)["result"])
	if len(object(t, projected[0])) != 3 {
		t.Fatalf("expected projected fields only, got %#v", projected[0])
	}
}

func TestV3UpdateAndPatch(t *testing.T) {
	app := newTestApp(t, config.Config{
		APISecret:    "this is my long pass phrase",
		DefaultRoles: []string{"denied"},
	})
	jwts := app.issueCollectionJWTs(t, "api3-update", "treatments", "create", "read", "update", "all")

	date := time.Date(2024, 4, 5, 6, 7, 8, 0, time.UTC).UnixMilli()
	base := map[string]any{
		"identifier": "test-update-identifier",
		"date":       date,
		"utcOffset":  -180,
		"app":        "test-app",
		"device":     "device-api3-update",
		"eventType":  "Correction Bolus",
		"insulin":    0.3,
	}

	res := app.request(http.MethodPut, "/api/v3/treatments/"+text(t, base["identifier"]), base, nil)
	mustStatus(t, res.StatusCode, http.StatusUnauthorized)

	res = app.request(http.MethodPut, "/api/v3/treatments/"+text(t, base["identifier"]), base, map[string]string{"Authorization": "Bearer " + jwts["update"]})
	mustStatus(t, res.StatusCode, http.StatusForbidden)
	forbidden := decodeJSONBody[map[string]any](t, res)
	mustEqual(t, text(t, forbidden["message"]), "Missing permission api:treatments:create", "upsert create permission")

	res = app.request(http.MethodPut, "/api/v3/treatments/"+text(t, base["identifier"]), base, map[string]string{"Authorization": "Bearer " + jwts["all"]})
	mustStatus(t, res.StatusCode, http.StatusCreated)
	created := decodeJSONBody[map[string]any](t, res)
	mustEqual(t, text(t, created["identifier"]), "test-update-identifier", "upsert identifier")

	updateDoc := map[string]any{
		"identifier": "MODIFIED",
		"date":       date,
		"utcOffset":  -180,
		"app":        "test-app",
		"device":     "device-api3-update",
		"eventType":  "Correction Bolus",
		"carbs":      10,
	}
	res = app.request(http.MethodPut, "/api/v3/treatments/test-update-identifier", updateDoc, map[string]string{"Authorization": "Bearer " + jwts["update"]})
	mustStatus(t, res.StatusCode, http.StatusOK)

	res = app.request(http.MethodGet, "/api/v3/treatments/test-update-identifier", nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusOK)
	updated := object(t, decodeJSONBody[map[string]any](t, res)["result"])
	if _, ok := updated["insulin"]; ok {
		t.Fatalf("expected PUT to replace the document")
	}
	mustEqual(t, number(t, updated["carbs"]), 10, "updated carbs")

	res = app.request(http.MethodPut, "/api/v3/treatments/test-update-identifier", map[string]any{
		"date":      date + 1000,
		"utcOffset": -180,
		"app":       "test-app",
		"device":    "device-api3-update",
		"eventType": "Correction Bolus",
	}, map[string]string{"Authorization": "Bearer " + jwts["update"]})
	mustStatus(t, res.StatusCode, http.StatusBadRequest)
	immutable := decodeJSONBody[map[string]any](t, res)
	mustEqual(t, text(t, immutable["message"]), "Field date cannot be modified by the client", "immutable date")

	res = app.request(http.MethodPatch, "/api/v3/treatments/test-update-identifier", map[string]any{
		"carbs": 11,
	}, map[string]string{"Authorization": "Bearer " + jwts["update"]})
	mustStatus(t, res.StatusCode, http.StatusOK)

	res = app.request(http.MethodGet, "/api/v3/treatments/test-update-identifier", nil, map[string]string{"Authorization": "Bearer " + jwts["read"]})
	mustStatus(t, res.StatusCode, http.StatusOK)
	patched := object(t, decodeJSONBody[map[string]any](t, res)["result"])
	mustEqual(t, number(t, patched["carbs"]), 11, "patched carbs")
	mustEqual(t, text(t, patched["modifiedBy"]), "api3-update-treatments-update", "patch modifiedBy")

	res = app.request(http.MethodPatch, "/api/v3/treatments/test-update-identifier", map[string]any{
		"identifier": "MODIFIED",
	}, map[string]string{"Authorization": "Bearer " + jwts["update"]})
	mustStatus(t, res.StatusCode, http.StatusBadRequest)
	immutable = decodeJSONBody[map[string]any](t, res)
	mustEqual(t, text(t, immutable["message"]), "Field identifier cannot be modified by the client", "immutable identifier")
}

func TestV3DeleteRequiresAuthentication(t *testing.T) {
	app := newTestApp(t, config.Config{
		APISecret:    "this is my long pass phrase",
		DefaultRoles: []string{"denied"},
	})
	jwts := app.issueCollectionJWTs(t, "api3-delete", "treatments", "delete")

	res := app.request(http.MethodDelete, "/api/v3/treatments/FAKE_IDENTIFIER", nil, nil)
	mustStatus(t, res.StatusCode, http.StatusUnauthorized)

	res = app.request(http.MethodDelete, "/api/v3/NOT_EXIST/FAKE_IDENTIFIER", nil, map[string]string{"Authorization": "Bearer " + jwts["delete"]})
	mustStatus(t, res.StatusCode, http.StatusNotFound)
}
