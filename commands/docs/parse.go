package docs

import (
	"context"
	"path/filepath"
	"strings"

	blackfriday "github.com/russross/blackfriday/v2"
	yaml "gopkg.in/yaml.v3"
)

// parseDirectives parses @ directives in the markdown body.
// It processes directives on raw markdown, then renders markdown on non-directive content.
func (m *Module) parseDirectives(ctx context.Context, body, docDir string) string {
	lines := strings.Split(body, "\n")
	var result []string
	var currentTabs *TabGroup
	var markdownBuffer []string
	inTabsBlock := false

	flushMarkdown := func() {
		if len(markdownBuffer) > 0 {
			result = append(result, renderMarkdown(strings.Join(markdownBuffer, "\n")))
			markdownBuffer = nil
		}
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Handle @tabs directive - starts a tab group
		if trimmed == "@tabs" {
			flushMarkdown()
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
			markdownBuffer = append(markdownBuffer, line)
			continue
		}

		// Handle @render directive
		if strings.HasPrefix(trimmed, "@render ") {
			flushMarkdown()
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
			flushMarkdown()
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
			flushMarkdown()
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
		markdownBuffer = append(markdownBuffer, line)
	}

	// Flush any remaining markdown
	flushMarkdown()

	// Flush any remaining tabs
	if currentTabs != nil && len(currentTabs.Tabs) > 0 {
		result = append(result, m.renderTabGroup(currentTabs))
	}

	return strings.Join(result, "\n")
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

func parseFrontmatter(content string) (DocMeta, string, error) {
	var meta DocMeta
	if !strings.HasPrefix(content, "---") {
		return meta, content, nil
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return meta, content, nil
	}

	err := yaml.Unmarshal([]byte(parts[1]), &meta)
	return meta, strings.TrimSpace(parts[2]), err
}

func renderMarkdown(in string) string {
	return string(blackfriday.Run([]byte(in)))
}
