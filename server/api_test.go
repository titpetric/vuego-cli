package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/server"
)

func TestRender_JSONData(t *testing.T) {
	req := server.RenderRequest{
		Template: `<div>{{ name }}</div>`,
		Data:     `{"name": "World"}`,
	}

	html, err := server.Render(context.Background(), nil, req)
	require.NoError(t, err)
	require.Equal(t, "<div>World</div>\n", html)
}

func TestRender_YAMLData(t *testing.T) {
	req := server.RenderRequest{
		Template: `<div>{{ name }}</div>`,
		Data:     "name: World",
	}

	html, err := server.Render(context.Background(), nil, req)
	require.NoError(t, err)
	require.Equal(t, "<div>World</div>\n", html)
}

func TestRender_AdditionalFiles(t *testing.T) {
	req := server.RenderRequest{
		Template: `<template include="partial.vuego"></template>`,
		Files: map[string]string{
			"partial.vuego": `<span>Included</span>`,
		},
	}

	html, err := server.Render(context.Background(), nil, req)
	require.NoError(t, err)
	require.Equal(t, "<span>Included</span>\n", html)
}

func TestRender_InvalidData(t *testing.T) {
	req := server.RenderRequest{
		Template: `<div>{{ name }}</div>`,
		Data:     `{invalid json`,
	}

	_, err := server.Render(context.Background(), nil, req)
	require.Error(t, err)
}

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

func TestRender_NamedSlotDefault(t *testing.T) {
	req := server.RenderRequest{
		Template: `<template include="sidebar.vuego"></template>`,
		Files: map[string]string{
			"sidebar.vuego": `<nav><slot name="header"><span>Basecoat</span></slot></nav>`,
		},
	}

	html, err := server.Render(context.Background(), nil, req)
	require.NoError(t, err)
	require.Contains(t, html, "Basecoat")
}

func TestRender_NamedSlotOverride(t *testing.T) {
	req := server.RenderRequest{
		Template: `<template include="sidebar.vuego"><template v-slot:header><span>Custom Brand</span></template></template>`,
		Files: map[string]string{
			"sidebar.vuego": `<nav><slot name="header"><span>Basecoat</span></slot></nav>`,
		},
	}

	html, err := server.Render(context.Background(), nil, req)
	require.NoError(t, err)
	require.Contains(t, html, "Custom Brand")
	require.NotContains(t, html, "Basecoat")
}

func TestRender_NamedSlotWithData(t *testing.T) {
	req := server.RenderRequest{
		Template: `<template include="sidebar.vuego"><template v-slot:header><span>{{ header.title }}</span></template></template>`,
		Data:     `{"header": {"title": "My App"}}`,
		Files: map[string]string{
			"sidebar.vuego": `<nav><slot name="header"><span>Basecoat</span></slot></nav>`,
		},
	}

	html, err := server.Render(context.Background(), nil, req)
	require.NoError(t, err)
	require.Contains(t, html, "My App")
	require.NotContains(t, html, "Basecoat")
}

func TestRender_LayoutWithRelativePath(t *testing.T) {
	req := server.RenderRequest{
		Template: `---
layout: base.vuego
---
<h1>{{ title }}</h1>
<p>Page content</p>`,
		Data: `title: Hello`,
		Files: map[string]string{
			"base.vuego": `<!DOCTYPE html>
<html>
<body>
<main v-html="content"></main>
</body>
</html>`,
		},
	}

	html, err := server.Render(context.Background(), nil, req)
	require.NoError(t, err)
	require.Contains(t, html, "<h1>Hello</h1>")
	require.Contains(t, html, "<p>Page content</p>")
	require.Contains(t, html, "<main>")
	require.Contains(t, html, "</html>")
}
