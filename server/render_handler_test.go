package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/server"
)

func TestRenderHandler_POST(t *testing.T) {
	handler := server.RenderHandler(nil)

	body, _ := json.Marshal(server.RenderRequest{
		Template: `<p>{{ message }}</p>`,
		Data:     `{"message": "Hello"}`,
	})

	req := httptest.NewRequest(http.MethodPost, "/render", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var resp server.RenderResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Empty(t, resp.Error)
	require.Equal(t, "<p>Hello</p>\n", resp.HTML)
}

func TestRenderHandler_MethodNotAllowed(t *testing.T) {
	handler := server.RenderHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/render", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var resp server.RenderResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, "method not allowed", resp.Error)
}

func TestRenderHandler_InvalidJSON(t *testing.T) {
	handler := server.RenderHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/render", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var resp server.RenderResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	require.Contains(t, resp.Error, "invalid JSON:")
}
