// Package php evaluates PHP snippets using the phpscript runner. It implements
// the model.Evaluator interface and reads its entry point and includes from the
// fs.FS supplied on the request.
package php

import (
	"bytes"
	"context"
	"io/fs"
	"sort"
	"strings"

	"github.com/titpetric/phpscript/runner"
	"github.com/titpetric/phpscript/stdlib"

	"github.com/titpetric/vuego-cli/internal/model"
)

// DefaultOptions configures the PHP VM used to evaluate snippets.
var DefaultOptions = runner.Options{
	SAPI:          "vuego-cli",
	WorkDir:       ".",
	WritablePaths: []string{"."},
}

// Evaluator runs PHP snippets.
type Evaluator struct {
	Options runner.Options
}

// New returns a PHP evaluator using DefaultOptions.
func New() *Evaluator {
	return &Evaluator{Options: DefaultOptions}
}

// Eval loads req.Entry from req.FS and executes it, returning the captured
// output as HTML. When the requested entry is missing it falls back to the
// first .php file found in the filesystem.
func (e *Evaluator) Eval(ctx context.Context, req model.Request) (model.Result, error) {
	entry := req.Entry
	if entry == "" {
		entry = "index.php"
	}
	if _, err := fs.Stat(req.FS, entry); err != nil {
		if alt := firstPHP(req.FS); alt != "" {
			entry = alt
		}
	}

	var out bytes.Buffer
	opts := e.Options
	opts.RootFS = req.FS

	rt := runner.New(&out, opts)
	rt.SetContext(ctx)
	stdlib.Register(rt)

	prog, err := rt.LoadFile(entry)
	if err != nil {
		return model.Result{}, err
	}
	if err := rt.Run(prog); err != nil {
		if _, ok := runner.IsExit(err); !ok {
			return model.Result{}, err
		}
	}
	return model.Result{ContentType: model.ContentTypeHTML, Content: out.String()}, nil
}

// firstPHP returns the lexically first .php file in the root of fsys, or "".
func firstPHP(fsys fs.FS) string {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return ""
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".php") {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return names[0]
}
