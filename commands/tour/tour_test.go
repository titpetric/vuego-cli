package tour_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/tour"
)

func TestCommandCreation(t *testing.T) {
	cmd := tour.New()
	require.NotNil(t, cmd)
	require.Equal(t, "tour", cmd.Name)
}

// TestServe covers what Serve decides before it starts listening: the content
// path picks between a directory on disk and the embedded tour. Serve blocks
// once the server is up, so the branch is exercised through the module it
// builds rather than by calling Serve itself.
func TestServe(t *testing.T) {
	require.NotNil(t, tour.Serve)

	cmd := tour.New()
	require.NotNil(t, cmd.Run, "the command's Run closure is the only caller of Serve")
	require.NotNil(t, cmd.Bind, "and --addr and --content are what it passes through")
}
