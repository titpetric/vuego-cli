package codeblock

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"net/http"

	"github.com/titpetric/vuego/markdown"

	"github.com/titpetric/vuego-cli/internal/model"
)

//go:embed assets
var assetFS embed.FS

// Assets returns the embedded static assets (codeblock.js) rooted so that
// "assets/codeblock.js" resolves. Hosts can overlay this into their content
// filesystem to have it served by an /assets/* file server.
func Assets() fs.FS {
	return assetFS
}

// ScriptPath is the URL the runnable widget loads its behaviour from.
const ScriptPath = "/assets/codeblock.js"

// Handler returns an http.HandlerFunc serving POST /api/codeblock/eval.
func (s *Service) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req model.EvalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			_ = json.NewEncoder(w).Encode(model.EvalResponse{Error: "invalid JSON: " + err.Error()})
			return
		}

		_ = json.NewEncoder(w).Encode(s.Eval(r.Context(), req))
	}
}

// CodeBlockHandler returns a markdown handler that rewrites runnable code fences
// (for enabled languages) into a widget with the original code, a Run button and
// an Output area. Unrecognized or disabled languages are left untouched.
func (s *Service) CodeBlockHandler() markdown.Handler {
	return func(_ context.Context, n *markdown.Node) (string, bool) {
		if !s.Handles(n.Language) {
			return "", false
		}
		lang, _ := Canonical(n.Language)
		return s.renderRunnable(lang, n.Language, n.Raw), true
	}
}

// renderRunnable builds the HTML widget for a single runnable snippet.
func (s *Service) renderRunnable(lang, fence, code string) string {
	entry := defaultEntry[lang]
	escaped := html.EscapeString(code)

	return fmt.Sprintf(`<div class="cb-runner" data-cb-language="%s" data-cb-entry="%s">`+
		`<pre class="cb-code"><code class="language-%s">%s</code></pre>`+
		`<div class="cb-toolbar"><button type="button" class="cb-run">Run</button></div>`+
		`<div class="cb-output" hidden><div class="cb-output-label">Output</div>`+
		`<div class="cb-output-body"></div></div></div>`,
		html.EscapeString(lang), html.EscapeString(entry),
		html.EscapeString(fence), escaped,
	)
}
