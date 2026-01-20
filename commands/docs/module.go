package docs

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html"
	"io/fs"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"

	chi "github.com/go-chi/chi/v5"
	blackfriday "github.com/russross/blackfriday/v2"
	"github.com/titpetric/lessgo"
	"github.com/titpetric/platform"
	"github.com/titpetric/vuego"
	yaml "gopkg.in/yaml.v3"
)

//go:embed templates
var embeddedTemplates embed.FS

// Module represents the docs module for the platform.
type Module struct {
	platform.UnimplementedModule

	contentFS   fs.FS
	contentPath string
	indexTmpl   string
	docTmpl     string
}

// NewModule creates a new docs module with a filesystem.
func NewModule(contentFS fs.FS, contentPath string) *Module {
	return &Module{contentFS: contentFS, contentPath: contentPath}
}

// Name returns the module name.
func (m *Module) Name() string {
	return "vuego-docs"
}

// Mount registers the docs routes.
func (m *Module) Mount(_ context.Context, r platform.Router) error {
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
	r.Get("/static/docs.less", m.serveLess)
	r.Get("/*", m.serveDoc)

	return nil
}

func (m *Module) serveLess(w http.ResponseWriter, r *http.Request) {
	sub, _ := fs.Sub(embeddedTemplates, "templates")
	handler := lessgo.NewHandler(sub, "/")
	r.URL.Path = "/docs.less"
	handler.ServeHTTP(w, r)
}

