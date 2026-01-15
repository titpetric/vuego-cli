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
