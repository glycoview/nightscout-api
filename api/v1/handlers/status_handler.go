package handlers

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/httpx"
)

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

func StatusTxt(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("STATUS OK"))
	}
}

func StatusHtml(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>Status OK</body></html>"))
	}
}

func StatusJs(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = fmt.Fprintf(w, "this.serverSettings = %s;", statusJSONPayload(dep))
	}
}

func StatusSvg(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://img.shields.io/badge/Nightscout-OK-green.svg", http.StatusFound)
	}
}

func StatusPng(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://img.shields.io/badge/Nightscout-OK-green.png", http.StatusFound)
	}
}

func statusJSONPayload(dep deps.Dependencies) string {
	return fmt.Sprintf(`{"apiEnabled":true,"careportalEnabled":%t,"settings":{"enable":["%s"]}}`, slices.Contains(dep.Config.Enable, "careportal"), strings.Join(dep.Config.Enable, `","`))
}
