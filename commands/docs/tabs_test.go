package docs_test

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/titpetric/vuego/markdown"

	"github.com/titpetric/vuego-cli/commands/docs"
)

func TestTabsHandler_RenderAndFile(t *testing.T) {
	contentFS := fstest.MapFS{
		"components/page.md": &fstest.MapFile{
			Data: []byte("---\ntitle: Alert\n---\n\n```tabs\n- label: Preview\n  type: render\n  src: demo.vuego\n- label: Code\n  type: file\n  src: demo.vuego\n- label: Data\n  type: file\n  src: demo.yml\n```\n"),
		},
		"components/demo.vuego": &fstest.MapFile{
			Data: []byte(`<div class="alert" v-html="title"></div>`),
		},
		"components/demo.yml": &fstest.MapFile{
			Data: []byte("title: Hello World"),
		},
	}

	m := docs.NewModule(contentFS)
	doc, err := m.LoadMarkdown("components/page.md")
	require.NoError(t, err)

	ctx := docs.WithDocDir(context.Background(), "components")
	var buf strings.Builder
	err = doc.RenderContext(ctx, &buf)
	require.NoError(t, err)

	result := buf.String()

	// Tab structure
	require.Contains(t, result, `role="tablist"`)
	require.Contains(t, result, `role="tabpanel"`)
	require.Contains(t, result, "Preview")
	require.Contains(t, result, "Code")
	require.Contains(t, result, "Data")

	// Preview tab renders vuego with data interpolation
	require.Contains(t, result, "Hello World")

	// Code tab shows raw vuego source escaped
	require.Contains(t, result, `v-html=&#34;title&#34;`)
	require.Contains(t, result, "language-html")

	// Data tab shows raw YAML with yaml mode
	require.Contains(t, result, "title: Hello World")
	require.Contains(t, result, "language-yaml")
}

func TestTabsHandler_Example(t *testing.T) {
	contentFS := fstest.MapFS{
		"components/page.md": &fstest.MapFile{
			Data: []byte("---\ntitle: Test\n---\n\n```tabs\n- type: example\n  src: alert.vuego\n```\n"),
		},
		"components/alert.vuego": &fstest.MapFile{
			Data: []byte(`<div class="alert">{{ message }}</div>`),
		},
		"components/alert.yml": &fstest.MapFile{
			Data: []byte("message: Something happened"),
		},
	}

	m := docs.NewModule(contentFS)
	doc, err := m.LoadMarkdown("components/page.md")
	require.NoError(t, err)

	ctx := docs.WithDocDir(context.Background(), "components")
	var buf strings.Builder
	err = doc.RenderContext(ctx, &buf)
	require.NoError(t, err)

	result := buf.String()

	// Produces Preview + Code tabs automatically
	require.Contains(t, result, `role="tablist"`)
	require.Contains(t, result, "Preview")
	require.Contains(t, result, "Code")

	// Preview renders the vuego template with data
	require.Contains(t, result, "Something happened")

	// Code tab shows raw source as html mode
	require.Contains(t, result, "language-html")
	require.Contains(t, result, "{{ message }}")
}

func TestTabsHandler_ExampleWithConditional(t *testing.T) {
	contentFS := fstest.MapFS{
		"page.md": &fstest.MapFile{
			Data: []byte("---\ntitle: Test\n---\n\n```tabs\n- type: example\n  src: widget.vuego\n```\n"),
		},
		"widget.vuego": &fstest.MapFile{
			Data: []byte(`<div><h2>{{ title }}</h2><section v-if="description">{{ description }}</section></div>`),
		},
		"widget.yml": &fstest.MapFile{
			Data: []byte("title: Alert Title\ndescription: Alert Description"),
		},
	}

	m := docs.NewModule(contentFS)
	doc, err := m.LoadMarkdown("page.md")
	require.NoError(t, err)

	ctx := docs.WithDocDir(context.Background(), ".")
	var buf strings.Builder
	err = doc.RenderContext(ctx, &buf)
	require.NoError(t, err)

	result := buf.String()

	// v-if="description" should render section since description is set
	require.Contains(t, result, "Alert Title")
	require.Contains(t, result, "Alert Description")

	// Code tab has raw vuego source with v-if
	require.Contains(t, result, "v-if=")
}

