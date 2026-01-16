package tour_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chi "github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/tour"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	module := tour.NewModule()
	router := chi.NewRouter()
	err := module.Mount(context.Background(), router)
	require.NoError(t, err)
	return router
}

func TestModule_ServesIndexPage(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "Vuego tour")
}

func TestModule_LessonEndpoint_ReturnsJSON(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lesson/interpolation/0", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var lesson tour.Lesson
	err := json.NewDecoder(rec.Body).Decode(&lesson)
	require.NoError(t, err)
	require.NotEmpty(t, lesson.Title)
}

func TestModule_LessonEndpoint_ReturnsHTML(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lesson/interpolation/0", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "Vuego tour")
}

func TestModule_RenderEndpoint(t *testing.T) {
	handler := newTestHandler(t)

	body := `{"template": "<div>Hello {{ name }}</div>", "data": "{\"name\": \"World\"}"}`
	req := httptest.NewRequest(http.MethodPost, "/render", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Contains(t, resp["html"], "Hello World")
}

func TestModule_RenderEndpoint_MethodNotAllowed(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/render", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestModule_RenderEndpoint_InvalidJSON(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/render", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Contains(t, resp["error"], "invalid JSON")
}

func TestModule_DonePage(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/done", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "Vuego tour")
}

func TestModule_StaticJS(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/static/tour.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/javascript", rec.Header().Get("Content-Type"))
}

func TestModule_StaticCSS(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/static/tour.css", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/css", rec.Header().Get("Content-Type"))
}

func TestModule_StylingLesson1_IncludesFormsLess(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lesson/styling/1", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var lesson tour.Lesson
	err := json.NewDecoder(rec.Body).Decode(&lesson)
	require.NoError(t, err)
	require.Equal(t, "External Stylesheets", lesson.Title)
	require.Contains(t, lesson.Files, "forms.less")
}

func TestModule_RenderEndpoint_InjectsStyleTags(t *testing.T) {
	handler := newTestHandler(t)

	// Render a template with form styles included (valid LESS syntax)
	req := httptest.NewRequest(http.MethodPost, "/render", nil)
	req.Header.Set("Content-Type", "application/json")

	reqBody := map[string]any{
		"template": `<html><head></head><body><style type="text/css+less">{{ file("forms.less") }}</style></body></html>`,
		"data":     "{}",
		"files": map[string]string{
			"forms.less": `.form-group {
  margin: 1rem;

  a {
    font-weight: bold;
  }
}`,
		},
	}
	jsonBody, _ := json.Marshal(reqBody)
	req.Body = io.NopCloser(bytes.NewReader(jsonBody))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Empty(t, resp["error"])
	require.Contains(t, resp["html"], "<style")
	require.Contains(t, resp["html"], ".form-group a {")
}
