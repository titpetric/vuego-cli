package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestVersionDefaults covers the values a build without ldflags carries.
//
// They are not decoration: version.Run compares Commit, CommitTime and Branch
// against "unknown" to decide whether to print those lines at all, and Modified
// against "true". A default changed here silently changes what `vuego-cli
// version` prints for a plain `go build`.
func TestVersionDefaults(t *testing.T) {
	require.Equal(t, "dev", Version)
	require.Equal(t, "unknown", Commit)
	require.Equal(t, "unknown", CommitTime)
	require.Equal(t, "unknown", Branch)
	require.Equal(t, "false", Modified)
}
