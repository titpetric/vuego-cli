package host_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/host"
	"github.com/titpetric/vuego-cli/config"
)

// TestNew covers the command the binary registers: its name, its title, and
// the two flags Bind defines. A renamed flag is a broken invocation nobody
// notices until they type it.
func TestNew(t *testing.T) {
	cmd := host.New()

	require.Equal(t, "host", cmd.Name)
	require.Equal(t, host.Name, cmd.Title)
	require.NotNil(t, cmd.Run)
	require.NotNil(t, cmd.Usage)
	require.Contains(t, cmd.Usage(), "vuego-cli.yml")
}

// TestDefaultAddr covers the address used when neither the command line nor
// the configuration file names one.
func TestDefaultAddr(t *testing.T) {
	require.Equal(t, ":8080", host.DefaultAddr)
}

// TestServe_ReportsAnUnreadableConfig covers the named configuration file that
// is not there. Serve blocks once it starts a server, so the cases worth
// testing are the ones that return before that.
func TestServe_ReportsAnUnreadableConfig(t *testing.T) {
	err := host.Serve(context.Background(), "", filepath.Join(t.TempDir(), "absent.yml"))
	require.Error(t, err)
}

// TestServe_ReportsAnInvalidConfig covers a configuration file that parses but
// does not describe a servable set of vhosts.
func TestServe_ReportsAnInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vuego-cli.yml")
	require.NoError(t, os.WriteFile(path, []byte("vhost:\n  - domain: \"\"\n    path: \"\"\n"), 0o600))

	err := host.Serve(context.Background(), "", path)
	require.Error(t, err)
}

// TestConfigFindIsTheNoFlagPath covers what Serve does with an empty
// configPath: it searches the current directory, and a miss is ErrNotFound
// rather than a failure, which is what sends it down the serve-this-directory
// fallback.
func TestConfigFindIsTheNoFlagPath(t *testing.T) {
	_, err := config.Find(t.TempDir())

	require.Error(t, err)
	require.True(t, errors.Is(err, config.ErrNotFound))
}
