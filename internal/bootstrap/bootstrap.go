// Package bootstrap, uygulamanın Composition Root'udur: tüm bağımlılıklar burada
// (yalnızca burada) somut tiplerle oluşturulur ve manuel constructor injection
// ile birbirine bağlanır. Böylece diğer tüm katmanlar arayüzlere bağımlı kalır
// (Dependency Inversion) ve hiçbir global state kullanılmaz.
package bootstrap

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	httpadapter "github.com/zatrano/gocore/internal/adapters/http"
	"github.com/zatrano/gocore/internal/infrastructure/config"
	"github.com/zatrano/gocore/internal/infrastructure/logger"
	"github.com/zatrano/gocore/internal/infrastructure/storage"
	"github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/worker"
)

// App, kurulmuş uygulamayı ve kapanışta serbest bırakılacak kaynakları taşır.
type App struct {
	Config *config.Config
	Logger *slog.Logger
	Server *httpadapter.Server

	pool        *pgxpool.Pool
	workers     *worker.Pool
	shutdownFns []func(context.Context) error
	cancelBg    context.CancelFunc
}

// Build, tüm bağımlılık grafiğini oluşturur. Herhangi bir adımda hata olursa
// o ana kadar açılmış kaynaklar temizlenir (fail-safe).
func Build(ctx context.Context, cfg *config.Config) (*App, error) {
	log := logger.New(cfg.App.Environment, "info")

	app := &App{Config: cfg, Logger: log}
	g := &graph{cfg: cfg, log: log, app: app}

	if err := g.wireInfra(ctx); err != nil {
		_ = app.Cleanup(ctx)
		return nil, err
	}
	if err := g.wireRepos(ctx); err != nil {
		_ = app.Cleanup(ctx)
		return nil, err
	}
	if err := g.wireAuthz(ctx); err != nil {
		_ = app.Cleanup(ctx)
		return nil, err
	}
	g.wireSecurity()
	g.wireWorkers() //nolint:contextcheck // arka plan worker havuzu; shutdown Build ctx ile
	g.wireAuth()
	g.wireOutbox(ctx)
	g.wireUser()
	if err := g.wireHTTP(); err != nil {
		_ = app.Cleanup(ctx)
		return nil, err
	}
	g.wireGoUI()
	g.wireServer() //nolint:contextcheck // fiber middleware kendi context'ini yönetir

	return app, nil
}

// i18nTranslator, *i18n.Translator'ı application notification.Translator portuna
// uyarlar (adapter). Böylece use-case katmanı pkg/i18n'e doğrudan bağlı kalmaz.
type i18nTranslator struct{ tr *i18n.Translator }

func (a i18nTranslator) T(locale, key, fallback string, args ...any) string {
	return a.tr.T(i18n.Locale(locale), key, fallback, args...)
}

// toLocales, string dil listesini i18n.Locale dilimine dönüştürür.
func toLocales(in []string) []i18n.Locale {
	out := make([]i18n.Locale, 0, len(in))
	for _, s := range in {
		out = append(out, i18n.Locale(s))
	}
	return out
}

// storageOrTemp, yerel dosya depolamayı ./storage-data dizininde kurar.
func storageOrTemp(_ *config.Config) (*storage.Local, error) {
	return storage.NewLocal("storage-data")
}

// Cleanup, açık kaynakları ters sırada kapatır (graceful shutdown).
func (a *App) Cleanup(ctx context.Context) error {
	if a.cancelBg != nil {
		a.cancelBg()
	}
	if a.workers != nil {
		_ = a.workers.Shutdown(ctx)
	}
	for i := len(a.shutdownFns) - 1; i >= 0; i-- {
		if err := a.shutdownFns[i](ctx); err != nil && a.Logger != nil {
			a.Logger.Warn("shutdown fn error", slog.String("error", err.Error()))
		}
	}
	if a.pool != nil {
		a.pool.Close()
	}
	return nil
}
