package migrations

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// embeds the migration files in the executable
//
//go:embed sql/*.sql
var files embed.FS

type migration struct {
	version int64
	name    string
	sql     string
}

func Run(ctx context.Context, db *pgxpool.Pool) error {
	migrations, err := load()
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
CREATE TABLE IF NOT EXISTS goose_db_version (
    id BIGSERIAL PRIMARY KEY,
    version_id BIGINT NOT NULL,
    is_applied BOOLEAN NOT NULL,
    tstamp TIMESTAMPTZ NOT NULL DEFAULT now()
)`); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(726155424001)`); err != nil {
		return err
	}

	applied, err := appliedVersions(ctx, tx)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if applied[migration.version] {
			continue
		}
		if _, err := tx.Exec(ctx, migration.sql); err != nil {
			return fmt.Errorf("migration %s failed: %w", migration.name, err)
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO goose_db_version (version_id, is_applied)
VALUES ($1, true)`, migration.version); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func appliedVersions(ctx context.Context, tx pgx.Tx) (map[int64]bool, error) {
	rows, err := tx.Query(ctx, `
SELECT version_id, is_applied
FROM goose_db_version
ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := map[int64]bool{}
	for rows.Next() {
		var version int64
		var isApplied bool
		if err := rows.Scan(&version, &isApplied); err != nil {
			return nil, err
		}
		applied[version] = isApplied
	}
	return applied, rows.Err()
}

func load() ([]migration, error) {
	names, err := files.ReadDir("sql")
	if err != nil {
		return nil, err
	}

	migrations := make([]migration, 0, len(names))
	for _, entry := range names {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := parseVersion(entry.Name())
		if err != nil {
			return nil, err
		}
		body, err := files.ReadFile(filepath.ToSlash(filepath.Join("sql", entry.Name())))
		if err != nil {
			return nil, err
		}
		up, err := upSection(string(body))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		migrations = append(migrations, migration{
			version: version,
			name:    entry.Name(),
			sql:     up,
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})
	return migrations, nil
}

func parseVersion(name string) (int64, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("invalid migration filename %q", name)
	}
	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid migration version %q: %w", name, err)
	}
	return version, nil
}

func upSection(sql string) (string, error) {
	upMarker := "-- +goose Up"
	downMarker := "-- +goose Down"
	upIndex := strings.Index(sql, upMarker)
	if upIndex < 0 {
		return "", fmt.Errorf("missing up marker")
	}
	body := sql[upIndex+len(upMarker):]
	if downIndex := strings.Index(body, downMarker); downIndex >= 0 {
		body = body[:downIndex]
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return "", fmt.Errorf("empty up migration")
	}
	return body, nil
}
