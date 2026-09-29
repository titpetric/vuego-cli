package docs_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	chi "github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/docs"
)

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
