package docs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// Tab represents a single tab.
type Tab struct {
	Label   string
	Content string
	IsCode  bool
	Mode    string // Ace editor mode (html, yaml, json, etc.)
}

// TabConfig represents a tab definition in YAML.
type TabConfig struct {
	Label string `yaml:"label"`
	Type  string `yaml:"type"` // "render", "file", "example"
	Src   string `yaml:"src"`
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
