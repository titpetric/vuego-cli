package vuego_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/internal/model"
	vuegoeval "github.com/titpetric/vuego-cli/internal/vuego"
)

// TestNew covers the constructor accepting a nil base filesystem, which is what
// a host with no shared components passes.
func TestNew(t *testing.T) {
	require.NotNil(t, vuegoeval.New(nil))
}

// TestEvaluator covers the type satisfying model.Evaluator.
func TestEvaluator(t *testing.T) {
	var ev model.Evaluator = vuegoeval.New(nil)
	require.NotNil(t, ev)
}

// TestEvaluator_Eval covers the default entry and the data file beside it: a
// snippet is one template plus its data, and neither is named on the request.
func TestEvaluator_Eval(t *testing.T) {
	res, err := vuegoeval.New(nil).Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"index.vuego": &fstest.MapFile{Data: []byte(`<div>{{ name }}</div>`)},
			"index.yaml":  &fstest.MapFile{Data: []byte(`name: World`)},
		},
	})

	require.NoError(t, err)
	require.Equal(t, model.ContentTypeHTML, res.ContentType)
	require.Equal(t, "<div>World</div>\n", res.Content)
}

// TestEvaluator_EvalDataFilePrecedence covers which data file wins when more
// than one sits beside the template: .yaml, then .yml, then .json.
func TestEvaluator_EvalDataFilePrecedence(t *testing.T) {
	res, err := vuegoeval.New(nil).Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"index.vuego": &fstest.MapFile{Data: []byte(`<div>{{ name }}</div>`)},
			"index.yaml":  &fstest.MapFile{Data: []byte(`name: yaml`)},
			"index.yml":   &fstest.MapFile{Data: []byte(`name: yml`)},
			"index.json":  &fstest.MapFile{Data: []byte(`{"name":"json"}`)},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "<div>yaml</div>\n", res.Content)
}

// TestEvaluator_EvalNamedEntry covers an entry other than index.vuego, and the
// data file matching that name rather than the default one.
func TestEvaluator_EvalNamedEntry(t *testing.T) {
	res, err := vuegoeval.New(nil).Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"card.vuego": &fstest.MapFile{Data: []byte(`<span>{{ title }}</span>`)},
			"card.json":  &fstest.MapFile{Data: []byte(`{"title":"Card"}`)},
			"index.yaml": &fstest.MapFile{Data: []byte(`title: Wrong`)},
		},
		Entry: "card.vuego",
	})

	require.NoError(t, err)
	require.Equal(t, "<span>Card</span>\n", res.Content)
}

// TestEvaluator_EvalWithoutData covers a template needing none, which renders
// rather than failing for a data file that was never there.
func TestEvaluator_EvalWithoutData(t *testing.T) {
	res, err := vuegoeval.New(nil).Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"index.vuego": &fstest.MapFile{Data: []byte(`<p>static</p>`)},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "<p>static</p>\n", res.Content)
}

// TestEvaluator_EvalUsesTheBaseFS covers what baseFS is for: a component the
// snippet includes but does not carry, which the host supplies once for every
// snippet it evaluates.
func TestEvaluator_EvalUsesTheBaseFS(t *testing.T) {
	baseFS := fstest.MapFS{
		"shared.vuego": &fstest.MapFile{Data: []byte(`<footer>shared</footer>`)},
	}

	res, err := vuegoeval.New(baseFS).Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"index.vuego": &fstest.MapFile{Data: []byte(`<template include="shared.vuego"></template>`)},
		},
	})

	require.NoError(t, err)
	require.Contains(t, res.Content, "shared")
}

// TestEvaluator_EvalMissingEntry covers the entry that is not in the file set.
func TestEvaluator_EvalMissingEntry(t *testing.T) {
	_, err := vuegoeval.New(nil).Eval(context.Background(), model.Request{FS: fstest.MapFS{}})
	require.Error(t, err)
}
