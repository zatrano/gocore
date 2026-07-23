// Command seed, geliştirme/test için veritabanına başlangıç verisi ekler.
// Idempotent'tir: aynı e-posta zaten varsa atlar.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/zatrano/gocore/internal/adapters/persistence/postgres"
	"github.com/zatrano/gocore/internal/domain/user"
	"github.com/zatrano/gocore/internal/infrastructure/config"
	"github.com/zatrano/gocore/internal/infrastructure/database"
	"github.com/zatrano/gocore/internal/infrastructure/security"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "seed error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer pool.Close()

	txManager := database.NewTxManager(pool)
	repo := postgres.NewUserRepository(txManager)
	hasher := security.NewArgon2Hasher(security.DefaultArgon2Params())

	// Varsayılan admin kullanıcısı.
	email, err := user.NewEmail("zatrano@zatrano.com")
	if err != nil {
		return err
	}

	exists, err := repo.ExistsByEmail(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		fmt.Println("admin kullanıcısı zaten mevcut, atlanıyor")
		return nil
	}

	encoded, err := hasher.Hash(ctx, "ZATRANO")
	if err != nil {
		return err
	}
	hashed, err := user.NewHashedPassword(encoded)
	if err != nil {
		return err
	}

	loc, _ := user.ParsePreferredLocale("tr")
	admin, err := user.Register(email, "System Admin", hashed, user.RoleAdmin, loc, user.Phone{})
	if err != nil {
		return err
	}
	if err := admin.Activate(); err != nil && !errors.Is(err, user.ErrAlreadyActive) {
		return err
	}

	if err := repo.Save(ctx, admin); err != nil {
		return err
	}
	fmt.Println("admin kullanıcısı oluşturuldu ✓ (zatrano@zatrano.com / ZATRANO)")
	return nil
}
