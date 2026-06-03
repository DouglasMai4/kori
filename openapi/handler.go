package openapi

import (
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
type ScalarOptions struct {
	Theme              string
	DarkMode           bool
	HideModels         bool
	HideDownloadButton bool
	DefaultOpenAllTags bool
	CustomCSS          string
}

func buildScalarHTML(title, specURL string, o ScalarOptions) string {
	theme := o.Theme
	if theme == "" {
		theme = "default"
	}

	config := map[string]any{
		"url":                specURL,
		"theme":              theme,
		"darkMode":           o.DarkMode,
		"hideModels":         o.HideModels,
		"hideDownloadButton": o.HideDownloadButton,
		"defaultOpenAllTags": o.DefaultOpenAllTags,
	}

	configJSON, _ := json.Marshal(config)

	customCSS := ""
	if o.CustomCSS != "" {
		customCSS = fmt.Sprintf("<style>\n%s\n  </style>", o.CustomCSS)
	}

	return fmt.Sprintf(scalarTemplate, title, customCSS, strings.ReplaceAll(string(configJSON), "'", "&#39;"))
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
