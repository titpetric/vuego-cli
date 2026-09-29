package exec_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/internal/exec"
	"github.com/titpetric/vuego-cli/internal/model"
)

// TestNew covers the default shell, which is what Eval falls back to for an
// evaluator built any other way.
func TestNew(t *testing.T) {
	require.Equal(t, "bash", exec.New().Shell)
}

// TestEvaluator covers the type satisfying model.Evaluator, which is what lets
// codeblock.Service hold it.
func TestEvaluator(t *testing.T) {
	var ev model.Evaluator = exec.New()
	require.NotNil(t, ev)
}

// TestEvaluator_Eval covers the transcript Eval builds: each command echoed
// behind "$ " and its combined output underneath, which is what makes the
// output read as a shell session rather than a wall of text.
func TestEvaluator_Eval(t *testing.T) {
	res, err := exec.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"index.sh": &fstest.MapFile{Data: []byte("echo one\n\necho two\n")},
		},
	})

	require.NoError(t, err)
	require.Equal(t, model.ContentTypeText, res.ContentType)
	require.Equal(t, "$ echo one\none\n$ echo two\ntwo\n", res.Content)
}

// TestEvaluator_EvalCapturesStderr covers a failing command: the exit status is
// not an error of the evaluation, and what the command wrote to stderr is part
// of the transcript.
func TestEvaluator_EvalCapturesStderr(t *testing.T) {
	res, err := exec.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"index.sh": &fstest.MapFile{Data: []byte("echo boom >&2; exit 3")},
		},
	})

	require.NoError(t, err)
	require.Contains(t, res.Content, "boom")
}

// TestEvaluator_EvalMissingEntry covers the entry that is not there, which is
// the one failure Eval does report.
func TestEvaluator_EvalMissingEntry(t *testing.T) {
	_, err := exec.New().Eval(context.Background(), model.Request{FS: fstest.MapFS{}})
	require.Error(t, err)
}

// TestEvaluator_EvalAppendsAMissingNewline covers output that does not end in
// one, so the next "$ " still starts on a line of its own.
func TestEvaluator_EvalAppendsAMissingNewline(t *testing.T) {
	res, err := exec.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"run.sh": &fstest.MapFile{Data: []byte("printf no-newline\necho after\n")},
		},
		Entry: "run.sh",
	})

	require.NoError(t, err)
	require.Equal(t, "$ printf no-newline\nno-newline\n$ echo after\nafter\n", res.Content)
}
