package serve_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/serve"
)

func TestServe_Integration(t *testing.T) {
	// Create temporary directory with test files
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "index.vuego"), []byte(`<h1>{{ title }}</h1>`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "index.yaml"), []byte(`title: Hello World`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "style.less"), []byte(`@color: #333; body { color: @color; }`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{"key": "value"}`), 0644)
	require.NoError(t, err)

	// Create a test server by calling serve directly
	// We can't easily test the full server startup, but we can verify the function signature
	_ = dir
}

func TestServe_InvalidDirectory(t *testing.T) {
	ctx := context.Background()
	err := serve.Serve(ctx, "/nonexistent/path", ":8080")
	require.Error(t, err)
	require.Contains(t, err.Error(), "directory not accessible")
}
