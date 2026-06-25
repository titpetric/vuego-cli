// Package exec evaluates shell snippets. Each non-empty line of the entry file
// is treated as a single command and executed via "bash -c". The result echoes
// each command prefixed with "$ " followed by its raw combined output, mirroring
// an interactive shell session.
package exec

import (
	"bytes"
	"context"
	"io/fs"
	"os/exec"
	"strings"

	"github.com/titpetric/vuego-cli/internal/model"
)

// Evaluator runs shell commands via "bash -c".
type Evaluator struct {
	// Shell is the shell binary used to run commands. Defaults to "bash".
	Shell string
}

// New returns a shell evaluator using bash.
func New() *Evaluator {
	return &Evaluator{Shell: "bash"}
}

// Eval reads req.Entry from req.FS and runs each non-empty line as a separate
// command. Command output (stdout and stderr combined) is returned as plain
// text, prefixed by the command that produced it.
func (e *Evaluator) Eval(ctx context.Context, req model.Request) (model.Result, error) {
	entry := req.Entry
	if entry == "" {
		entry = "index.sh"
	}
	data, err := fs.ReadFile(req.FS, entry)
	if err != nil {
		return model.Result{}, err
	}

	shell := e.Shell
	if shell == "" {
		shell = "bash"
	}

	var out bytes.Buffer
	for _, line := range strings.Split(string(data), "\n") {
		cmd := strings.TrimSpace(line)
		if cmd == "" {
			continue
		}
		out.WriteString("$ " + cmd + "\n")
		output, _ := exec.CommandContext(ctx, shell, "-c", cmd).CombinedOutput()
		out.Write(output)
		if len(output) > 0 && !bytes.HasSuffix(output, []byte("\n")) {
			out.WriteByte('\n')
		}
	}

	return model.Result{ContentType: model.ContentTypeText, Content: out.String()}, nil
}
