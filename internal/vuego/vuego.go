// Package vuego evaluates .vuego templates. It reads the entry template and its
// adjacent data file from the request filesystem and renders HTML by reusing
// the existing server.Render implementation.
package vuego

import (
	"context"
	"io/fs"
	"strings"

	"github.com/titpetric/vuego"

	"github.com/titpetric/vuego-cli/internal/model"
	"github.com/titpetric/vuego-cli/server"
)

// Evaluator renders .vuego templates to HTML.
type Evaluator struct {
	baseFS fs.FS
	opts   []vuego.LoadOption
}

// New returns a vuego evaluator. baseFS supplies static dependencies such as
// shared components and layouts that the rendered templates may include; it may
// be nil. The provided load options are applied to every render (for example
// vuego.WithLessProcessor()).
func New(baseFS fs.FS, opts ...vuego.LoadOption) *Evaluator {
	return &Evaluator{baseFS: baseFS, opts: opts}
}

// Eval renders req.Entry from req.FS, which holds only the snippet's own files.
// The matching data file (.yaml/.yml/.json) next to the entry is loaded
// automatically. Shared components come from the evaluator's base filesystem.
func (e *Evaluator) Eval(ctx context.Context, req model.Request) (model.Result, error) {
	entry := req.Entry
	if entry == "" {
		entry = "index.vuego"
	}

	tpl, err := fs.ReadFile(req.FS, entry)
	if err != nil {
		return model.Result{}, err
	}

	files, err := readFiles(req.FS)
	if err != nil {
		return model.Result{}, err
	}

	html, err := server.Render(ctx, e.baseFS, server.RenderRequest{
		Template: string(tpl),
		Data:     dataFor(files, entry),
		Files:    files,
	}, e.opts...)
	if err != nil {
		return model.Result{}, err
	}
	return model.Result{ContentType: model.ContentTypeHTML, Content: html}, nil
}

// dataFor returns the content of the data file adjacent to entry, if any.
func dataFor(files map[string]string, entry string) string {
	base := strings.TrimSuffix(entry, ".vuego")
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		if content, ok := files[base+ext]; ok {
			return content
		}
	}
	return ""
}

// readFiles collects every regular file in fsys into a name->content map.
func readFiles(fsys fs.FS) (map[string]string, error) {
	files := make(map[string]string)
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		files[p] = string(data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
