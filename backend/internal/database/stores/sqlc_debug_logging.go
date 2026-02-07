package stores

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type sqliteDebugDBTX struct {
	inner sqlitedb.DBTX
}

func (d sqliteDebugDBTX) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	res, err := d.inner.ExecContext(ctx, query, args...)
	logDBQuery(ctx, "sqlite", "exec", query, args, time.Since(start), err)
	return res, err
}

func (d sqliteDebugDBTX) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	start := time.Now()
	stmt, err := d.inner.PrepareContext(ctx, query)
	logDBQuery(ctx, "sqlite", "prepare", query, nil, time.Since(start), err)
	return stmt, err
}

func (d sqliteDebugDBTX) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := d.inner.QueryContext(ctx, query, args...)
	logDBQuery(ctx, "sqlite", "query", query, args, time.Since(start), err)
	return rows, err
}

func (d sqliteDebugDBTX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// sql.Row exposes no error until Scan(), so we log call metadata here.
	logDBQuery(ctx, "sqlite", "query_row", query, args, 0, nil)
	return d.inner.QueryRowContext(ctx, query, args...)
}

type pgDebugDBTX struct {
	inner pgdb.DBTX
}

func (d pgDebugDBTX) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	start := time.Now()
	tag, err := d.inner.Exec(ctx, query, args...)
	logDBQuery(ctx, "postgres", "exec", query, args, time.Since(start), err)
	return tag, err
}

func (d pgDebugDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	start := time.Now()
	rows, err := d.inner.Query(ctx, query, args...)
	logDBQuery(ctx, "postgres", "query", query, args, time.Since(start), err)
	return rows, err
}

func (d pgDebugDBTX) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return pgDebugRow{
		ctx:   ctx,
		start: time.Now(),
		query: query,
		args:  args,
		inner: d.inner.QueryRow(ctx, query, args...),
	}
}

type pgDebugRow struct {
	ctx   context.Context
	start time.Time
	query string
	args  []interface{}
	inner pgx.Row
}

func (r pgDebugRow) Scan(dest ...interface{}) error {
	err := r.inner.Scan(dest...)
	logDBQuery(r.ctx, "postgres", "query_row", r.query, r.args, time.Since(r.start), err)
	return err
}

func logDBQuery(ctx context.Context, driver, op, query string, args []interface{}, duration time.Duration, err error) {
	if !slog.Default().Enabled(ctx, slog.LevelDebug) {
		return
	}

	attrs := []any{
		"driver", driver,
		"op", op,
		"sql", compactSQL(query),
		"args_count", len(args),
		"args", summarizeDBArgs(args),
	}
	if duration > 0 {
		attrs = append(attrs, "duration_ms", duration.Milliseconds())
	}
	if err != nil {
		if isNotFound(err) {
			attrs = append(attrs, "not_found", true)
		} else {
			attrs = append(attrs, "error", err)
		}
	}

	slog.DebugContext(ctx, "Database query", attrs...)
}

func compactSQL(query string) string {
	if query == "" {
		return ""
	}
	clean := strings.Join(strings.Fields(query), " ")
	const maxLen = 1000
	if len(clean) > maxLen {
		return clean[:maxLen] + "...(truncated)"
	}
	return clean
}

func summarizeDBArgs(args []interface{}) []any {
	if len(args) == 0 {
		return nil
	}
	out := make([]any, 0, len(args))
	for _, arg := range args {
		out = append(out, summarizeDBArg(arg))
	}
	return out
}

func summarizeDBArg(arg interface{}) any {
	switch v := arg.(type) {
	case nil:
		return nil
	case string:
		const maxLen = 128
		if len(v) > maxLen {
			return v[:maxLen] + "...(truncated)"
		}
		return v
	case []byte:
		return fmt.Sprintf("<bytes:%d>", len(v))
	case time.Time:
		return v.UTC().Format(time.RFC3339Nano)
	default:
		return v
	}
}
