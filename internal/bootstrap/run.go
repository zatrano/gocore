package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/zatrano/gocore/internal/infrastructure/config"
)

// Run, yapılandırmayı yükler, HTTP sunucusunu (API + GoUI web) başlatır ve
// graceful shutdown yapar. cmd/server giriş noktası burayı çağırır.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := Build(ctx, cfg)
	if err != nil {
		return err
	}

	startPprof(app)

	serverErr := make(chan error, 1)
	go func() {
		if err := app.Server.Start(); err != nil {
			serverErr <- err
		}
	}()

	base := fmt.Sprintf("http://%s", cfg.HTTP.Addr())
	app.Logger.Info("uygulama hazır",
		"env", cfg.App.Environment,
		"version", cfg.App.Version,
		"web", base+"/",
		"api", base+"/api/v1",
		"docs", base+"/docs",
	)

	select {
	case err := <-serverErr:
		return fmt.Errorf("sunucu hatası: %w", err)
	case <-ctx.Done():
		app.Logger.Info("kapatma sinyali alındı, graceful shutdown başlıyor")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()

	if err := app.Server.Shutdown(shutdownCtx); err != nil {
		app.Logger.Error("http shutdown hatası", "error", err)
	}
	if err := app.Cleanup(shutdownCtx); err != nil {
		app.Logger.Error("cleanup hatası", "error", err)
	}

	app.Logger.Info("uygulama düzgünce kapatıldı")
	return nil
}
