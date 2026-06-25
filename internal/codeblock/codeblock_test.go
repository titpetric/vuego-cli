package codeblock_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/titpetric/vuego"
	"github.com/titpetric/vuego/markdown"

	"github.com/titpetric/vuego-cli/internal/codeblock"
	"github.com/titpetric/vuego-cli/internal/model"
)

func newService() *codeblock.Service {
	cfg := codeblock.Config{
		EnablePHP:    true,
		EnableVuego:  true,
		EnableExec:   true,
		EnableSQLite: true,
	}
	return codeblock.New(cfg, nil, vuego.WithLessProcessor())
}

func TestEval_PHP(t *testing.T) {
	s := newService()
	resp := s.Eval(context.Background(), model.EvalRequest{
		Language: "php",
		Files:    map[string]string{"index.php": "<?php echo 'Hello PHP';"},
	})
	require.Empty(t, resp.Error)
	require.Equal(t, model.ContentTypeHTML, resp.ContentType)
	require.Equal(t, "Hello PHP", resp.Content)
}

func TestEval_Vuego(t *testing.T) {
	s := newService()
	resp := s.Eval(context.Background(), model.EvalRequest{
		Language: "vuego",
		Files: map[string]string{
			"index.vuego": "<div>Hello {{ name }}</div>",
			"index.yaml":  "name: World",
		},
	})
	require.Empty(t, resp.Error)
	require.Equal(t, model.ContentTypeHTML, resp.ContentType)
	require.Contains(t, resp.Content, "Hello World")
}

func TestEval_SQL(t *testing.T) {
	s := newService()
	resp := s.Eval(context.Background(), model.EvalRequest{
		Language: "sql",
		Files: map[string]string{
			"index.up.sql": "CREATE TABLE user_account (id INTEGER PRIMARY KEY, name TEXT); INSERT INTO user_account (name) VALUES ('Ada');",
			"index.sql":    "SELECT id, name FROM user_account;",
		},
	})
	require.Empty(t, resp.Error)
	require.Equal(t, model.ContentTypeHTML, resp.ContentType)
	require.Contains(t, resp.Content, "<th>name</th>")
	require.Contains(t, resp.Content, "<td>Ada</td>")
}

func TestEval_Exec_SingleCommand(t *testing.T) {
	s := newService()
	resp := s.Eval(context.Background(), model.EvalRequest{
		Language: "exec",
		Files:    map[string]string{"index.sh": "echo hello"},
	})
	require.Empty(t, resp.Error)
	require.Equal(t, model.ContentTypeText, resp.ContentType)
	require.Equal(t, "$ echo hello\nhello\n", resp.Content)
}

func TestEval_Exec_MultipleCommands(t *testing.T) {
	s := newService()
	resp := s.Eval(context.Background(), model.EvalRequest{
		Language: "exec",
		Files:    map[string]string{"index.sh": "echo hello\necho world"},
	})
	require.Empty(t, resp.Error)
	require.Equal(t, "$ echo hello\nhello\n$ echo world\nworld\n", resp.Content)
}

func TestEval_DisabledLanguage(t *testing.T) {
	s := codeblock.New(codeblock.Config{EnableVuego: true}, nil)
	resp := s.Eval(context.Background(), model.EvalRequest{
		Language: "exec",
		Files:    map[string]string{"index.sh": "echo hi"},
	})
	require.Contains(t, resp.Error, "not enabled")
}

func TestEval_UnknownLanguage(t *testing.T) {
	s := newService()
	resp := s.Eval(context.Background(), model.EvalRequest{Language: "ruby"})
	require.Contains(t, resp.Error, "unknown language")
}

func TestHandles(t *testing.T) {
	s := newService()
	require.True(t, s.Handles("php"))
	require.True(t, s.Handles("html+vuego"))
	require.True(t, s.Handles("sqlite3"))
	require.True(t, s.Handles("bash"))
	require.False(t, s.Handles("tabs"))
	require.False(t, s.Handles("go"))

	disabled := codeblock.New(codeblock.Config{EnableVuego: true}, nil)
	require.False(t, disabled.Handles("php"))
}

func TestCodeBlockHandler_RewritesRunnableFence(t *testing.T) {
	s := newService()
	h := s.CodeBlockHandler()

	out, handled := h(context.Background(), &markdown.Node{
		Type:     markdown.NodeCodeBlock,
		Raw:      "<?php echo 1;",
		Language: "php",
	})
	require.True(t, handled)
	require.Contains(t, out, `class="cb-runner"`)
	require.Contains(t, out, `data-cb-language="php"`)
	require.Contains(t, out, `data-cb-entry="index.php"`)
	require.Contains(t, out, `class="cb-run"`)
	require.Contains(t, out, "cb-output")
	// code is HTML-escaped inside the widget
	require.Contains(t, out, "&lt;?php echo 1;")
}

func TestCodeBlockHandler_IgnoresUnknownFence(t *testing.T) {
	s := newService()
	h := s.CodeBlockHandler()
	_, handled := h(context.Background(), &markdown.Node{
		Type:     markdown.NodeCodeBlock,
		Raw:      "package main\n\n func main() {}",
		Language: "go",
	})
	require.False(t, handled)
}

func TestHTTPHandler_RoundTrip(t *testing.T) {
	s := newService()
	body, _ := json.Marshal(model.EvalRequest{
		Language: "exec",
		Files:    map[string]string{"index.sh": "echo hi"},
	})
	req := httptest.NewRequest(http.MethodPost, codeblock.EvalPath, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp model.EvalResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Empty(t, resp.Error)
	require.Equal(t, "$ echo hi\nhi\n", resp.Content)
}

func TestHTTPHandler_InvalidJSON(t *testing.T) {
	s := newService()
	req := httptest.NewRequest(http.MethodPost, codeblock.EvalPath, strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	var resp model.EvalResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Contains(t, resp.Error, "invalid JSON")
}