func TestTabsHandler_IgnoresNonTabs(t *testing.T) {
	contentFS := fstest.MapFS{
		"test.md": &fstest.MapFile{
			Data: []byte("---\ntitle: Test\n---\n\n```html\n<div>hello</div>\n```\n"),
		},
	}

	m := docs.NewModule(contentFS)
	doc, err := m.LoadMarkdown("test.md")
	require.NoError(t, err)

	ctx := docs.WithDocDir(context.Background(), ".")
	var buf strings.Builder
	err = doc.RenderContext(ctx, &buf)
	require.NoError(t, err)

	result := buf.String()
	require.NotContains(t, result, `role="tablist"`)
	require.Contains(t, result, `&lt;div&gt;hello&lt;/div&gt;`)
}

func TestTabsHandler_InvalidYAML(t *testing.T) {
	contentFS := fstest.MapFS{
		"test.md": &fstest.MapFile{
			Data: []byte("---\ntitle: Test\n---\n\n```tabs\n[not valid yaml\n```\n"),
		},
	}

	m := docs.NewModule(contentFS)
	doc, err := m.LoadMarkdown("test.md")
	require.NoError(t, err)

	ctx := docs.WithDocDir(context.Background(), ".")
	var buf strings.Builder
	err = doc.RenderContext(ctx, &buf)
	require.NoError(t, err)

	result := buf.String()
	require.Contains(t, result, "<!-- tabs error:")
}

func TestTabsHandler_FileMode(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		mode     string
	}{
		{"vuego as html", "template.vuego", "language-html"},
		{"yml as yaml", "data.yml", "language-yaml"},
		{"json as json", "data.json", "language-json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentFS := fstest.MapFS{
				"test.md": &fstest.MapFile{
					Data: []byte("---\ntitle: Test\n---\n\n```tabs\n- label: Source\n  type: file\n  src: " + tt.filename + "\n```\n"),
				},
				tt.filename: &fstest.MapFile{
					Data: []byte("content"),
				},
			}

			m := docs.NewModule(contentFS)
			doc, err := m.LoadMarkdown("test.md")
			require.NoError(t, err)

			ctx := docs.WithDocDir(context.Background(), ".")
			var buf strings.Builder
			err = doc.RenderContext(ctx, &buf)
			require.NoError(t, err)

			require.Contains(t, buf.String(), tt.mode)
		})
	}
}

func TestTabsHandler_UnknownType(t *testing.T) {
	contentFS := fstest.MapFS{
		"test.md": &fstest.MapFile{
			Data: []byte("---\ntitle: Test\n---\n\n```tabs\n- label: Foo\n  type: unknown\n  src: foo.txt\n```\n"),
		},
	}

	m := docs.NewModule(contentFS)
	doc, err := m.LoadMarkdown("test.md")
	require.NoError(t, err)

	ctx := docs.WithDocDir(context.Background(), ".")
	var buf strings.Builder
	err = doc.RenderContext(ctx, &buf)
	require.NoError(t, err)

	require.Contains(t, buf.String(), "<!-- unknown type: unknown -->")
}

// Tests for @ directive paragraph handling (legacy syntax)

// TestTabGroup covers the group being what a rendered fence becomes: one
// panel per tab, the first one selected and the rest hidden, which is the
// whole of the widget's initial state.
func TestTabGroup(t *testing.T) {
	m := docs.NewModule(fstest.MapFS{
		"demo.vuego": &fstest.MapFile{Data: []byte(`<p>hi</p>`)},
	})

	out, handled := m.TabsHandler()(context.Background(), &markdown.Node{
		Type:     markdown.NodeCodeBlock,
		Language: "tabs",
		Raw: `- label: One
  type: render
  src: demo.vuego
- label: Two
  type: file
  src: demo.vuego
`,
	})

	require.True(t, handled)
	require.Equal(t, 2, strings.Count(out, `role="tab"`), "one button per tab")
	require.Equal(t, 2, strings.Count(out, `role="tabpanel"`), "one panel per tab")
	require.Equal(t, 1, strings.Count(out, `aria-selected="true"`), "exactly one tab starts selected")
	require.Equal(t, 1, strings.Count(out, " hidden>"), "every panel but the first starts hidden")

	var group docs.TabGroup
	require.Empty(t, group.Tabs, "the zero group holds no tabs")

	empty, handled := m.TabsHandler()(context.Background(), &markdown.Node{
		Type:     markdown.NodeCodeBlock,
		Language: "tabs",
		Raw:      "[]",
	})
	require.True(t, handled)
	require.Equal(t, "", empty, "a group with no tabs renders nothing")
}
