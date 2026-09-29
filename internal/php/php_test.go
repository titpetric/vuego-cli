package php_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/internal/model"
	"github.com/titpetric/vuego-cli/internal/php"
)

// TestNew covers the evaluator carrying DefaultOptions, which is what confines
// a snippet's writes to the request filesystem.
func TestNew(t *testing.T) {
	require.Equal(t, php.DefaultOptions, php.New().Options)
}

// TestEvaluator covers the type satisfying model.Evaluator.
func TestEvaluator(t *testing.T) {
	var ev model.Evaluator = php.New()
	require.NotNil(t, ev)
}

// TestEvaluator_Eval covers the ordinary run: index.php is executed and what it
// printed comes back as HTML.
func TestEvaluator_Eval(t *testing.T) {
	res, err := php.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"index.php": &fstest.MapFile{Data: []byte(`<?php echo "Hello";`)},
		},
	})

	require.NoError(t, err)
	require.Equal(t, model.ContentTypeHTML, res.ContentType)
	require.Equal(t, "Hello", res.Content)
}

// TestEvaluator_EvalFallsBackToTheFirstPHPFile covers a snippet whose file is
// not called index.php. The frontend names the entry after the fence, and a
// lesson written with one differently named file still has to run.
func TestEvaluator_EvalFallsBackToTheFirstPHPFile(t *testing.T) {
	res, err := php.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"zeta.php":  &fstest.MapFile{Data: []byte(`<?php echo "zeta";`)},
			"alpha.php": &fstest.MapFile{Data: []byte(`<?php echo "alpha";`)},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "alpha", res.Content, "the lexically first .php file is the fallback")
}

// TestEvaluator_EvalHonoursAnExplicitEntry covers Entry winning over the
// fallback when the named file is really there.
func TestEvaluator_EvalHonoursAnExplicitEntry(t *testing.T) {
	res, err := php.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"alpha.php": &fstest.MapFile{Data: []byte(`<?php echo "alpha";`)},
			"chosen.php": &fstest.MapFile{
				Data: []byte(`<?php echo "chosen";`),
			},
		},
		Entry: "chosen.php",
	})

	require.NoError(t, err)
	require.Equal(t, "chosen", res.Content)
}

// TestEvaluator_EvalWithNoPHPFile covers a file set holding nothing to run,
// which is a load error rather than empty output.
func TestEvaluator_EvalWithNoPHPFile(t *testing.T) {
	_, err := php.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{"notes.txt": &fstest.MapFile{Data: []byte("nothing")}},
	})

	require.Error(t, err)
}
