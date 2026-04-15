package apidocs

import (
	"embed"
	"fmt"
	"net/http"
)

//go:embed openapi.yaml
var openAPIFS embed.FS

func OpenAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFileFS(w, r, openAPIFS, "openapi.yaml")
}

func ScalarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = fmt.Fprintf(w, scalarHTML, "/openapi.yaml")
}

const scalarHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Horologia API Docs</title>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="%s"
      data-title="Horologia API"
      data-theme="default"
      data-layout="modern"
      data-default-http-client="fetch"
    ></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>
`
