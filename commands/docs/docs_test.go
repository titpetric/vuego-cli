package docs_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/docs"
)

// TestNew covers the command the binary registers under "docs": its name, its
// title and the flag Bind defines.
func TestNew(t *testing.T) {
	cmd := docs.New()

	require.Equal(t, "docs", cmd.Name)
	require.Equal(t, docs.Name, cmd.Title)
	require.NotNil(t, cmd.Run)
	require.NotNil(t, cmd.Bind)
}

// TestServe covers the signature the command's Run closure calls. Serve starts
// a server and blocks, so what is checked here is that the entry point exists
// with the shape the command depends on.
func TestServe(t *testing.T) {
	require.NotNil(t, docs.Serve)
}
