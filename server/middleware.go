package server

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/titpetric/vuego"
	yaml "gopkg.in/yaml.v3"
)

// MiddlewareOption configures the middleware behavior.
type MiddlewareOption func(*middlewareConfig)

type middlewareConfig struct {
	loadOptions []vuego.LoadOption
}

// WithLoadOption adds a LoadOption to the middleware's Vue instance.
func WithLoadOption(opt ...vuego.LoadOption) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		cfg.loadOptions = append(cfg.loadOptions, opt...)
	}
}

// Middleware creates an http.Handler that processes .vuego files from the given filesystem.
// It renders .vuego files with accompanying .yml or .json data files.
// Non-.vuego requests are passed through to the next handler or return 404.
func Middleware(contentFS fs.FS, opts ...MiddlewareOption) http.Handler {
	cfg := &middlewareConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &middlewareHandler{
		fs:          contentFS,
		loadOptions: cfg.loadOptions,
	}
}

type middlewareHandler struct {
	fs          fs.FS
	loadOptions []vuego.LoadOption
}

func (h *middlewareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := strings.TrimPrefix(r.URL.Path, "/")

	// Try to serve as .vuego file
	vuegoPath := urlPath
	if !strings.HasSuffix(urlPath, ".vuego") {
		vuegoPath = urlPath + ".vuego"
	}

	// Check if .vuego file exists
	if _, err := fs.Stat(h.fs, vuegoPath); err == nil {
		h.serveVuego(w, r, vuegoPath)
		return
	}

	// Not a .vuego file, return 404
	http.NotFound(w, r)
}

func (h *middlewareHandler) serveVuego(w http.ResponseWriter, r *http.Request, filePath string) {
	// Load data from .yml or .json file
	data, err := LoadDataFile(h.fs, filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load data: %v", err), http.StatusInternalServerError)
		return
	}

	// Render the template
	tmpl := vuego.NewFS(h.fs, h.loadOptions...)
	var buf bytes.Buffer
	if err := tmpl.Load(filePath).Fill(data).Render(r.Context(), &buf); err != nil {
		http.Error(w, fmt.Sprintf("render error: %v", err), http.StatusInternalServerError)
		return
	}

	// Inject style link for adjacent .css or .less file
	html, err := injectStyleLink(&buf, filePath, r.URL.Path, h.fs)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to process HTML: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

// LoadDataFile loads data from .yml, .yaml, or .json file accompanying a .vuego file.
func LoadDataFile(contentFS fs.FS, vuegoPath string) (map[string]any, error) {
	basePath := strings.TrimSuffix(vuegoPath, ".vuego")
	data := make(map[string]any)

	// Try .yml first, then .yaml, then .json
	for _, ext := range []string{".yml", ".yaml", ".json"} {
		dataPath := basePath + ext
		content, err := fs.ReadFile(contentFS, dataPath)
		if err != nil {
			continue
		}

		if err := yaml.Unmarshal(content, &data); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", dataPath, err)
		}
		return data, nil
	}

	return data, nil
}

// MiddlewareDir creates an http.Handler that processes .vuego files from the given directory path.
// This is a convenience wrapper around Middleware that uses os.DirFS.
func MiddlewareDir(dir string, opts ...MiddlewareOption) http.Handler {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	return Middleware(os.DirFS(absDir), opts...)
}

// injectStyleLink finds an adjacent .css or .less file (preferring .css) and injects a style link.
func injectStyleLink(buf *bytes.Buffer, vuegoPath, urlPath string, contentFS fs.FS) (string, error) {
	htmlContent := buf.String()
	basePath := strings.TrimSuffix(vuegoPath, ".vuego")
	urlBasePath := strings.TrimSuffix(urlPath, ".vuego")

	// Check for .css first (preferred), then .less
	var styleExt string
	for _, ext := range []string{".css", ".less"} {
		stylePath := basePath + ext
		if _, err := fs.Stat(contentFS, stylePath); err == nil {
			styleExt = ext
			break
		}
	}

	// If no style file found, return original content
	if styleExt == "" {
		return htmlContent, nil
	}

	// Build single style link tag
	styleLink := fmt.Sprintf(`<link rel="stylesheet" href="%s%s">`, urlBasePath, styleExt)

	// Inject before </head> if it exists, otherwise before </body>, or at the end
	if idx := strings.Index(htmlContent, "</head>"); idx >= 0 {
		return htmlContent[:idx] + styleLink + htmlContent[idx:], nil
	}
	if idx := strings.Index(htmlContent, "</body>"); idx >= 0 {
		return htmlContent[:idx] + styleLink + htmlContent[idx:], nil
	}
	return htmlContent + styleLink, nil
}
