package handler

import (
	"net/http"
	"os"
	"path/filepath"
)

type DocsHandler struct {
	openapiDir string
}

func NewDocsHandler(openapiDir string) *DocsHandler {
	return &DocsHandler{openapiDir: openapiDir}
}

func (d *DocsHandler) HandleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	specPath := filepath.Join(d.openapiDir, "api", "v1", "openapi.yaml")
	data, err := os.ReadFile(specPath)
	if err != nil {
		http.Error(w, "spec not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(data)
}

func RedirectDocs(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
}

func (d *DocsHandler) HandleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(swaggerUIHTML))
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>API Documentation - Ecosyste.ms: Archives</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/docs/api/v1/openapi.yaml",
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
      ],
      layout: "BaseLayout"
    })
  </script>
</body>
</html>`
