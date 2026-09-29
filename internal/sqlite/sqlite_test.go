package sqlite_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/internal/model"
	"github.com/titpetric/vuego-cli/internal/sqlite"
)

// TestNew covers the constructor, which carries no configuration: every run
// gets a fresh in-memory database.
func TestNew(t *testing.T) {
	require.NotNil(t, sqlite.New())
}

// TestEvaluator covers the type satisfying model.Evaluator.
func TestEvaluator(t *testing.T) {
	var ev model.Evaluator = sqlite.New()
	require.NotNil(t, ev)
}

// TestHasSQL covers how the tour picks the SQL evaluator: the file set is
// searched for a .sql file, and .up.sql counts because it ends the same way.
func TestHasSQL(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  bool
	}{
		{"plain query", map[string]string{"index.sql": "SELECT 1;"}, true},
		{"migration only", map[string]string{"schema.up.sql": "CREATE TABLE t (a INT);"}, true},
		{"no sql", map[string]string{"index.vuego": "<div/>", "index.yaml": "a: 1"}, false},
		{"empty", map[string]string{}, false},
		{"nil", nil, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, sqlite.HasSQL(test.files))
		})
	}
}

// TestEvaluator_Eval covers the whole pipeline: the migration is applied first,
// then the query runs against what it created, and the rows come back as a
// table.
func TestEvaluator_Eval(t *testing.T) {
	res, err := sqlite.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"schema.up.sql": &fstest.MapFile{Data: []byte(
				`CREATE TABLE user_account (name TEXT);
INSERT INTO user_account (name) VALUES ('ada');`)},
			"index.sql": &fstest.MapFile{Data: []byte(`SELECT name FROM user_account;`)},
		},
	})

	require.NoError(t, err)
	require.Equal(t, model.ContentTypeHTML, res.ContentType)
	require.Contains(t, res.Content, "<th>name</th>")
	require.Contains(t, res.Content, "<td>ada</td>")
}

// TestEvaluator_EvalWithoutAMigration covers a snippet that builds its own
// schema inline, which has no migration to apply and must not be reported as
// one missing.
func TestEvaluator_EvalWithoutAMigration(t *testing.T) {
	res, err := sqlite.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"index.sql": &fstest.MapFile{Data: []byte(
				`CREATE TABLE t (a INT);
INSERT INTO t (a) VALUES (7);
SELECT a FROM t;`)},
		},
	})

	require.NoError(t, err)
	require.Contains(t, res.Content, "<td>7</td>")
}

// TestEvaluator_EvalWithNoQuery covers statements that select nothing, where
// there is no table to render and the run still succeeded.
func TestEvaluator_EvalWithNoQuery(t *testing.T) {
	res, err := sqlite.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"index.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE t (a INT);`)},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "<pre>ok</pre>", res.Content)
}

// TestEvaluator_EvalReportsTheFailingFile covers a broken statement: the error
// names the file it came from, which is the only way to tell which of several
// snippets failed.
func TestEvaluator_EvalReportsTheFailingFile(t *testing.T) {
	_, err := sqlite.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"index.sql": &fstest.MapFile{Data: []byte(`SELECT * FROM missing_table;`)},
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "index.sql")
}

// TestEvaluator_EvalEscapesResultValues covers a value carrying markup, which
// is rendered into a page and must not be able to close the cell it sits in.
func TestEvaluator_EvalEscapesResultValues(t *testing.T) {
	res, err := sqlite.New().Eval(context.Background(), model.Request{
		FS: fstest.MapFS{
			"index.sql": &fstest.MapFile{Data: []byte(`SELECT '<script>' AS tag;`)},
		},
	})

	require.NoError(t, err)
	require.NotContains(t, res.Content, "<script>")
	require.Contains(t, res.Content, "&lt;script&gt;")
}
