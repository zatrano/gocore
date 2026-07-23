package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zatrano/gocore/internal/infrastructure/config"
	"github.com/zatrano/gocore/migrations"
)

// schema_migrations: tek satırda current version + dirty bayrağı tutulur.
const schemaMigrationsTable = "schema_migrations"

var migrationFileRE = regexp.MustCompile(`^(\d+)_.+\.(up|down)\.sql$`)

type migrationFile struct {
	Version int64
	Name    string
	UpSQL   string
	DownSQL string
}

// MigrateUp, bekleyen tüm migration'ları uygular.
func MigrateUp(cfg config.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := newMigratePool(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := ensureSchemaMigrations(ctx, pool); err != nil {
		return err
	}
	files, err := loadMigrations(migrations.FS)
	if err != nil {
		return err
	}
	applied, dirty, err := currentVersion(ctx, pool)
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("migrate up: schema_migrations dirty=true (version=%d); önce düzeltin", applied)
	}

	for _, m := range files {
		if m.Version <= applied {
			continue
		}
		if err := applyUp(ctx, pool, m); err != nil {
			return fmt.Errorf("migrate up %d: %w", m.Version, err)
		}
	}
	return nil
}

// MigrateDown, son migration'ı (veya steps kadarını) geri alır. steps<=0 ise hepsi.
func MigrateDown(cfg config.DB, steps int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := newMigratePool(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := ensureSchemaMigrations(ctx, pool); err != nil {
		return err
	}
	files, err := loadMigrations(migrations.FS)
	if err != nil {
		return err
	}
	byVer := make(map[int64]migrationFile, len(files))
	for _, m := range files {
		byVer[m.Version] = m
	}

	for {
		applied, dirty, err := currentVersion(ctx, pool)
		if err != nil {
			return err
		}
		if dirty {
			return fmt.Errorf("migrate down: schema_migrations dirty=true (version=%d); önce düzeltin", applied)
		}
		if applied == 0 {
			return nil
		}
		m, ok := byVer[applied]
		if !ok {
			return fmt.Errorf("migrate down: gömülü dosyalarda version %d yok", applied)
		}
		if strings.TrimSpace(m.DownSQL) == "" {
			return fmt.Errorf("migrate down %d: down SQL yok", m.Version)
		}
		prev := previousVersion(files, applied)
		if err := applyDown(ctx, pool, m, prev); err != nil {
			return fmt.Errorf("migrate down %d: %w", m.Version, err)
		}
		if steps > 0 {
			steps--
			if steps == 0 {
				return nil
			}
		}
	}
}

func previousVersion(files []migrationFile, current int64) int64 {
	var prev int64
	for _, m := range files {
		if m.Version < current && m.Version > prev {
			prev = m.Version
		}
	}
	return prev
}

// newMigratePool, çoklu SQL ifadeli .sql dosyaları için SimpleProtocol kullanır.
func newMigratePool(ctx context.Context, cfg config.DB) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("migrate: config: %w", err)
	}
	poolCfg.MaxConns = 2
	poolCfg.MinConns = 0
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("migrate: pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: ping: %w", err)
	}
	return pool, nil
}

func ensureSchemaMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS `+schemaMigrationsTable+` (
    version BIGINT PRIMARY KEY,
    dirty   BOOLEAN NOT NULL
)`)
	if err != nil {
		return fmt.Errorf("migrate: schema_migrations: %w", err)
	}
	return nil
}

func currentVersion(ctx context.Context, pool *pgxpool.Pool) (int64, bool, error) {
	var version int64
	var dirty bool
	err := pool.QueryRow(ctx,
		`SELECT version, dirty FROM `+schemaMigrationsTable+` LIMIT 1`,
	).Scan(&version, &dirty)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("migrate: current version: %w", err)
	}
	return version, dirty, nil
}

func applyUp(ctx context.Context, pool *pgxpool.Pool, m migrationFile) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM `+schemaMigrationsTable); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO `+schemaMigrationsTable+` (version, dirty) VALUES ($1, TRUE)`, m.Version); err != nil {
		return err
	}
	if strings.TrimSpace(m.UpSQL) != "" {
		if _, err := tx.Exec(ctx, m.UpSQL); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE `+schemaMigrationsTable+` SET dirty = FALSE WHERE version = $1`, m.Version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func applyDown(ctx context.Context, pool *pgxpool.Pool, m migrationFile, prev int64) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE `+schemaMigrationsTable+` SET dirty = TRUE WHERE version = $1`, m.Version); err != nil {
		return err
	}
	if strings.TrimSpace(m.DownSQL) != "" {
		if _, err := tx.Exec(ctx, m.DownSQL); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM `+schemaMigrationsTable); err != nil {
		return err
	}
	if prev > 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO `+schemaMigrationsTable+` (version, dirty) VALUES ($1, FALSE)`, prev); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func loadMigrations(fsys fs.FS) ([]migrationFile, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("migrate: list: %w", err)
	}

	type pair struct {
		up, down string
		name     string
	}
	byVer := map[int64]*pair{}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		m := migrationFileRE.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		ver, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("migrate: version %q: %w", name, err)
		}
		p := byVer[ver]
		if p == nil {
			base := strings.TrimSuffix(name, ".up.sql")
			base = strings.TrimSuffix(base, ".down.sql")
			p = &pair{name: base}
			byVer[ver] = p
		}
		body, err := fs.ReadFile(fsys, path.Clean(name))
		if err != nil {
			return nil, err
		}
		switch m[2] {
		case "up":
			p.up = string(body)
		case "down":
			p.down = string(body)
		}
	}

	out := make([]migrationFile, 0, len(byVer))
	for ver, p := range byVer {
		if p.up == "" {
			return nil, fmt.Errorf("migrate: version %d için .up.sql yok", ver)
		}
		out = append(out, migrationFile{Version: ver, Name: p.name, UpSQL: p.up, DownSQL: p.down})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}
