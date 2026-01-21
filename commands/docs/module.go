package docs

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"

	chi "github.com/go-chi/chi/v5"
	"github.com/titpetric/platform"
	"github.com/titpetric/vuego"
	yaml "gopkg.in/yaml.v3"
)

//go:embed templates
var embeddedTemplates embed.FS

// Module represents the docs module for the platform.
type Module struct {
	platform.UnimplementedModule

	vuego vuego.Template
	data  map[string]any

	FS        fs.FS
	indexTmpl string
	docTmpl   string
}

// NewModule creates a new docs module with a filesystem.
func NewModule(contentFS fs.FS) *Module {
	return &Module{
		FS:    contentFS,
		vuego: vuego.NewFS(contentFS, vuego.WithLessProcessor(), vuego.WithComponents()),
		data:  make(map[string]any),
	}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "vuego-docs"
}

// Mount registers the docs routes.
func (m *Module) Mount(_ context.Context, r platform.Router) error {
	// Load shared vuego data state
	m.fill(&m.data)
	m.vuego.Fill(m.data)

	// Load templates
	indexData, err := embeddedTemplates.ReadFile("templates/index.vuego")
	if err != nil {
		return fmt.Errorf("loading index template: %w", err)
	}
	m.indexTmpl = string(indexData)

	docData, err := embeddedTemplates.ReadFile("templates/doc.vuego")
	if err != nil {
		return fmt.Errorf("loading doc template: %w", err)
	}
	m.docTmpl = string(docData)

	r.Get("/", m.serveIndex)
	//r.Get("/assets/*", StaticFS(m.FS))
	r.Get("/assets/*", http.FileServer(http.FS(m.FS)).ServeHTTP)
	r.Get("/*", m.serveDoc)

	return nil
}

func (m *Module) serveIndex(w http.ResponseWriter, r *http.Request) {
	// Check if README.md exists
	if readme, err := fs.ReadFile(m.FS, "README.md"); err == nil {
		m.renderDoc(r.Context(), w, "README.md", string(readme))
		return
	}

	// Otherwise show directory listing
	m.renderDirListing(r.Context(), w, ".")
}

func (m *Module) serveDoc(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	urlPath := chi.URLParam(r, "*")
	if urlPath == "" {
		urlPath = "."
	}

	// Try .vuego file first (exact match)
	if strings.HasSuffix(urlPath, ".vuego") {
		if content, err := fs.ReadFile(m.FS, urlPath); err == nil {
			m.renderVuego(ctx, w, urlPath, string(content))
		}
		return
	}

	// Try .md file
	mdPath := urlPath
	if !strings.HasSuffix(mdPath, ".md") && !strings.HasSuffix(mdPath, ".vuego") {
		mdPath = urlPath + ".md"
	}

	if strings.HasSuffix(mdPath, ".md") {
		if content, err := fs.ReadFile(m.FS, mdPath); err == nil {
			m.renderDoc(ctx, w, mdPath, string(content))
			return
		}
	}

	// Try as directory with README.md
	if readme, err := fs.ReadFile(m.FS, path.Join(urlPath, "README.md")); err == nil {
		m.renderDoc(ctx, w, path.Join(urlPath, "README.md"), string(readme))
		return
	}

	// Try directory listing
	if entries, err := fs.ReadDir(m.FS, urlPath); err == nil && len(entries) > 0 {
		m.renderDirListing(ctx, w, urlPath)
		return
	}

	http.NotFound(w, r)
}

// DocMeta represents frontmatter metadata for a doc.
type DocMeta struct {
	Title    string `yaml:"title"`
	Subtitle string `yaml:"subtitle"`
	Layout   string `yaml:"layout"`
}

func (m *Module) renderDoc(ctx context.Context, w http.ResponseWriter, docPath string, content string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Parse frontmatter
	meta, body, err := parseFrontmatter(content)
	if err != nil {
		http.Error(w, "Error parsing doc: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get directory for relative file lookups
	docDir := path.Dir(docPath)

	// Build HTML directly to preserve DOCTYPE, html, head, body tags
	// Start with global data from data/*.yml files, then add doc-specific data
	data := maps.Clone(m.data)
	data["title"] = meta.Title
	data["subtitle"] = meta.Subtitle
	data["description"] = meta.Subtitle
	data["content"] = m.parseDirectives(ctx, body, docDir)

	var buf bytes.Buffer
	if err := m.vuego.Load("layouts/page.vuego").Fill(data).Render(ctx, &buf); err != nil {
		http.Error(w, "Error rendering layout: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(buf.Bytes())
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

func (m *Module) renderVuego(ctx context.Context, w http.ResponseWriter, filePath string, content string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Load sidecar data file
	baseName := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	var data map[string]any
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		m.scan(&data, baseName+ext)
	}

	// Render the vuego template
	var buf bytes.Buffer
	if err := m.vuego.New().Fill(data).RenderString(ctx, &buf, content); err != nil {
		http.Error(w, "Error rendering vuego: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(buf.Bytes())
}

func (m *Module) renderDirListing(ctx context.Context, w http.ResponseWriter, dir string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	entries, err := fs.ReadDir(m.FS, dir)
	if err != nil {
		http.Error(w, "Cannot list directory", http.StatusInternalServerError)
		return
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

	var buf bytes.Buffer
	if err := m.vuego.New().Fill(data).RenderString(ctx, &buf, m.indexTmpl); err != nil {
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = buf.WriteTo(w)
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
	content, err := fs.ReadFile(m.FS, fullPath)
	if err != nil {
		return fmt.Sprintf("<!-- error reading %s: %v -->", filePath, err)
	}

	// Load sidecar data file
	baseName := strings.TrimSuffix(fullPath, filepath.Ext(fullPath))
	var data map[string]any
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		dataPath := baseName + ext
		m.scan(&data, dataPath)
	}

	// Render vuego template
	var buf bytes.Buffer
	if err := m.vuego.New().Fill(data).RenderString(ctx, &buf, string(content)); err != nil {
		return fmt.Sprintf("<!-- render error: %v -->", err)
	}

	return buf.String()
}
