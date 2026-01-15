package diff_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/diff"
)

func TestRun_WrongNumberOfArguments(t *testing.T) {
	cmd := diff.New()

	err := cmd.Run(context.TODO(), []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires exactly 2 file arguments")

	err = cmd.Run(context.TODO(), []string{"file1.html"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires exactly 2 file arguments")

	err = cmd.Run(context.TODO(), []string{"a", "b", "c"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires exactly 2 file arguments")
}
