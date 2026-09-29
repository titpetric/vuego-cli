package docs

import (
	"context"
)

type docDirKey struct{}

// WithDocDir returns a context with the document directory.
func WithDocDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, docDirKey{}, dir)
}

// DocDir returns the document directory from context.
func DocDir(ctx context.Context) string {
	if v, ok := ctx.Value(docDirKey{}).(string); ok {
		return v
	}
	return "."
}