func (m *Module) serveIndex(w http.ResponseWriter, r *http.Request) {
	// Check if README.md exists
	if readme, err := fs.ReadFile(m.contentFS, "README.md"); err == nil {
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
		if content, err := fs.ReadFile(m.contentFS, urlPath); err == nil {
			m.renderVuego(ctx, w, urlPath, string(content))
			return
		}
	}

	// Try .md file
	mdPath := urlPath
	if !strings.HasSuffix(mdPath, ".md") && !strings.HasSuffix(mdPath, ".vuego") {
		mdPath = urlPath + ".md"
	}

	if strings.HasSuffix(mdPath, ".md") {
		if content, err := fs.ReadFile(m.contentFS, mdPath); err == nil {
			m.renderDoc(ctx, w, mdPath, string(content))
			return
		}
	}

	// Try as directory with README.md
	if readme, err := fs.ReadFile(m.contentFS, path.Join(urlPath, "README.md")); err == nil {
		m.renderDoc(ctx, w, path.Join(urlPath, "README.md"), string(readme))
		return
	}

	// Try directory listing
	if entries, err := fs.ReadDir(m.contentFS, urlPath); err == nil && len(entries) > 0 {
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
	meta, body := parseFrontmatter(content)

	renderMarkdown := func(in string) string {
		html := blackfriday.Run([]byte(in))
		return string(html)
	}

	// Get directory for relative file lookups
	docDir := path.Dir(docPath)

	// Process @ directives
	processedBody := m.processDirectives(ctx, renderMarkdown(body), docDir)

	// Build HTML directly to preserve DOCTYPE, html, head, body tags
	subtitle := ""
	if meta.Subtitle != "" {
		subtitle = fmt.Sprintf(`<p class="text-muted-foreground text-lg mt-2">%s</p>`, html.EscapeString(meta.Subtitle))
	}

	titleHeader := ""
	if meta.Title != "" {
		titleHeader = fmt.Sprintf(`<header class="mb-8"><h1 class="text-3xl font-bold">%s</h1>%s</header>`,
			html.EscapeString(meta.Title), subtitle)
	}

	var baseLayout = `---
layout: page
---

<div v-html="content"></div>
`

	var buf bytes.Buffer
	err := vuego.NewFS(m.contentFS).Assign("title", titleHeader).Assign("content", processedBody).RenderString(ctx, &buf, baseLayout)
	if err != nil {
		log.Println("vuego renderdoc error:", err)
	}

	_, _ = w.Write(buf.Bytes())
}

func (m *Module) renderVuego(ctx context.Context, w http.ResponseWriter, filePath string, content string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Load sidecar data file
	baseName := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	var data map[string]any
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		dataPath := baseName + ext
		if dataContent, err := fs.ReadFile(m.contentFS, dataPath); err == nil {
			_ = yaml.Unmarshal(dataContent, &data)
			break
		}
	}

	// Render the vuego template
	tmpl := vuego.NewFS(m.contentFS, vuego.WithLessProcessor(), vuego.WithComponents())
	var buf bytes.Buffer
	if err := tmpl.Fill(data).RenderString(ctx, &buf, content); err != nil {
		http.Error(w, "Error rendering vuego: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Wrap in basic HTML if not already a full document
	rendered := buf.String()
	if !strings.Contains(rendered, "<!DOCTYPE") && !strings.Contains(rendered, "<html") {
		rendered = m.wrapInHTML(filePath, rendered)
	}

	_, _ = w.Write([]byte(rendered))
}

func (m *Module) wrapInHTML(filePath, content string) string {
	title := filepath.Base(filePath)
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/basecoat-css@latest/dist/basecoat.cdn.min.css">
  <script src="https://cdn.jsdelivr.net/npm/basecoat-css@latest/dist/js/all.min.js" defer></script>
</head>
<body class="bg-background text-foreground p-8">
  <nav class="mb-4"><a href="/" class="text-muted-foreground hover:text-foreground text-sm">&larr; Back</a></nav>
  <div class="max-w-4xl">%s</div>
</body>
</html>`, html.EscapeString(title), content)
}

func (m *Module) renderDirListing(ctx context.Context, w http.ResponseWriter, dir string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	entries, err := fs.ReadDir(m.contentFS, dir)
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

	tmpl := vuego.New(vuego.WithLessProcessor())
	var buf bytes.Buffer
	if err := tmpl.Fill(data).RenderString(ctx, &buf, m.indexTmpl); err != nil {
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = buf.WriteTo(w)
}

func parseFrontmatter(content string) (DocMeta, string) {
	var meta DocMeta
	if !strings.HasPrefix(content, "---") {
		return meta, content
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return meta, content
	}

	_ = yaml.Unmarshal([]byte(parts[1]), &meta)
	return meta, strings.TrimSpace(parts[2])
}

// processDirectives processes @ directives in the markdown body.
func (m *Module) processDirectives(ctx context.Context, body, docDir string) string {
	lines := strings.Split(body, "\n")
	var result []string
	var currentTabs *TabGroup
	inTabsBlock := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Handle @tabs directive - starts a tab group
		if trimmed == "@tabs" {
			inTabsBlock = true
			currentTabs = &TabGroup{}
			continue
		}

		// If in tabs block and we hit a blank line, end the tabs block
		if inTabsBlock && trimmed == "" {
			if currentTabs != nil && len(currentTabs.Tabs) > 0 {
				result = append(result, m.renderTabGroup(currentTabs))
			}
			inTabsBlock = false
			currentTabs = nil
			result = append(result, line)
			continue
		}

		// Handle @render directive
		if strings.HasPrefix(trimmed, "@render ") {
			tab := m.parseRenderDirective(ctx, trimmed, docDir)
			if inTabsBlock && currentTabs != nil {
				currentTabs.Tabs = append(currentTabs.Tabs, tab)
			} else {
				result = append(result, m.renderSingleTab(tab))
			}
			continue
		}

		// Handle @file directive
		if strings.HasPrefix(trimmed, "@file ") {
			tab := m.parseFileDirective(trimmed, docDir)
			if inTabsBlock && currentTabs != nil {
				currentTabs.Tabs = append(currentTabs.Tabs, tab)
			} else {
				result = append(result, m.renderSingleTab(tab))
			}
			continue
		}

		// Handle @example directive
		if strings.HasPrefix(trimmed, "@example ") {
			tabs := m.parseExampleDirective(ctx, trimmed, docDir)
			result = append(result, m.renderTabGroup(tabs))
			continue
		}

		// Regular line
		if inTabsBlock && currentTabs != nil && len(currentTabs.Tabs) > 0 {
			// Flush tabs before continuing with normal content
			result = append(result, m.renderTabGroup(currentTabs))
			inTabsBlock = false
			currentTabs = nil
		}
		result = append(result, line)
	}

	// Flush any remaining tabs
	if currentTabs != nil && len(currentTabs.Tabs) > 0 {
		result = append(result, m.renderTabGroup(currentTabs))
	}

	return strings.Join(result, "\n")
}

// TabGroup represents a group of tabs.
type TabGroup struct {
	Tabs []Tab
}

// Tab represents a single tab.
type Tab struct {
	Label   string
	Content string
	IsCode  bool
	Mode    string // Ace editor mode (html, yaml, json, etc.)
}

func (m *Module) parseRenderDirective(ctx context.Context, line, docDir string) Tab {
	// @render "Label" file.vuego
	parts := parseDirectiveParts(strings.TrimPrefix(line, "@render "))
	if len(parts) < 2 {
		return Tab{Label: "Preview", Content: "<!-- missing args -->"}
	}
	label := parts[0]
	filePath := parts[1]

	rendered := m.renderVuegoFile(ctx, docDir, filePath)
	return Tab{Label: label, Content: rendered, IsCode: false}
}

func (m *Module) parseFileDirective(line, docDir string) Tab {
	// @file "Label" file.vuego
	parts := parseDirectiveParts(strings.TrimPrefix(line, "@file "))
	if len(parts) < 2 {
		return Tab{Label: "Code", Content: "<!-- missing args -->"}
	}
	label := parts[0]
	filePath := parts[1]

	content := m.readFile(docDir, filePath)
	ext := filepath.Ext(filePath)
	mode := strings.TrimPrefix(ext, ".")
	if mode == "vuego" {
		mode = "html"
	} else if mode == "yml" {
		mode = "yaml"
	}

	return Tab{Label: label, Content: content, IsCode: true, Mode: mode}
}

func (m *Module) parseExampleDirective(ctx context.Context, line, docDir string) *TabGroup {
	// @example file.vuego file.yaml
	parts := parseDirectiveParts(strings.TrimPrefix(line, "@example "))
	if len(parts) < 1 {
		return &TabGroup{}
	}

	vuegoPart := parts[0]
	rendered := m.renderVuegoFile(ctx, docDir, vuegoPart)
	code := m.readFile(docDir, vuegoPart)

	return &TabGroup{
		Tabs: []Tab{
			{Label: "Preview", Content: rendered, IsCode: false},
			{Label: "Code", Content: code, IsCode: true, Mode: "html"},
		},
	}
}

func parseDirectiveParts(s string) []string {
	var parts []string
	s = strings.TrimSpace(s)

	for len(s) > 0 {
		if s[0] == '"' {
			// Quoted string
			end := strings.Index(s[1:], "\"")
			if end == -1 {
				parts = append(parts, s[1:])
				break
			}
			parts = append(parts, s[1:end+1])
			s = strings.TrimSpace(s[end+2:])
		} else {
			// Unquoted
			end := strings.IndexAny(s, " \t")
			if end == -1 {
				parts = append(parts, s)
				break
			}
			parts = append(parts, s[:end])
			s = strings.TrimSpace(s[end:])
		}
	}
	return parts
}

func (m *Module) readFile(docDir, filePath string) string {
	fullPath := path.Join(docDir, filePath)
	content, err := fs.ReadFile(m.contentFS, fullPath)
	if err != nil {
		return fmt.Sprintf("<!-- error reading %s: %v -->", filePath, err)
	}
	return string(content)
}

func (m *Module) renderVuegoFile(ctx context.Context, docDir, filePath string) string {
	fullPath := path.Join(docDir, filePath)
	content, err := fs.ReadFile(m.contentFS, fullPath)
	if err != nil {
		return fmt.Sprintf("<!-- error reading %s: %v -->", filePath, err)
	}

	// Load sidecar data file
	baseName := strings.TrimSuffix(fullPath, filepath.Ext(fullPath))
	var data map[string]any
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		dataPath := baseName + ext
		if dataContent, err := fs.ReadFile(m.contentFS, dataPath); err == nil {
			_ = yaml.Unmarshal(dataContent, &data)
			break
		}
	}

	// Render vuego template
	tmpl := vuego.New(vuego.WithLessProcessor())
	var buf bytes.Buffer
	if err := tmpl.Fill(data).RenderString(ctx, &buf, string(content)); err != nil {
		return fmt.Sprintf("<!-- render error: %v -->", err)
	}

	return buf.String()
}

var tabGroupCounter int

func (m *Module) renderTabGroup(tg *TabGroup) string {
	if len(tg.Tabs) == 0 {
		return ""
	}

	tabGroupCounter++
	groupID := tabGroupCounter

	var sb strings.Builder
	sb.WriteString(`<div class="tabs">`)
	sb.WriteString(`<div role="tablist">`)

	for i, tab := range tg.Tabs {
		selected := "false"
		tabindex := "-1"
		if i == 0 {
			selected = "true"
			tabindex = "0"
		}
		sb.WriteString(fmt.Sprintf(
			`<button role="tab" aria-controls="panel-%d-%d" aria-selected="%s" tabindex="%s">%s</button>`,
			groupID, i, selected, tabindex, html.EscapeString(tab.Label),
		))
	}
	sb.WriteString(`</div>`)

	for i, tab := range tg.Tabs {
		hidden := ""
		if i > 0 {
			hidden = " hidden"
		}
		sb.WriteString(fmt.Sprintf(`<section id="panel-%d-%d" role="tabpanel"%s>`, groupID, i, hidden))
		if tab.IsCode {
			mode := tab.Mode
			if mode == "" {
				mode = "text"
			}
			sb.WriteString(fmt.Sprintf(
				`<div class="ace-editor" data-mode="%s">%s</div>`,
				html.EscapeString(mode),
				html.EscapeString(tab.Content),
			))
		} else {
			sb.WriteString(fmt.Sprintf(`<div class="preview">%s</div>`, tab.Content))
		}
		sb.WriteString(`</section>`)
	}
	sb.WriteString(`</div>`)

	return sb.String()
}

func (m *Module) renderSingleTab(tab Tab) string {
	if tab.IsCode {
		mode := tab.Mode
		if mode == "" {
			mode = "text"
		}
		return fmt.Sprintf(
			`<div class="ace-editor" data-mode="%s">%s</div>`,
			html.EscapeString(mode),
			html.EscapeString(tab.Content),
		)
	}
	return fmt.Sprintf(`<div class="preview">%s</div>`, tab.Content)
}
