package serve_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/serve"
)

func TestCommandCreation(t *testing.T) {
	cmd := serve.New()
	require.NotNil(t, cmd)
	require.Equal(t, "serve", cmd.Name)
}

func TestNewModule(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "index.vuego"), []byte(`<h1>{{ title }}</h1>`), 0644)
	require.NoError(t, err)

	module, err := serve.NewModule(dir)
	require.NoError(t, err)
	require.NotNil(t, module)
	require.Equal(t, "vuego-serve", module.Name())
}

func TestNewModule_InvalidDirectory(t *testing.T) {
	module, err := serve.NewModule("/nonexistent/path")
	require.Error(t, err)
	require.Nil(t, module)
	require.Contains(t, err.Error(), "directory not accessible")
}

func TestServe_InvalidDirectory(t *testing.T) {
	ctx := context.Background()
	err := serve.Serve(ctx, "/nonexistent/path", ":8080")
	require.Error(t, err)
	require.Contains(t, err.Error(), "directory not accessible")
}
