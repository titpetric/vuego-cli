// Package model holds the shared type definitions used by the code block
// evaluation pipeline: the Evaluator interface together with its request and
// result types, and the HTTP DTOs exchanged with the frontend.
package model

import (
	"context"
	"io/fs"
)

// Common content types returned by evaluators.
const (
	ContentTypeHTML = "text/html"
	ContentTypeText = "text/plain"
	ContentTypeJSON = "application/json"
)

// Request is the input to an Evaluator. The caller is responsible for
// constructing an fs.FS containing every file required for the run and for
// nominating the entry point to evaluate (for example "index.php").
type Request struct {
	// FS holds all files available to the evaluation.
	FS fs.FS
	// Entry is the file within FS that should be evaluated.
	Entry string
}

// Result is the output of an Evaluator. ContentType lets the frontend decide
// how to render Content (HTML in an iframe, plain text in a <pre>, JSON as a
// table, and so on).
type Result struct {
	ContentType string
	Content     string
}

// Evaluator evaluates a code snippet built on top of an fs.FS. Implementations
// live in the internal/php, internal/vuego, internal/exec and internal/sqlite
// packages.
type Evaluator interface {
	Eval(ctx context.Context, req Request) (Result, error)
}
