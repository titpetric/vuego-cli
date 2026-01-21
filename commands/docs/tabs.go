package docs

import (
	"fmt"
	"html"
	"strings"
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
		return fmt.Sprintf(
			`<code class="hljs language-%s"><pre>%s</pre></code>`,
			html.EscapeString(mode),
			html.EscapeString(tab.Content),
		)
	}
	return fmt.Sprintf(`<div class="preview flex min-h-[350px] w-full justify-center p-10 items-center">%s</div>`, tab.Content)
}
