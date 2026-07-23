// Command migrate, veritabanı şema migration'larını yönetir.
// Kullanım: migrate up | migrate down [steps]
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/zatrano/gocore/internal/infrastructure/config"
	"github.com/zatrano/gocore/internal/infrastructure/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("kullanım: migrate <up|down> [steps]")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch os.Args[1] {
	case "up":
		if err := database.MigrateUp(cfg.DB); err != nil {
			return err
		}
		fmt.Println("migration'lar uygulandı ✓")
	case "down":
		steps := 0
		if len(os.Args) >= 3 {
			steps, _ = strconv.Atoi(os.Args[2])
		}
		if err := database.MigrateDown(cfg.DB, steps); err != nil {
			return err
		}
		fmt.Println("migration'lar geri alındı ✓")
	default:
		return fmt.Errorf("bilinmeyen komut: %s (up|down)", os.Args[1])
	}
	return nil
}
