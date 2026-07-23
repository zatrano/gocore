package bootstrap

import (
	"errors"
	"net/http"
	"net/http/pprof"
	"time"
)

// startPprof, profilleme endpoint'lerini yalnızca localhost:6060 üzerinde ayrı
// bir mux'ta açar (DefaultServeMux'a kayıt yapmaz).
func startPprof(app *App) {
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		srv := &http.Server{
			Addr:              "127.0.0.1:6060",
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		}
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Logger.Warn("pprof sunucusu durdu", "error", err)
		}
	}()
}
