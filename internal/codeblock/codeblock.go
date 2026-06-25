// Package codeblock wires the individual evaluators (php, vuego, exec, sqlite)
// into a single service used by the tour and docs servers. It exposes:
//
//   - an HTTP handler for POST /api/codeblock/eval
//   - a markdown handler that rewrites runnable code fences into widgets with a
//     "Run" button and an "Output" area
//
// Which evaluators are available is decided by Config, so each host (tour, docs)
// opts in to exactly the languages it wants to support.
package codeblock

import (
	"context"
	"fmt"
	"io/fs"
	"testing/fstest"

	"github.com/titpetric/vuego"

	"github.com/titpetric/vuego-cli/internal/exec"
	"github.com/titpetric/vuego-cli/internal/model"
	"github.com/titpetric/vuego-cli/internal/php"
	"github.com/titpetric/vuego-cli/internal/sqlite"
	vuegoeval "github.com/titpetric/vuego-cli/internal/vuego"
)

// EvalPath is the HTTP path that the frontend posts evaluation requests to.
const EvalPath = "/api/codeblock/eval"

// Canonical language identifiers.
const (
	LangVuego = "vuego"
	LangPHP   = "php"
	LangExec  = "exec"
	LangSQL   = "sql"
)

// Config selects which evaluators are enabled. Hosts define their own default
// (for example tour.DefaultConfig, docs.DefaultConfig) so evaluation is opt-in.
type Config struct {
	EnablePHP    bool
	EnableVuego  bool
	EnableExec   bool
	EnableSQLite bool
}

// aliases maps fence info strings to a canonical language identifier.
var aliases = map[string]string{
	"vuego":                   LangVuego,
	"html+vuego":              LangVuego,
	"php":                     LangPHP,
	"application/x-httpd-php": LangPHP,
	"exec":                    LangExec,
	"bash":                    LangExec,
	"sh":                      LangExec,
	"shell":                   LangExec,
	"sql":                     LangSQL,
	"sqlite":                  LangSQL,
	"sqlite3":                 LangSQL,
}

// defaultEntry maps a canonical language to its default entry filename.
var defaultEntry = map[string]string{
	LangVuego: "index.vuego",
	LangPHP:   "index.php",
	LangExec:  "index.sh",
	LangSQL:   "index.sql",
}

// Service evaluates code snippets for the enabled languages.
type Service struct {
	evaluators map[string]model.Evaluator
}

// New builds a Service from cfg. baseFS supplies shared dependencies (components,
// layouts) for the vuego evaluator and may be nil; opts are vuego load options
// applied to vuego renders.
func New(cfg Config, baseFS fs.FS, opts ...vuego.LoadOption) *Service {
	evaluators := make(map[string]model.Evaluator)
	if cfg.EnableVuego {
		evaluators[LangVuego] = vuegoeval.New(baseFS, opts...)
	}
	if cfg.EnablePHP {
		evaluators[LangPHP] = php.New()
	}
	if cfg.EnableExec {
		evaluators[LangExec] = exec.New()
	}
	if cfg.EnableSQLite {
		evaluators[LangSQL] = sqlite.New()
	}
	return &Service{evaluators: evaluators}
}

// Canonical resolves a fence info string to a canonical language identifier.
// It returns ("", false) when the language is unknown.
func Canonical(language string) (string, bool) {
	lang, ok := aliases[language]
	return lang, ok
}

// Handles reports whether the given fence language is recognized and enabled.
func (s *Service) Handles(language string) bool {
	lang, ok := Canonical(language)
	if !ok {
		return false
	}
	_, enabled := s.evaluators[lang]
	return enabled
}

// Eval runs a single evaluation request and returns a response suitable for
// JSON encoding. Errors are reported in the response rather than returned.
func (s *Service) Eval(ctx context.Context, req model.EvalRequest) model.EvalResponse {
	lang, ok := Canonical(req.Language)
	if !ok {
		return model.EvalResponse{Error: fmt.Sprintf("unknown language: %q", req.Language)}
	}
	ev, enabled := s.evaluators[lang]
	if !enabled {
		return model.EvalResponse{Error: fmt.Sprintf("evaluation not enabled for: %q", req.Language)}
	}

	entry := req.Entry
	if entry == "" {
		entry = defaultEntry[lang]
	}

	res, err := ev.Eval(ctx, model.Request{FS: buildFS(req.Files), Entry: entry})
	if err != nil {
		return model.EvalResponse{Error: err.Error()}
	}
	return model.EvalResponse{ContentType: res.ContentType, Content: res.Content}
}

// buildFS turns a name->content map into an fs.FS for an evaluation.
func buildFS(files map[string]string) fs.FS {
	fsys := fstest.MapFS{}
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return fsys
}
