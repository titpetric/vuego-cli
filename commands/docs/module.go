package docs

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"

	chi "github.com/go-chi/chi/v5"
	"github.com/titpetric/platform"
	"github.com/titpetric/vuego"
	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/vuego-cli/basecoat"
)

//go:embed templates
var embeddedTemplates embed.FS

// Module represents the docs module for the platform.
type Module struct {
	platform.UnimplementedModule

	vuego vuego.Template

	FS        fs.FS
	indexTmpl string
}

// handler wraps an error-returning handler function with platform error handling.
func handler(fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			status := http.StatusInternalServerError
			var statusErr statusError
			if errors.As(err, &statusErr) {
				status = statusErr.status
			}
			platform.Error(w, r, status, err)
		}
	}
}

type statusError struct {
	status int
	err    error
}

func (e statusError) Error() string { return e.err.Error() }

func (e statusError) Unwrap() error { return e.err }

func notFound(err error) error {
	return statusError{status: http.StatusNotFound, err: err}
}

// NewModule creates a new docs module with a filesystem.
func NewModule(contentFS fs.FS) *Module {
	ofs := vuego.NewOverlayFS(contentFS, basecoat.FS)
	return &Module{
		FS:    ofs,
		vuego: vuego.NewFS(ofs, vuego.WithLessProcessor()),
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "vuego-docs"
}

// Mount registers the docs routes.
func (m *Module) Mount(_ context.Context, r platform.Router) error {
	// Load embedded index template
	indexData, err := embeddedTemplates.ReadFile("templates/index.vuego")
	if err != nil {
		return fmt.Errorf("loading index template: %w", err)
	}
	m.indexTmpl = string(indexData)

	r.Get("/", handler(m.serveIndex))
	r.Get("/assets/*", http.FileServer(http.FS(m.FS)).ServeHTTP)
	r.Get("/*", handler(m.serveDoc))

	return nil
}

func (m *Module) serveIndex(w http.ResponseWriter, r *http.Request) error {
	// Check if README.md exists
	if readme, err := fs.ReadFile(m.FS, "README.md"); err == nil {
		return m.renderDoc(r.Context(), w, "README.md", string(readme))
	}

	// Otherwise show directory listing
	return m.renderDirListing(r.Context(), w, ".")
}

func (m *Module) serveDoc(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	urlPath := chi.URLParam(r, "*")
	urlPath = strings.TrimSuffix(urlPath, "/")
	if urlPath == "" {
		urlPath = "."
	}

	// Try .vuego file first (exact match)
	if strings.HasSuffix(urlPath, ".vuego") {
		if _, err := fs.Stat(m.FS, urlPath); err != nil {
			return notFound(err)
		}
		return m.renderVuego(ctx, w, urlPath)
	}

	// Try .md file
	mdPath := urlPath
	if !strings.HasSuffix(mdPath, ".md") && !strings.HasSuffix(mdPath, ".vuego") {
		mdPath = urlPath + ".md"
	}

	if strings.HasSuffix(mdPath, ".md") {
		if content, err := fs.ReadFile(m.FS, mdPath); err == nil {
			return m.renderDoc(ctx, w, mdPath, string(content))
		}
	}

	// Try as directory with README.md
	if readme, err := fs.ReadFile(m.FS, path.Join(urlPath, "README.md")); err == nil {
		return m.renderDoc(ctx, w, path.Join(urlPath, "README.md"), string(readme))
	}

	// Try directory listing
	if entries, err := fs.ReadDir(m.FS, urlPath); err == nil && len(entries) > 0 {
		return m.renderDirListing(ctx, w, urlPath)
	}

	return notFound(fmt.Errorf("not found: %s", urlPath))
}

// DocMeta represents frontmatter metadata for a doc.
type DocMeta struct {
	Title    string `yaml:"title"`
	Subtitle string `yaml:"subtitle"`
	Layout   string `yaml:"layout"`
}

func (m *Module) renderDoc(ctx context.Context, w http.ResponseWriter, docPath string, content string) error {
	// Parse frontmatter
	meta, body, err := parseFrontmatter(content)
	if err != nil {
		return fmt.Errorf("parsing doc: %w", err)
	}

	// Get directory for relative file lookups
	docDir := path.Dir(docPath)

	// Build HTML directly to preserve DOCTYPE, html, head, body tags
	// Start with global data from data/*.yml files, then add doc-specific data
	data := map[string]any{
		"title":       meta.Title,
		"subtitle":    meta.Subtitle,
		"description": meta.Subtitle,
		"content":     m.parseDirectives(ctx, body, docDir),
	}

	m.fill(&data)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// compute layout path for the doc
	layoutName := "page"
	if meta.Layout != "" {
		layoutName = meta.Layout
	}
	layout := layoutName
	if !strings.Contains(layoutName, ".vuego") {
		layout = "layouts/" + layout + ".vuego"
	}

	var buf bytes.Buffer
	if err := m.vuego.Load(layout).Fill(data).Render(ctx, &buf); err != nil {
		return fmt.Errorf("rendering layout: %w", err)
	}

	_, _ = w.Write(buf.Bytes())
	return nil
}

func (m *Module) fill(dest *map[string]any) {
	files, err := fs.Glob(m.FS, "data/*.yml")
	if err != nil {
		return
	}

	for _, filename := range files {
		m.scan(dest, filename)
	}
}

func (m *Module) scan(dest *map[string]any, filename string) {
	content, err := fs.ReadFile(m.FS, filename)
	if err != nil {
		return
	}

	_ = yaml.Unmarshal(content, dest)
}

func (m *Module) renderVuego(ctx context.Context, w http.ResponseWriter, filePath string) error {
	// Load sidecar data file
	baseName := strings.TrimSuffix(filePath, filepath.Ext(filePath))

	var data map[string]any
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		m.scan(&data, baseName+ext)
	}

	m.fill(&data)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var buf bytes.Buffer
	if err := m.vuego.Load(filePath).Fill(data).Render(ctx, &buf); err != nil {
		return fmt.Errorf("rendering vuego: %w", err)
	}

	_, _ = w.Write(buf.Bytes())
	return nil
}

