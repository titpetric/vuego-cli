package codeblock_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/titpetric/vuego/markdown"

	"github.com/titpetric/vuego-cli/internal/codeblock"
	"github.com/titpetric/vuego-cli/internal/model"
)

// TestAssets covers the embedded script being reachable under the path the
// widget loads it from. A host overlays this filesystem and serves it at
// /assets/*, so the name inside has to match ScriptPath.
func TestAssets(t *testing.T) {
	data, err := fs.ReadFile(codeblock.Assets(), "assets/codeblock.js")

	require.NoError(t, err)
	require.NotEmpty(t, data)
	require.Equal(t, "/assets/codeblock.js", codeblock.ScriptPath)
}

// TestService_Handler covers the evaluation endpoint: a request names a
// language and its files, and the response carries the rendered output.
func TestService_Handler(t *testing.T) {
	svc := codeblock.New(codeblock.Config{EnablePHP: true}, nil)

	body, err := json.Marshal(model.EvalRequest{
		Language: "php",
		Files:    map[string]string{"index.php": `<?php echo "hi";`},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, codeblock.EvalPath, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	svc.Handler().ServeHTTP(rec, req)

	var resp model.EvalResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Empty(t, resp.Error)
	require.Equal(t, "hi", resp.Content)
}

// TestService_HandlerInvalidJSON covers a malformed body, which is reported in
// the response rather than as a status code: the frontend reads the same shape
// either way.
func TestService_HandlerInvalidJSON(t *testing.T) {
	svc := codeblock.New(codeblock.Config{EnablePHP: true}, nil)

	req := httptest.NewRequest(http.MethodPost, codeblock.EvalPath, bytes.NewReader([]byte(`{bad`)))
	rec := httptest.NewRecorder()
	svc.Handler().ServeHTTP(rec, req)

	var resp model.EvalResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Contains(t, resp.Error, "invalid JSON:")
}

// TestService_HandlerDisabledLanguage covers a language the host did not opt
// into, which is refused in the response rather than evaluated.
func TestService_HandlerDisabledLanguage(t *testing.T) {
	svc := codeblock.New(codeblock.Config{EnablePHP: true}, nil)

	body, err := json.Marshal(model.EvalRequest{
		Language: "exec",
		Files:    map[string]string{"index.sh": "echo no"},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, codeblock.EvalPath, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	svc.Handler().ServeHTTP(rec, req)

	var resp model.EvalResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Contains(t, resp.Error, "evaluation not enabled")
	require.Empty(t, resp.Content)
}

// TestService_CodeBlockHandlerLeavesDisabledFences covers the markdown side of
// the same opt-in: a fence whose language is not enabled is not handled, so the
// default code block rendering takes it.
func TestService_CodeBlockHandlerLeavesDisabledFences(t *testing.T) {
	svc := codeblock.New(codeblock.Config{EnablePHP: true}, nil)
	handler := svc.CodeBlockHandler()

	_, handled := handler(context.Background(), &markdown.Node{
		Type:     markdown.NodeCodeBlock,
		Language: "exec",
		Raw:      "echo no",
	})

	require.False(t, handled)
}

// TestService_CodeBlockHandlerEscapesCode covers the snippet being written into
// the page as text: a fence containing markup must not be able to break out of
// the <code> element holding it.
func TestService_CodeBlockHandlerEscapesCode(t *testing.T) {
	svc := codeblock.New(codeblock.Config{EnableVuego: true}, nil)
	handler := svc.CodeBlockHandler()

	out, handled := handler(context.Background(), &markdown.Node{
		Type:     markdown.NodeCodeBlock,
		Language: "vuego",
		Raw:      `<div class="x">{{ name }}</div>`,
	})

	require.True(t, handled)
	require.Contains(t, out, `data-cb-language="vuego"`)
	require.Contains(t, out, `data-cb-entry="index.vuego"`)
	require.Contains(t, out, "&lt;div class=&#34;x&#34;&gt;")
	require.NotContains(t, out, `<div class="x">`)
}
