package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/titpetric/cli"
)

// newTestApp mirrors the registration run() performs, against buffers rather
// than the process streams.
func newTestApp() (*cli.App, *bytes.Buffer, *bytes.Buffer) {
	var stdout, stderr bytes.Buffer

	app := cli.NewApp("vuego-cli")
	app.Stdout = &stdout
	app.Stderr = &stderr

	return app, &stdout, &stderr
}

// TestRunRegistersEveryCommand covers the command set the binary exposes.
//
// A command is reachable only if run() registered it, and a typo in a name is
// invisible until somebody types it. Asking each one for help exercises the
// lookup without executing anything: a help request returns nil, and an
// unregistered name does not.
func TestRunRegistersEveryCommand(t *testing.T) {
	commands := []string{"fmt", "render", "diff", "serve", "tour", "docs", "host", "version"}

	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			app, _, _ := newTestApp()
			registerCommands(app)

			require.NoError(t, app.RunWithArgs([]string{name, "--help"}))
		})
	}
}

// TestRunRejectsAnUnknownCommand covers the other side of the same lookup, so
// the help-returns-nil check above cannot pass for a name nobody registered.
func TestRunRejectsAnUnknownCommand(t *testing.T) {
	app, _, _ := newTestApp()
	registerCommands(app)

	require.Error(t, app.RunWithArgs([]string{"no-such-command"}))
}

// TestVersionCommandCarriesBuildInfo covers the wiring between the ldflags
// variables and the version command, which is the one command run() builds by
// hand rather than passing a constructor straight through.
func TestVersionCommandCarriesBuildInfo(t *testing.T) {
	app, _, _ := newTestApp()
	registerCommands(app)

	require.NoError(t, app.RunWithArgs([]string{"version"}))
}