func (m *Module) renderDirListing(ctx context.Context, w http.ResponseWriter, dir string) error {
	entries, err := fs.ReadDir(m.FS, dir)
	if err != nil {
		return fmt.Errorf("listing directory: %w", err)
	}

	type Entry struct {
		Name  string
		Path  string
		IsDir bool
	}

	var items []Entry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		entryPath := path.Join(dir, name)
		if dir == "." {
			entryPath = name
		}
		items = append(items, Entry{
			Name:  name,
			Path:  "/" + entryPath,
			IsDir: e.IsDir(),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return items[i].Name < items[j].Name
	})

	title := dir
	if dir == "." {
		title = "Documentation"
	}

	data := map[string]any{
		"title":   title,
		"entries": items,
		"path":    dir,
	}
	m.fill(&data)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Render the inline template content (without layout processing)
	var contentBuf bytes.Buffer
	if err := m.vuego.New().Fill(data).RenderString(ctx, &contentBuf, m.indexTmpl); err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}

	// Wrap in the base layout
	data["content"] = contentBuf.String()
	var buf bytes.Buffer
	if err := m.vuego.Load("layouts/page.vuego").Fill(data).Render(ctx, &buf); err != nil {
		return fmt.Errorf("rendering layout: %w", err)
	}

	_, _ = buf.WriteTo(w)
	return nil
}

func (m *Module) readFile(docDir, filePath string) string {
	fullPath := path.Join(docDir, filePath)
	content, err := fs.ReadFile(m.FS, fullPath)
	if err != nil {
		return fmt.Sprintf("<!-- error reading %s: %v -->", filePath, err)
	}
	return string(content)
}

func (m *Module) renderVuegoFile(ctx context.Context, docDir, filePath string) string {
	fullPath := path.Join(docDir, filePath)

	// Load sidecar data file, starting with global data
	baseName := strings.TrimSuffix(fullPath, filepath.Ext(fullPath))
	var data map[string]any
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		dataPath := baseName + ext
		m.scan(&data, dataPath)
	}
	m.fill(&data)

	var buf bytes.Buffer
	if err := m.vuego.Load(fullPath).Fill(data).Render(ctx, &buf); err != nil {
		return fmt.Sprintf("<!-- render error: %v -->", err)
	}

	return buf.String()
}
