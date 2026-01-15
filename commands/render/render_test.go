package render_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/render"
)

func TestRun_WrongNumberOfArguments(t *testing.T) {
	cmd := render.New()

	// No arguments
	err := cmd.Run(context.TODO(), []string{})
	require.Error(t, err)
	require.Equal(t, "render: requires 1 or 2 arguments", err.Error())

	// Too many arguments
	err = cmd.Run(context.TODO(), []string{"a", "b", "c"})
	require.Error(t, err)
	require.Equal(t, "render: requires 1 or 2 arguments", err.Error())
}
