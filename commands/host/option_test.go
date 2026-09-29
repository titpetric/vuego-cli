package host_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/host"
	"github.com/titpetric/vuego-cli/config"
)

// TestOption_WithResolver covers the seam the tests rely on: the module opens
// every vhost's content through the resolver, so replacing it serves an
// in-memory filesystem where DirFS would have wanted a directory on disk.
func TestOption_WithResolver(t *testing.T) {
	var asked []string

	resolver := func(path string) (fs.FS, error) {
		asked = append(asked, path)
		return fstest.MapFS{
			"README.md": &fstest.MapFile{Data: []byte("# From memory")},
		}, nil
	}

	cfg := &config.Config{
		VHosts: []config.VHost{
			{Domain: "docs.localhost", Path: "/does/not/exist", Mode: "docs"},
		},
	}

	module, err := host.NewModule(cfg, host.WithResolver(resolver))

	require.NoError(t, err)
	require.NotNil(t, module)
	require.Equal(t, []string{"/does/not/exist"}, asked,
		"the module opened the vhost through the replaced resolver, not DirFS")
}

// TestOption_WithResolverReportsFailures covers a resolver that cannot open a
// vhost, which fails module construction rather than producing a site that
// serves nothing.
func TestOption_WithResolverReportsFailures(t *testing.T) {
	resolver := func(string) (fs.FS, error) {
		return nil, fs.ErrNotExist
	}

	cfg := &config.Config{
		VHosts: []config.VHost{
			{Domain: "docs.localhost", Path: "./content", Mode: "docs"},
		},
	}

	_, err := host.NewModule(cfg, host.WithResolver(resolver))
	require.Error(t, err)
}
