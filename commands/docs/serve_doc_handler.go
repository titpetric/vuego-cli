package docs

import (
	"net/http"
)

// ServeDocHandler returns an http.HandlerFunc that serves docs (for testing).
func ServeDocHandler(m *Module) http.HandlerFunc {
	return handler(m.serveDoc)
}
