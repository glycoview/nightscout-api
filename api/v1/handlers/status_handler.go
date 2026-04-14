package handlers

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/httpx"
)

// StatusJSON serves the v1 JSON status payload.
func StatusJSON(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"apiEnabled":        true,
			"careportalEnabled": slices.Contains(dep.Config.Enable, "careportal"),
			"settings": map[string]any{
				"enable": dep.Config.Enable,
			},
		})
	}
}

// StatusTxt serves the plain-text status response.
func StatusTxt(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("STATUS OK"))
	}
}

// StatusHtml serves a minimal HTML status response.
func StatusHtml(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>Status OK</body></html>"))
	}
}

// StatusJs serves the browser-oriented JavaScript status payload.
func StatusJs(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = fmt.Fprintf(w, "this.serverSettings = %s;", statusJSONPayload(dep))
	}
}

// StatusSvg redirects to a badge-style SVG status image.
func StatusSvg(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://img.shields.io/badge/Nightscout-OK-green.svg", http.StatusFound)
	}
}

// StatusPng redirects to a badge-style PNG status image.
func StatusPng(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://img.shields.io/badge/Nightscout-OK-green.png", http.StatusFound)
	}
}

func statusJSONPayload(dep deps.Dependencies) string {
	return fmt.Sprintf(`{"apiEnabled":true,"careportalEnabled":%t,"settings":{"enable":["%s"]}}`, slices.Contains(dep.Config.Enable, "careportal"), strings.Join(dep.Config.Enable, `","`))
}
