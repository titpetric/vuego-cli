package serve

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	chi "github.com/go-chi/chi/v5"
	"github.com/titpetric/lessgo"

	"github.com/titpetric/vuego-cli/server"
)

// Run executes the serve command with the given arguments.
// Usage: vuego serve [options] [directory].
func Run(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", ":8080", "HTTP server address")

	if err := fs.Parse(args); err != nil {
		return err
	}

	positional := fs.Args()
	dir := "."
	if len(positional) > 0 {
		dir = positional[0]
	}

	return Serve(dir, *addr)
}

// Serve starts an HTTP server that serves templates and assets from the given directory.
// It uses os.DirFS to create a filesystem rooted at the specified directory.
// The server provides:
// - .vuego file rendering via server middleware
// - .less file compilation via lessgo middleware
// - Directory listing and file serving for all other files.
func Serve(dir string, addr string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}

	if _, err := os.Stat(absDir); err != nil {
		return fmt.Errorf("directory not accessible: %w", err)
	}

	dirFS := os.DirFS(absDir)

	// Create router with middleware stack
	mux := chi.NewRouter()

	// File server with directory listing
	fileServer := http.FileServer(http.FS(dirFS))

	// Apply middleware in reverse order of application
	// lessgo middleware for .less files
	mux.Use(lessgo.NewMiddleware(dirFS, "/"))

	// vuego middleware for .vuego files (wrapped to pass through to file server)
	mux.Use(func(next http.Handler) http.Handler {
		return &vuegoMiddleware{
			vuegoHandler: server.Middleware(dirFS),
			next:         next,
		}
	})

	// File server with directory listing as final handler
	mux.Handle("/*", fileServer)

	log.Printf("Serving %s on http://localhost%s", absDir, addr)
	return http.ListenAndServe(addr, mux)
}

// vuegoMiddleware wraps the vuego handler and falls through to the next handler
// if the vuego handler doesn't handle the request (404).
type vuegoMiddleware struct {
	vuegoHandler http.Handler
	next         http.Handler
}

func (m *vuegoMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Buffer the response to check if it's a 404
	buf := &bytes.Buffer{}
	wrapped := &bufferingResponseWriter{
		buf:     buf,
		headers: make(http.Header),
	}
	m.vuegoHandler.ServeHTTP(wrapped, r)

	// If it was a 404, pass to next handler instead
	if wrapped.statusCode == http.StatusNotFound {
		m.next.ServeHTTP(w, r)
		return
	}

	// Write buffered response to actual response writer
	for k, v := range wrapped.headers {
		w.Header()[k] = v
	}
	w.WriteHeader(wrapped.statusCode)
	_, _ = w.Write(buf.Bytes())
}

// bufferingResponseWriter buffers response to check if it's a 404
type bufferingResponseWriter struct {
	buf        *bytes.Buffer
	headers    http.Header
	statusCode int
}

func (w *bufferingResponseWriter) Header() http.Header {
	return w.headers
}

func (w *bufferingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *bufferingResponseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.buf.Write(b)
}
