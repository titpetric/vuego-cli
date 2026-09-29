package version_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/commands/version"
)

// TestInfo covers the zero value, which is what a caller building the struct
// field by field starts from. Run reads every field as a string and compares
// three of them against "unknown", so an empty field is not the same as an
// absent one.
func TestInfo(t *testing.T) {
	var info version.Info

	require.Equal(t, "", info.Version)
	require.Equal(t, "", info.Commit)
	require.Equal(t, "", info.CommitTime)
	require.Equal(t, "", info.Branch)
	require.Equal(t, "", info.Modified)
}

// TestNewWithInfo covers the command the binary registers: its name and title,
// and that running it carries the info it was built with through to Run.
func TestNewWithInfo(t *testing.T) {
	cmd := version.NewWithInfo(version.Info{Version: "1.2.3"})

	require.Equal(t, "version", cmd.Name)
	require.Equal(t, version.Name, cmd.Title)
	require.NotNil(t, cmd.Run)
	require.NoError(t, cmd.Run(context.Background(), nil))
}

// TestRun covers the reporting itself over the shapes the ldflags produce: a
// build with no VCS information, one with all of it, and a commit long enough
// to be shortened.
func TestRun(t *testing.T) {
	tests := []struct {
		name string
		info version.Info
	}{
		{"unset build", version.Info{
			Version:    "dev",
			Commit:     "unknown",
			CommitTime: "unknown",
			Branch:     "unknown",
			Modified:   "false",
		}},
		{"tagged build", version.Info{
			Version:    "v1.0.0",
			Commit:     "3d27913068b109facd8070b0ec47599d1054f49a",
			CommitTime: "2026-09-28 09:48:57 +0200",
			Branch:     "main",
			Modified:   "false",
		}},
		{"dirty tree", version.Info{
			Version:  "v1.0.0",
			Commit:   "abc123",
			Branch:   "main",
			Modified: "true",
		}},
		{"zero value", version.Info{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, version.Run(test.info))
		})
	}
}
