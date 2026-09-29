package docs

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/titpetric/vuego/markdown"
	yaml "gopkg.in/yaml.v3"
)

var tabGroupCounter int

// TabGroup represents a group of tabs.
type TabGroup struct {
	Tabs []Tab
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
