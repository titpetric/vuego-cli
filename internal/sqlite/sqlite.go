// Package sqlite evaluates SQL snippets against an in-memory SQLite database.
// Migrations (*.up.sql / *.sqlite3) are applied first, then the remaining .sql
// files are executed; the final SELECT result is rendered as an HTML table.
//
// The logic was moved out of the tour package so it can be reused by any code
// block evaluation surface.
package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing/fstest"

	_ "modernc.org/sqlite"

	"github.com/go-bridget/mig/migrate"
	"github.com/jmoiron/sqlx"
	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/vuego-cli/internal/model"
)

// Evaluator runs SQL snippets.
type Evaluator struct{}

// New returns a SQL evaluator.
func New() *Evaluator {
	return &Evaluator{}
}

// Eval reads all files from req.FS and runs the SQL snippets, returning the
// result of the first non-empty query as an HTML table.
func (e *Evaluator) Eval(ctx context.Context, req model.Request) (model.Result, error) {
	files, err := readFiles(req.FS)
	if err != nil {
		return model.Result{}, err
	}
	out, err := render(ctx, files)
	if err != nil {
		return model.Result{}, err
	}
	return model.Result{ContentType: model.ContentTypeHTML, Content: out}, nil
}

// HasSQL reports whether any file in the set is a .sql file.
func HasSQL(files map[string]string) bool {
	for name := range files {
		if strings.HasSuffix(name, ".sql") {
			return true
		}
	}
	return false
}

// readFiles collects every regular file in fsys into a name->content map.
func readFiles(fsys fs.FS) (map[string]string, error) {
	files := make(map[string]string)
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		files[p] = string(data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func render(ctx context.Context, files map[string]string) (string, error) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		return "", err
	}
	defer db.Close()

	if err := migrateSQL(ctx, db, files); err != nil {
		return "", err
	}
	params, err := sqlParams(files)
	if err != nil {
		return "", err
	}
	for _, name := range sortedSQLFiles(files) {
		if strings.HasSuffix(name, ".up.sql") {
			continue
		}
		out, err := execSQL(ctx, db.DB, files[name], params)
		if err != nil {
			return "", fmt.Errorf("%s: %w", name, err)
		}
		if out != "" {
			return out, nil
		}
	}
	return "<pre>ok</pre>", nil
}

// migrateSQL applies the migrations in the file set to db. Both *.up.sql and
// *.sqlite3 count as migrations; the latter is renamed so mig's default
// *.up.sql pattern picks it up. Names are flattened to their base, which puts
// every migration at the root of the in-memory FS handed to mig.
//
// A snippet with no migrations at all is normal, and the empty set returns
// before Apply, which reports ErrNoMigrations for it.
func migrateSQL(ctx context.Context, db *sqlx.DB, files map[string]string) error {
	fsys := fstest.MapFS{}
	for name, content := range files {
		if strings.HasSuffix(name, ".up.sql") || strings.HasSuffix(name, ".sqlite3") {
			migrationName := name
			if strings.HasSuffix(name, ".sqlite3") {
				migrationName = strings.TrimSuffix(name, ".sqlite3") + ".up.sql"
			}
			fsys[path.Base(migrationName)] = &fstest.MapFile{Data: []byte(content)}
		}
	}
	if len(fsys) == 0 {
		return nil
	}

	m, err := migrate.NewManager(db, fsys, "tour")
	if err != nil {
		return err
	}
	applied, err := m.Apply(ctx)
	for _, item := range applied {
		slog.Default().Info("migration", "file", item.Filename, "status", item.Status)
	}
	return err
}

