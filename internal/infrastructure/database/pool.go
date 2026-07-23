// Package database, PostgreSQL bağlantı havuzunu (pgxpool) kurar ve
// transaction yönetimi (Unit of Work) sağlar.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zatrano/gocore/internal/infrastructure/config"
)

// NewPool, yapılandırılmış ve optimize edilmiş bir pgxpool.Pool üretir.
// Bağlantı havuzu parametreleri (max/min conn, lifetime, idle) config'ten gelir.
func NewPool(ctx context.Context, cfg config.DB) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: config parse: %w", err)
	}

	// Havuz optimizasyonu.
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = 30 * time.Second
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	// Prepared statement cache: pgx varsayılan olarak statement'ları cache'ler.
	// Bu, tekrar eden sorgularda parse maliyetini düşürür ve SQL injection'a karşı
	// parametreli sorgu kullanımını zorunlu kılar. NOT: QueryExecMode sabitleri
	// iota'ya _ (0) ile başlar; bu yüzden 0 GEÇERSİZDİR ("unknown QueryExecMode").
	// Adlandırılmış sabit kullanılmalıdır.
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database: pool oluşturulamadı: %w", err)
	}

	// Bağlantıyı doğrula (fail-fast).
	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping başarısız: %w", err)
	}

	return pool, nil
}
