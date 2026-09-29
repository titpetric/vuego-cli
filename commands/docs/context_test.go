package docs_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/docs"
)

func TestDocDir(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, ".", docs.DocDir(ctx))

	ctx = docs.WithDocDir(ctx, "components")
	require.Equal(t, "components", docs.DocDir(ctx))
}
