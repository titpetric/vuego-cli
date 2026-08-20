package serve_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	chi "github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/serve"
)

func TestCommandCreation(t *testing.T) {
	cmd := serve.New()
	require.NotNil(t, cmd)
	require.Equal(t, "serve", cmd.Name)
}

func TestNewModule(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "index.vuego"), []byte(`<h1>{{ title }}</h1>`), 0644)
	require.NoError(t, err)

	module, err := serve.NewModule(dir)
	require.NoError(t, err)
	require.NotNil(t, module)
	require.Equal(t, "vuego-serve", module.Name())
}

func TestNewModule_InvalidDirectory(t *testing.T) {
	module, err := serve.NewModule("/nonexistent/path")
	require.Error(t, err)
	require.Nil(t, module)
	require.Contains(t, err.Error(), "directory not accessible")
}

func TestServe_InvalidDirectory(t *testing.T) {
	ctx := context.Background()
	err := serve.Serve(ctx, "/nonexistent/path", ":8080")
	require.Error(t, err)
	require.Contains(t, err.Error(), "directory not accessible")
}

func TestNewModuleFS(t *testing.T) {
	module := serve.NewModuleFS(fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("hello")},
	})
	require.NotNil(t, module)
	require.Equal(t, "vuego-serve", module.Name())
}

// TestModule_MountOnSharedRouter pins that the module attaches its
// middleware to the route rather than to the router. The platform mounts the
// telemetry dashboard before this module, so by the time Mount runs the
// router already carries routes, and chi rejects Use on such a router.
func TestModule_MountOnSharedRouter(t *testing.T) {
	router := chi.NewRouter()
	router.Get("/debug/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})

	module := serve.NewModuleFS(fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("hello")},
	})

	require.NotPanics(t, func() {
		require.NoError(t, module.Mount(context.Background(), router))
	})

	// The pre-existing route still wins over the module's catch all.
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/debug/ping", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "pong", w.Body.String())

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/hello.txt", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "hello", w.Body.String())
}
