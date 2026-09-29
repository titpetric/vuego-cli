package model_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/internal/model"
)

// stubEvaluator is the smallest thing that satisfies model.Evaluator, and
// exists to prove the interface is implementable as written.
type stubEvaluator struct {
	got model.Request
}

func (s *stubEvaluator) Eval(_ context.Context, req model.Request) (model.Result, error) {
	s.got = req
	return model.Result{ContentType: model.ContentTypeText, Content: "ok"}, nil
}

// TestEvaluator covers the contract every evaluator in internal/php,
// internal/vuego, internal/exec and internal/sqlite is held to: a Request goes
// in whole and a Result comes back.
func TestEvaluator(t *testing.T) {
	fsys := fstest.MapFS{"index.php": &fstest.MapFile{Data: []byte("<?php")}}
	stub := &stubEvaluator{}

	var ev model.Evaluator = stub
	res, err := ev.Eval(context.Background(), model.Request{FS: fsys, Entry: "index.php"})

	require.NoError(t, err)
	require.Equal(t, model.ContentTypeText, res.ContentType)
	require.Equal(t, "ok", res.Content)
	require.Equal(t, "index.php", stub.got.Entry)
	require.Equal(t, fsys, stub.got.FS)
}

// TestRequest covers the zero value, which is what a caller that forgot to
// nominate an entry hands over. Every evaluator reads Entry == "" as "use my
// default", so the zero value has to stay distinguishable.
func TestRequest(t *testing.T) {
	var req model.Request

	require.Nil(t, req.FS)
	require.Equal(t, "", req.Entry)
}

// TestResult covers the content types the frontend switches on to decide
// whether to render into an iframe, a <pre> or a table.
func TestResult(t *testing.T) {
	require.Equal(t, "text/html", model.ContentTypeHTML)
	require.Equal(t, "text/plain", model.ContentTypeText)
	require.Equal(t, "application/json", model.ContentTypeJSON)

	var res model.Result
	require.Equal(t, "", res.ContentType)
	require.Equal(t, "", res.Content)
}
