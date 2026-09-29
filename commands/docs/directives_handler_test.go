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

func TestDirectivesHandler_Tabs(t *testing.T) {
	contentFS := fstest.MapFS{
		"components/page.md": &fstest.MapFile{
			Data: []byte("---\ntitle: Alert\n---\n\n@tabs\n@render \"Preview\" demo.vuego\n@file \"Code\" demo.vuego\n@file \"Data\" demo.yml\n"),
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

func TestDirectivesHandler_Example(t *testing.T) {
	contentFS := fstest.MapFS{
		"components/page.md": &fstest.MapFile{
			Data: []byte("---\ntitle: Test\n---\n\n@example alert.vuego\n"),
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

func TestDirectivesHandler_StandaloneRender(t *testing.T) {
	contentFS := fstest.MapFS{
		"page.md": &fstest.MapFile{
			Data: []byte("---\ntitle: Test\n---\n\n@render \"Preview\" widget.vuego\n"),
		},
		"widget.vuego": &fstest.MapFile{
			Data: []byte(`<div>{{ title }}</div>`),
		},
		"widget.yml": &fstest.MapFile{
			Data: []byte("title: Standalone"),
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

	// Single render without tabs should show preview div
	require.Contains(t, result, "Standalone")
	require.Contains(t, result, `class="preview`)
	// Should NOT have tablist since it's standalone
	require.NotContains(t, result, `role="tablist"`)
}

func TestDirectivesHandler_StandaloneFile(t *testing.T) {
	contentFS := fstest.MapFS{
		"page.md": &fstest.MapFile{
			Data: []byte("---\ntitle: Test\n---\n\n@file \"Source\" data.yml\n"),
		},
		"data.yml": &fstest.MapFile{
			Data: []byte("key: value"),
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

	require.Contains(t, result, "key: value")
	require.Contains(t, result, "language-yaml")
	require.NotContains(t, result, `role="tablist"`)
}

func TestDirectivesHandler_NonDirectiveParagraph(t *testing.T) {
	contentFS := fstest.MapFS{
		"page.md": &fstest.MapFile{
			Data: []byte("---\ntitle: Test\n---\n\nThis is a normal paragraph.\n"),
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

	// Normal paragraph should be rendered as-is
	require.Contains(t, result, "This is a normal paragraph.")
	require.NotContains(t, result, `role="tablist"`)
}

// TestModule_DirectivesHandler covers the handler only claiming paragraphs it
// owns. It is registered for every paragraph in the document, so anything not
// starting with @ has to fall through to the default rendering.
func TestModule_DirectivesHandler(t *testing.T) {
	m := docs.NewModule(fstest.MapFS{
		"demo.vuego": &fstest.MapFile{Data: []byte(`<p>rendered</p>`)},
	})
	handler := m.DirectivesHandler()

	tests := []struct {
		name        string
		raw         string
		wantHandled bool
		wantBody    string
	}{
		{"a render directive", `@render "Preview" demo.vuego`, true, "rendered"},
		{"a file directive", `@file "Code" demo.vuego`, true, ""},
		{"ordinary prose", "Just a paragraph.", false, ""},
		{"an email address", "Write to me@example.com.", false, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out, handled := handler(context.Background(), &markdown.Node{
				Type: markdown.NodeParagraph,
				Raw:  test.raw,
			})

			require.Equal(t, test.wantHandled, handled)
			if test.wantBody != "" {
				require.Contains(t, out, test.wantBody)
			}
		})
	}
}
