package openapi

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

// JSONHandler returns an HTTP handler that serves the OpenAPI spec as JSON.
func (s *Spec) JSONHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc := s.build()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(doc); err != nil {
			http.Error(w, "failed to encode spec", http.StatusInternalServerError)
		}
	}
}

// YAMLHandler returns an HTTP handler that serves the OpenAPI spec as YAML.
func (s *Spec) YAMLHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc := s.build()

		raw, err := json.Marshal(doc)
		if err != nil {
			http.Error(w, "failed to encode spec", http.StatusInternalServerError)
			return
		}

		var generic any
		if err := json.Unmarshal(raw, &generic); err != nil {
			http.Error(w, "failed to re-encode spec", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		if err := yaml.NewEncoder(w).Encode(generic); err != nil {
			http.Error(w, "failed to encode YAML", http.StatusInternalServerError)
		}
	}
}

// ScalarHandler returns an HTTP handler that serves the Scalar API reference UI.
// specURL is the URL where the spec JSON is served (e.g. "/openapi.json").
func (s *Spec) ScalarHandler(specURL string, opts ...ScalarOptions) http.HandlerFunc {
	o := ScalarOptions{}
	if len(opts) > 0 {
		o = opts[0]
	}

	html := buildScalarHTML(s.config.Title, specURL, o)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	}
}

// ScalarOptions customizes the Scalar API reference UI.
//
// Theme defaults to "kori", the kori brand theme. Set it to any built-in
// Scalar theme name (e.g. "default", "moon", "purple") to use that instead.
// CustomCSS is appended after the theme, so it can override any of it.
type ScalarOptions struct {
	Theme              string
	DarkMode           bool
	HideModels         bool
	HideDownloadButton bool
	DefaultOpenAllTags bool
	CustomCSS          string
}

//go:embed scalar_theme.css
var koriScalarTheme string

const koriFontLinks = `<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@fontsource-variable/geist/index.min.css" />
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@fontsource-variable/geist-mono/index.min.css" />`

// koriFavicon is the kori mark: a rounded diamond in the brand gradient.
const koriFavicon = `<link rel="icon" type="image/svg+xml" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'%3E%3Cdefs%3E%3ClinearGradient id='g' x1='0' y1='0' x2='1' y2='1'%3E%3Cstop offset='0' stop-color='%23bbd8ff'/%3E%3Cstop offset='.38' stop-color='%2375b7ff'/%3E%3Cstop offset='1' stop-color='%23315bff'/%3E%3C/linearGradient%3E%3C/defs%3E%3Crect x='7' y='7' width='18' height='18' rx='3.4' transform='rotate(45 16 16)' fill='url(%23g)'/%3E%3C/svg%3E" />`

func buildScalarHTML(title, specURL string, o ScalarOptions) string {
	theme := o.Theme
	if theme == "" {
		theme = "kori"
	}
	kori := theme == "kori"

	config := map[string]any{
		"url":                specURL,
		"theme":              theme,
		"darkMode":           o.DarkMode,
		"hideModels":         o.HideModels,
		"hideDownloadButton": o.HideDownloadButton,
		"defaultOpenAllTags": o.DefaultOpenAllTags,
	}
	if kori {
		config["theme"] = "none"
		config["withDefaultFonts"] = false
	}

	configJSON, _ := json.Marshal(config)

	var head strings.Builder
	if kori {
		head.WriteString(koriFavicon)
		head.WriteString("\n  ")
		head.WriteString(koriFontLinks)
		head.WriteString("\n  <style>\n")
		head.WriteString(koriScalarTheme)
		head.WriteString("\n  </style>")
	}
	if o.CustomCSS != "" {
		fmt.Fprintf(&head, "\n  <style>\n%s\n  </style>", o.CustomCSS)
	}

	return fmt.Sprintf(scalarTemplate, title, head.String(), strings.ReplaceAll(string(configJSON), "'", "&#39;"))
}

const scalarTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>%s</title>
  %s
</head>
<body>
  <script
    id="api-reference"
    type="application/json"
    data-configuration='%s'
  ></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`
