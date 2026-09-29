package docs_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/titpetric/vuego/markdown"
	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/vuego-cli/commands/docs"
)

// TestTabConfig covers the YAML a ```tabs fence is written in. The keys are
// content format, not an implementation detail: renaming one renames it in
// every document that already uses it.
func TestTabConfig(t *testing.T) {
	var configs []docs.TabConfig

	err := yaml.Unmarshal([]byte(`
- label: Preview
  type: render
  src: demo.vuego
- label: Code
  type: file
  src: demo.vuego
`), &configs)

	require.NoError(t, err)
	require.Len(t, configs, 2)
	require.Equal(t, docs.TabConfig{Label: "Preview", Type: "render", Src: "demo.vuego"}, configs[0])
	require.Equal(t, docs.TabConfig{Label: "Code", Type: "file", Src: "demo.vuego"}, configs[1])
}

// TestTab covers the zero value. IsCode decides whether a tab is rendered
// through the code editor template or dropped into the preview pane, and Mode
// is only consulted for the former, so an unset Tab has to mean "preview".
func TestTab(t *testing.T) {
	var tab docs.Tab

	require.Equal(t, "", tab.Label)
	require.Equal(t, "", tab.Content)
	require.False(t, tab.IsCode)
	require.Equal(t, "", tab.Mode)
}

// tabsHTML runs a ```tabs fence through the module and returns the rendered
// group.
func tabsHTML(t *testing.T, fsys fstest.MapFS, fence string) string {
	t.Helper()

	handler := docs.NewModule(fsys).TabsHandler()
	out, handled := handler(context.Background(), &markdown.Node{
		Type:     markdown.NodeCodeBlock,
		Language: "tabs",
		Raw:      fence,
	})
	require.True(t, handled)
	return out
}

// TestTabBuildsEachType covers the three tab types and what each one puts in
// the panel: render evaluates the file, file shows its source, and example
// does both as a pair.
func TestTabBuildsEachType(t *testing.T) {
	fsys := fstest.MapFS{
		"demo.vuego": &fstest.MapFile{Data: []byte(`<p>{{ name }}</p>`)},
		"demo.yml":   &fstest.MapFile{Data: []byte(`name: World`)},
	}

	t.Run("render evaluates the file", func(t *testing.T) {
		out := tabsHTML(t, fsys, "- label: Preview\n  type: render\n  src: demo.vuego\n")

		require.Contains(t, out, "<p>World</p>")
		require.Contains(t, out, ">Preview</button>")
	})

	t.Run("file shows the source", func(t *testing.T) {
		out := tabsHTML(t, fsys, "- label: Code\n  type: file\n  src: demo.vuego\n")

		require.Contains(t, out, ">Code</button>")
		require.NotContains(t, out, "<p>World</p>", "a file tab shows the template, it does not run it")
	})

	t.Run("example is a preview and a code tab", func(t *testing.T) {
		out := tabsHTML(t, fsys, "- label: Ignored\n  type: example\n  src: demo.vuego\n")

		require.Contains(t, out, ">Preview</button>")
		require.Contains(t, out, ">Code</button>")
	})

	t.Run("an unknown type is reported in place", func(t *testing.T) {
		out := tabsHTML(t, fsys, "- label: Odd\n  type: nonsense\n  src: demo.vuego\n")

		require.Contains(t, out, "unknown type: nonsense")
	})
}

// TestTabEscapesLabels covers a label carrying markup. Labels come from the
// document, and they are written into a button element.
func TestTabEscapesLabels(t *testing.T) {
	fsys := fstest.MapFS{"demo.vuego": &fstest.MapFile{Data: []byte(`<p>hi</p>`)}}

	out := tabsHTML(t, fsys, "- label: \"<script>x</script>\"\n  type: render\n  src: demo.vuego\n")

	require.NotContains(t, out, "<script>x</script>")
	require.Contains(t, out, "&lt;script&gt;")
}