func sortedSQLFiles(files map[string]string) []string {
	var names []string
	for name := range files {
		if strings.HasSuffix(name, ".sql") {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(i, j int) bool {
		if path.Base(names[i]) == "index.sql" {
			return true
		}
		if path.Base(names[j]) == "index.sql" {
			return false
		}
		return names[i] < names[j]
	})
	return names
}

func execSQL(ctx context.Context, db *sql.DB, src string, params map[string]any) (string, error) {
	var last string
	for _, stmt := range splitSQL(src) {
		if stmt == "" {
			continue
		}
		boundStmt, args := bindSQL(stmt, params)
		lower := strings.ToLower(strings.TrimSpace(stmt))
		if strings.HasPrefix(lower, "select") || strings.HasPrefix(lower, "with") {
			out, err := queryHTML(ctx, db, boundStmt, args...)
			if err != nil {
				return "", err
			}
			last = out
			continue
		}
		if _, err := db.ExecContext(ctx, boundStmt, args...); err != nil {
			return "", err
		}
	}
	return last, nil
}

func sqlParams(files map[string]string) (map[string]any, error) {
	for name, content := range files {
		if strings.HasSuffix(name, ".params.json") || strings.HasSuffix(name, ".params.yaml") || strings.HasSuffix(name, ".params.yml") {
			var params map[string]any
			if err := yaml.Unmarshal([]byte(content), &params); err != nil {
				return nil, err
			}
			return params, nil
		}
	}
	return map[string]any{"id": int64(1), "1": int64(1)}, nil
}

var namedSQLParam = regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`)

func bindSQL(stmt string, params map[string]any) (string, []any) {
	var args []any
	stmt = namedSQLParam.ReplaceAllStringFunc(stmt, func(match string) string {
		name := strings.TrimPrefix(match, ":")
		args = append(args, params[name])
		return "?"
	})
	for n := 1; strings.Contains(stmt, "$"+strconv.Itoa(n)); n++ {
		stmt = strings.ReplaceAll(stmt, "$"+strconv.Itoa(n), "?")
		args = append(args, params[strconv.Itoa(n)])
	}
	for i := strings.Count(stmt, "?") - len(args); i > 0; i-- {
		key := strconv.Itoa(len(args) + 1)
		if len(args) == 0 {
			key = "id"
		}
		args = append(args, params[key])
	}
	return stmt, args
}

func splitSQL(src string) []string {
	var stmts []string
	var buf strings.Builder
	inSingle := false
	inDouble := false
	for _, r := range src {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case ';':
			if !inSingle && !inDouble {
				stmts = append(stmts, strings.TrimSpace(buf.String()))
				buf.Reset()
				continue
			}
		}
		buf.WriteRune(r)
	}
	if tail := strings.TrimSpace(buf.String()); tail != "" {
		stmts = append(stmts, tail)
	}
	return stmts
}

func queryHTML(ctx context.Context, db *sql.DB, stmt string, args ...any) (string, error) {
	rows, err := db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	buf.WriteString(`<style>
body{margin:0;padding:1rem;font-family:system-ui,sans-serif;color:#333}
.table{width:100%;border-collapse:collapse;font-size:14px}
.table th,.table td{padding:.5rem .75rem;border-bottom:1px solid #e5e7eb;text-align:left;vertical-align:top}
.table th{background:#f8fafc;color:#111827;font-weight:600}
.table tbody tr:nth-child(even){background:#f9fafb}
.table tbody tr:hover{background:#eef2ff}
</style>`)
	buf.WriteString("<table class=\"table\"><thead><tr>")
	for _, col := range cols {
		buf.WriteString("<th>" + html.EscapeString(col) + "</th>")
	}
	buf.WriteString("</tr></thead><tbody>")
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return "", err
		}
		buf.WriteString("<tr>")
		for _, val := range vals {
			buf.WriteString("<td>" + html.EscapeString(sqlValueString(val)) + "</td>")
		}
		buf.WriteString("</tr>")
	}
	if err := rows.Err(); err != nil && err != io.EOF {
		return "", err
	}
	buf.WriteString("</tbody></table>")
	return buf.String(), nil
}

func sqlValueString(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(x)
	default:
		return fmt.Sprint(x)
	}
}
