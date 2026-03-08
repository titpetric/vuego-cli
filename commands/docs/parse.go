package docs

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/titpetric/vuego/markdown"
)

// DirectivesHandler returns a handler for paragraphs containing @ directives.
// It detects lines starting with @ and processes them as directives,
// maintaining compatibility with the legacy @tabs, @render, @file, @example syntax.
func (m *Module) DirectivesHandler() markdown.Handler {
	return func(ctx context.Context, n *markdown.Node) (string, bool) {
		if !strings.HasPrefix(n.Raw, "@") {
			return "", false
		}

		docDir := DocDir(ctx)
		lines := strings.Split(n.Raw, "\n")

		var result []string
		var currentTabs *TabGroup
		inTabsBlock := false

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Handle @tabs directive - starts a tab group
			if trimmed == "@tabs" {
				inTabsBlock = true
				currentTabs = &TabGroup{}
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
				if inTabsBlock && currentTabs != nil {
					currentTabs.Tabs = append(currentTabs.Tabs, tabs.Tabs...)
				} else {
					result = append(result, m.renderTabGroup(tabs))
				}
				continue
			}

			// Non-directive line inside tabs block ends it
			if inTabsBlock && currentTabs != nil && len(currentTabs.Tabs) > 0 {
				result = append(result, m.renderTabGroup(currentTabs))
				inTabsBlock = false
				currentTabs = nil
			}
		}

		// Flush any remaining tabs
		if currentTabs != nil && len(currentTabs.Tabs) > 0 {
			result = append(result, m.renderTabGroup(currentTabs))
		}

		return strings.Join(result, "\n"), true
	}
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
