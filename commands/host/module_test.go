package host_test

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	chi "github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/host"
	"github.com/titpetric/vuego-cli/config"
)

// docsSite returns the minimum a docs vhost needs: a theme declaring an empty
// menu, which the basecoat sidebar partial requires, and a README.
func docsSite(readme string) fstest.MapFS {
	return fstest.MapFS{
		"theme.yml": &fstest.MapFile{Data: []byte("menu: []\n")},
		"README.md": &fstest.MapFile{Data: []byte(readme)},
	}
}

// resolver returns a host.Resolver serving the named filesystems, so a vhost
// path in a test is a key rather than a directory on disk.
func resolver(sites map[string]fs.FS) host.Resolver {
	return func(path string) (fs.FS, error) {
		contentFS, ok := sites[path]
		if !ok {
			return nil, fs.ErrNotExist
		}
		return contentFS, nil
	}
}

// testRouter mounts a host module for cfg on a fresh chi router, mirroring
// what the platform does at startup.
func testRouter(t *testing.T, cfg *config.Config, sites map[string]fs.FS) chi.Router {
	t.Helper()

	module, err := host.NewModule(cfg, host.WithResolver(resolver(sites)))
	require.NoError(t, err)

	router := chi.NewRouter()
	require.NoError(t, module.Mount(context.Background(), router))
	return router
}

// get issues a GET request for a host and path.
func get(t *testing.T, router chi.Router, host, path string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Host = host

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestModule_RoutesByDomain(t *testing.T) {
	docsFS := docsSite("# Docs site")
	docsFS["guide/install.md"] = &fstest.MapFile{Data: []byte("# Installing the thing")}

	staticFS := fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("static payload")},
	}

	sites := map[string]fs.FS{
		"/docs":   docsFS,
		"/other":  docsSite("# Other site"),
		"/static": staticFS,
	}

	cfg := &config.Config{
		VHosts: []config.VHost{
			{Domain: "docs.example.com", Path: "/docs", Mode: config.ModeDocs},
			{Domain: "other.example.com", Path: "/other", Mode: config.ModeDocs},
			{Domain: "static.example.com", Path: "/static", Mode: config.ModeServe},
		},
	}
	router := testRouter(t, cfg, sites)

	t.Run("first domain", func(t *testing.T) {
		w := get(t, router, "docs.example.com", "/")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Docs site")
		assert.NotContains(t, w.Body.String(), "Other site")
	})

	t.Run("second domain", func(t *testing.T) {
		w := get(t, router, "other.example.com", "/")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Other site")
		assert.NotContains(t, w.Body.String(), "Docs site")
	})

	t.Run("serve mode domain", func(t *testing.T) {
		w := get(t, router, "static.example.com", "/hello.txt")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "static payload", w.Body.String())
	})

	// The docs module resolves nested paths with chi.URLParam(r, "*"), which
	// only reads back correctly against the vhost's own route context.
	t.Run("nested path", func(t *testing.T) {
		w := get(t, router, "docs.example.com", "/guide/install")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Installing the thing")
	})

	t.Run("host with port", func(t *testing.T) {
		w := get(t, router, "docs.example.com:8080", "/")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Docs site")
	})

	t.Run("uppercase host", func(t *testing.T) {
		w := get(t, router, "Docs.Example.COM", "/")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Docs site")
	})

	t.Run("unknown host", func(t *testing.T) {
		w := get(t, router, "nope.example.com", "/")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "no site configured for host")
	})

	t.Run("unknown path within a known host", func(t *testing.T) {
		w := get(t, router, "docs.example.com", "/missing")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NotContains(t, w.Body.String(), "no site configured for host")
	})
}

func TestModule_ForwardedHost(t *testing.T) {
	sites := map[string]fs.FS{"/docs": docsSite("# Docs site")}
	vhosts := []config.VHost{{Domain: "docs.example.com", Path: "/docs", Mode: config.ModeDocs}}

	request := func(router chi.Router, forwarded string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "internal.lb"
		req.Header.Set("X-Forwarded-Host", forwarded)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("ignored by default", func(t *testing.T) {
		router := testRouter(t, &config.Config{VHosts: vhosts}, sites)
		w := request(router, "docs.example.com")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("honoured when trusted", func(t *testing.T) {
		router := testRouter(t, &config.Config{TrustForwardedHost: true, VHosts: vhosts}, sites)
		w := request(router, "docs.example.com")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Docs site")
	})

	t.Run("proxy chain uses the first entry", func(t *testing.T) {
		router := testRouter(t, &config.Config{TrustForwardedHost: true, VHosts: vhosts}, sites)
		w := request(router, "docs.example.com, inner.proxy")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Docs site")
	})
}

func TestModule_TourMode(t *testing.T) {
	tourFS := fstest.MapFS{
		"README.md":    &fstest.MapFile{Data: []byte("# Welcome to the tour")},
		"01-basics.md": &fstest.MapFile{Data: []byte("# Basics\n\n## First lesson\n\n@file: hello.vuego\n")},
		"basics/hello.vuego": &fstest.MapFile{
			Data: []byte("<h1>{{ title }}</h1>"),
		},
	}

	cfg := &config.Config{
		VHosts: []config.VHost{{Domain: "tour.example.com", Path: "/tour", Mode: config.ModeTour}},
	}
	router := testRouter(t, cfg, map[string]fs.FS{"/tour": tourFS})

	w := get(t, router, "tour.example.com", "/")
	assert.Equal(t, http.StatusOK, w.Code)

	// The tour registers a route with path parameters, which only resolve
	// against a route context belonging to the vhost's own router.
	w = get(t, router, "tour.example.com", "/lesson/basics/1")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNewModule_InvalidMode(t *testing.T) {
	cfg := &config.Config{
		VHosts: []config.VHost{{Domain: "a.example.com", Path: "/site", Mode: "blog"}},
	}

	_, err := host.NewModule(cfg, host.WithResolver(resolver(map[string]fs.FS{"/site": fstest.MapFS{}})))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown mode")
	assert.Contains(t, err.Error(), "a.example.com")
}

func TestNewModule_UnresolvablePath(t *testing.T) {
	cfg := &config.Config{
		VHosts: []config.VHost{{Domain: "a.example.com", Path: "/absent", Mode: config.ModeDocs}},
	}

	_, err := host.NewModule(cfg, host.WithResolver(resolver(nil)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "a.example.com")
}

func TestDirFS(t *testing.T) {
	t.Run("missing directory", func(t *testing.T) {
		_, err := host.DirFS("/nonexistent/path")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "directory not accessible")
	})

	t.Run("existing directory", func(t *testing.T) {
		contentFS, err := host.DirFS(".")
		require.NoError(t, err)
		require.NotNil(t, contentFS)

		// This package's own source is readable through it.
		_, err = fs.Stat(contentFS, "module.go")
		require.NoError(t, err)
	})
}

func TestModule_Name(t *testing.T) {
	module, err := host.NewModule(&config.Config{})
	require.NoError(t, err)
	assert.Equal(t, "vuego-host", module.Name())
}

func TestCommandCreation(t *testing.T) {
	cmd := host.New()
	require.NotNil(t, cmd)
	assert.Equal(t, "host", cmd.Name)
}
