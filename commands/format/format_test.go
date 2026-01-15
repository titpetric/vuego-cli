package format_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/format"
)

func TestRun_NoFiles(t *testing.T) {
	cmd := format.New()
	err := cmd.Run(context.TODO(), []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing file argument")
}

func TestRun_FormatsFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.vuego")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("<div><span>hello</span></div>")
	require.NoError(t, err)
	tmpFile.Close()

	cmd := format.New()
	err = cmd.Run(context.TODO(), []string{tmpFile.Name()})
	require.NoError(t, err)
}
