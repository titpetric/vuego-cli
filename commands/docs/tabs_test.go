package docs_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	chi "github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

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

func TestServeDoc_ImageAssets(t *testing.T) {
	pngData := []byte{0x89, 'P', 'N', 'G'}

	tests := []struct {
		name        string
		path        string
		file        string
		contentType string
	}{
		{"png", "/reference/schema/minimal.png", "reference/schema/minimal.png", "image/png"},
		{"jpg", "/photos/hero.jpg", "photos/hero.jpg", "image/jpeg"},
		{"gif", "/icons/loading.gif", "icons/loading.gif", "image/gif"},
		{"svg", "/icons/logo.svg", "icons/logo.svg", "image/svg+xml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentFS := fstest.MapFS{
				tt.file: &fstest.MapFile{Data: pngData},
			}

			m := docs.NewModule(contentFS)

			r := chi.NewRouter()
			r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
				docs.ServeDocHandler(m)(w, r)
			})

			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestServeDoc_ImageDuplicateSegment(t *testing.T) {
	// Image at reference/schema/minimal.png served via /reference/schema/schema/minimal.png
	// (browser resolves ./schema/minimal.png relative to /reference/schema/)
	pngData := []byte{0x89, 'P', 'N', 'G'}
	contentFS := fstest.MapFS{
		"reference/schema/minimal.png": &fstest.MapFile{Data: pngData},
	}

	m := docs.NewModule(contentFS)

	r := chi.NewRouter()
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		docs.ServeDocHandler(m)(w, r)
	})

	// Direct path works
	req := httptest.NewRequest("GET", "/reference/schema/minimal.png", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Duplicated segment path also works
	req = httptest.NewRequest("GET", "/reference/schema/schema/minimal.png", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestServeDoc_ImageNotFound(t *testing.T) {
	contentFS := fstest.MapFS{}
	m := docs.NewModule(contentFS)

	r := chi.NewRouter()
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		docs.ServeDocHandler(m)(w, r)
	})

	req := httptest.NewRequest("GET", "/missing.png", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.NotEqual(t, http.StatusOK, rec.Code)
}

func TestDocDir(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, ".", docs.DocDir(ctx))

	ctx = docs.WithDocDir(ctx, "components")
	require.Equal(t, "components", docs.DocDir(ctx))
}
