package docs_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	chi "github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/docs"
	"github.com/titpetric/vuego-cli/internal/codeblock"
)

// themedFS is the minimum a docs site needs: a theme declaring an empty menu,
// which the basecoat sidebar partial requires.
func themedFS(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{
		"theme.yml": &fstest.MapFile{Data: []byte("menu: []\n")},
	}
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return fsys
}

// mounted returns a router with a docs module mounted on it, mirroring what
// the platform does at startup.
func mounted(t *testing.T, fsys fstest.MapFS) chi.Router {
	t.Helper()

	module := docs.NewModule(fsys)
	router := chi.NewRouter()
	require.NoError(t, module.Mount(context.Background(), router))
	return router
}

// TestModule_Name covers the name the platform registers the module under.
func TestModule_Name(t *testing.T) {
	require.Equal(t, "vuego-docs", docs.NewModule(fstest.MapFS{}).Name())
}

// TestNewModule covers the overlay the module is built on: the content
// filesystem sits on top, and the embedded templates, basecoat, markdown and
// codeblock assets underneath. A site overrides a default by shipping a file
// of the same name.
func TestNewModule(t *testing.T) {
	module := docs.NewModule(fstest.MapFS{
		"README.md": &fstest.MapFile{Data: []byte("# Mine")},
	})

	require.NotNil(t, module.FS)

	content, err := module.FS.Open("README.md")
	require.NoError(t, err, "the content filesystem is reachable through the overlay")
	require.NoError(t, content.Close())

	asset, err := module.FS.Open("assets/codeblock.js")
	require.NoError(t, err, "the embedded codeblock asset is reachable too")
	require.NoError(t, asset.Close())
}

// TestModule_Mount covers the routes the module registers. Each is asked for
// rather than inspected, since chi does not report what it mounted.
func TestModule_Mount(t *testing.T) {
	router := mounted(t, themedFS(map[string]string{
		"README.md":      "# Index",
		"guide/intro.md": "# Intro",
	}))

	tests := []struct {
		name string
		path string
		want string
	}{
		{"index serves README.md", "/", "Index"},
		{"a doc by its path", "/guide/intro", "Intro"},
		{"a doc by its filename", "/guide/intro.md", "Intro"},
		{"the codeblock script", codeblock.ScriptPath, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			if test.want != "" {
				require.Contains(t, rec.Body.String(), test.want)
			}
		})
	}
}

// TestModule_MountInjectsTheCodeblockScript covers the tag appended to every
// rendered page, without which a "Run" button does nothing.
func TestModule_MountInjectsTheCodeblockScript(t *testing.T) {
	router := mounted(t, themedFS(map[string]string{"README.md": "# Index"}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Contains(t, rec.Body.String(), codeblock.ScriptPath)
}

// TestModule_MountServesADirectoryListing covers a directory with no README,
// which is listed rather than reported as missing.
func TestModule_MountServesADirectoryListing(t *testing.T) {
	router := mounted(t, themedFS(map[string]string{
		"README.md":       "# Index",
		"guide/first.md":  "# First",
		"guide/second.md": "# Second",
	}))

	req := httptest.NewRequest(http.MethodGet, "/guide", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "first.md")
	require.Contains(t, rec.Body.String(), "second.md")
}

// TestModule_MountReportsAMissingDoc covers a path matching nothing, which is
// a 404 rather than an empty page.
func TestModule_MountReportsAMissingDoc(t *testing.T) {
	router := mounted(t, themedFS(map[string]string{"README.md": "# Index"}))

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

// TestModule_LoadMarkdown covers the front matter reaching the caller, which
// is what the layout fills its title from.
func TestModule_LoadMarkdown(t *testing.T) {
	module := docs.NewModule(fstest.MapFS{
		"page.md": &fstest.MapFile{Data: []byte("---\ntitle: Hello\n---\n\n# Body\n")},
	})

	doc, err := module.LoadMarkdown("page.md")

	require.NoError(t, err)
	require.Equal(t, "Hello", doc.FrontMatter()["title"])
}

// TestModule_LoadMarkdownReportsAMissingFile covers the error path, so a
// missing document does not read as an empty one.
func TestModule_LoadMarkdownReportsAMissingFile(t *testing.T) {
	_, err := docs.NewModule(fstest.MapFS{}).LoadMarkdown("absent.md")
	require.Error(t, err)
}

// TestDefaultConfig covers which evaluators the docs server exposes. Shell
// execution runs arbitrary commands on the host, and it is off here on
// purpose: this is the assertion that notices if it is ever turned on.
func TestDefaultConfig(t *testing.T) {
	require.True(t, docs.DefaultConfig.EnablePHP)
	require.True(t, docs.DefaultConfig.EnableVuego)
	require.True(t, docs.DefaultConfig.EnableSQLite)
	require.False(t, docs.DefaultConfig.EnableExec)
}
