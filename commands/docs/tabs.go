package docs

import (
	"context"
	"fmt"
	"html"
	"path/filepath"
	"strings"

	"github.com/titpetric/vuego/markdown"
	yaml "gopkg.in/yaml.v3"
)

var tabGroupCounter int

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

func (m *Module) renderTabGroup(tg *TabGroup) string {
	if len(tg.Tabs) == 0 {
		return ""
	}

	tabGroupCounter++
	groupID := tabGroupCounter

	var sb strings.Builder
	sb.WriteString(`<div class="relative my-6">`)
	sb.WriteString(`<div class="ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 relative rounded-md border">`)
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

		sb.WriteString(m.renderSingleTab(tab))

		sb.WriteString(`</section>`)
	}
	sb.WriteString(`</div>`)
	sb.WriteString(`</div>`)
	sb.WriteString(`</div>`)

	return sb.String()
}

func (m *Module) renderSingleTab(tab Tab) string {
	if tab.IsCode {
		mode := tab.Mode
		if mode == "" {
			mode = "text"
		}
		var buf strings.Builder
		data := map[string]any{
			"code": tab.Content,
			"mode": mode,
		}
		if err := m.vuego.Load("templates/code_tab.vuego").Fill(data).Render(context.Background(), &buf); err != nil {
			return fmt.Sprintf("<!-- code tab error: %v -->", err)
		}
		return buf.String()
	}
	return fmt.Sprintf(`<div class="preview flex min-h-[150px] max-h-[650px] w-full justify-center p-10 items-center">%s</div>`, tab.Content)
}

// TabConfig represents a tab definition in YAML.
type TabConfig struct {
	Label string `yaml:"label"`
	Type  string `yaml:"type"` // "render", "file", "example"
	Src   string `yaml:"src"`
}

type docDirKey struct{}

// WithDocDir returns a context with the document directory.
func WithDocDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, docDirKey{}, dir)
}

// DocDir returns the document directory from context.
func DocDir(ctx context.Context) string {
	if v, ok := ctx.Value(docDirKey{}).(string); ok {
		return v
	}
	return "."
}

// TabsHandler returns a handler for ```tabs code blocks.
func (m *Module) TabsHandler() markdown.Handler {
	return func(ctx context.Context, n *markdown.Node) (string, bool) {
		if n.Language != "tabs" {
			return "", false
		}

		docDir := DocDir(ctx)

		var configs []TabConfig
		if err := yaml.Unmarshal([]byte(n.Raw), &configs); err != nil {
			return fmt.Sprintf("<!-- tabs error: %v -->", err), true
		}

		tg := &TabGroup{}
		for _, cfg := range configs {
			tabs := m.buildTab(ctx, cfg, docDir)
			tg.Tabs = append(tg.Tabs, tabs...)
		}

		return m.renderTabGroup(tg), true
	}
}

func (m *Module) buildTab(ctx context.Context, cfg TabConfig, docDir string) []Tab {
	switch cfg.Type {
	case "render":
		content := m.renderVuegoFile(ctx, docDir, cfg.Src)
		return []Tab{{Label: cfg.Label, Content: content, IsCode: false}}
	case "file":
		content := m.readFile(docDir, cfg.Src)
		ext := filepath.Ext(cfg.Src)
		mode := strings.TrimPrefix(ext, ".")
		if mode == "vuego" {
			mode = "html"
		} else if mode == "yml" {
			mode = "yaml"
		}
		return []Tab{{Label: cfg.Label, Content: content, IsCode: true, Mode: mode}}
	case "example":
		rendered := m.renderVuegoFile(ctx, docDir, cfg.Src)
		code := m.readFile(docDir, cfg.Src)
		return []Tab{
			{Label: "Preview", Content: rendered, IsCode: false},
			{Label: "Code", Content: code, IsCode: true, Mode: "html"},
		}
	default:
		return []Tab{{Label: cfg.Label, Content: fmt.Sprintf("<!-- unknown type: %s -->", cfg.Type)}}
	}
}
