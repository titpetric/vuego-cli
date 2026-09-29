package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/titpetric/vuego"

	"github.com/titpetric/vuego-cli/server"
)

// TestMiddlewareOption covers the option type being the currency Middleware
// and MiddlewareDir both take, so a caller can build a list once and hand it
// to either.
//
// The config an option writes to is unexported on purpose: WithLoadOption is
// the only way to make one, and that is what keeps the set of knobs closed.
func TestMiddlewareOption(t *testing.T) {
	opts := []server.MiddlewareOption{
		server.WithLoadOption(vuego.WithLessProcessor()),
	}

	require.NotNil(t, server.Middleware(fstest.MapFS{}, opts...))
	require.NotNil(t, server.MiddlewareDir(t.TempDir(), opts...))
	require.NotNil(t, server.Middleware(fstest.MapFS{}), "no options is a valid call")
}

// TestWithLoadOption covers the option reaching the renderer. A .less block is
// only compiled when the LessProcessor was passed through, so the rendered
// output is what says whether the option arrived.
func TestWithLoadOption(t *testing.T) {
	fsys := fstest.MapFS{
		"page.vuego": &fstest.MapFile{Data: []byte(
			`<html><head><style type="text/css+less">@c: red; p { color: @c; }</style></head><body><p>hi</p></body></html>`)},
	}

	tests := []struct {
		name    string
		opts    []server.MiddlewareOption
		wantCSS bool
	}{
		{"without the processor", nil, false},
		{"with the processor", []server.MiddlewareOption{
			server.WithLoadOption(vuego.WithLessProcessor()),
		}, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			server.Middleware(fsys, test.opts...).ServeHTTP(
				rec, httptest.NewRequest(http.MethodGet, "/page.vuego", nil))

			require.Equal(t, http.StatusOK, rec.Code)
			if test.wantCSS {
				require.Contains(t, rec.Body.String(), "color: red")
				require.NotContains(t, rec.Body.String(), "@c")
			} else {
				require.Contains(t, rec.Body.String(), "@c")
			}
		})
	}
}

// TestWithLoadOptionAccumulates covers several options, passed one at a time
// and several at once, all reaching the same configuration.
func TestWithLoadOptionAccumulates(t *testing.T) {
	fsys := fstest.MapFS{
		"page.vuego": &fstest.MapFile{Data: []byte(`<p>{{ name }}</p>`)},
		"page.yml":   &fstest.MapFile{Data: []byte(`name: World`)},
	}

	handler := server.Middleware(fsys,
		server.WithLoadOption(vuego.WithLessProcessor()),
		server.WithLoadOption(vuego.WithComponents(), vuego.WithLessProcessor()),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/page.vuego", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "<p>World</p>")
}
