package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX, hem *pgxpool.Pool hem de pgx.Tx tarafından karşılanan ortak sorgu
// arayüzüdür. Repository'ler bu arayüz üzerinden çalışır; böylece aynı kod hem
// havuzla hem de bir transaction içinde çalışabilir. (sqlc de bu arayüzü üretir.)
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

// txCtxKey, aktif transaction'ı context'te taşımak için özel anahtar.
type txCtxKey struct{}

// TxManager, appshared.TxManager portunu ve DBTX çözümleyicisini uygular.
// Transaction, context aracılığıyla repository'lere görünmez şekilde taşınır;
// bu yüzden repository imzaları temiz kalır ve use-case'ler birden fazla
// repository çağrısını tek atomik işlemde birleştirebilir (Unit of Work).
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager, TxManager'ı havuzla kurar.
func NewTxManager(pool *pgxpool.Pool) *TxManager { return &TxManager{pool: pool} }

// DB, context'te aktif bir transaction varsa onu, yoksa havuzu döner.
// Repository'ler her sorgudan önce bunu çağırır.
func (m *TxManager) DB(ctx context.Context) DBTX {
	if tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx); ok {
		return tx
	}
	return m.pool
}

// WithinTx, fn'i tek bir transaction içinde çalıştırır. İç içe çağrılarda
// mevcut transaction yeniden kullanılır (savepoint yerine reuse; basit ve
// öngörülebilir). fn hata dönerse rollback, başarılıysa commit yapılır.
func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	// Zaten bir transaction içindeysek, yenisini açma.
	if _, ok := ctx.Value(txCtxKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}

	// Panic durumunda bile transaction'ın geri alınmasını garanti et.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	ctx = context.WithValue(ctx, txCtxKey{}, tx)
	if err = fn(ctx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}
