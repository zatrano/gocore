// Command server, tek HTTP sürecinin giriş noktasıdır: JSON API (/api/v1) ve
// GoUI web arayüzü (/, /auth/login, /dashboard, …) aynı Fiber sunucusunda çalışır.
package main

import (
	"fmt"
	"os"

	"github.com/zatrano/gocore/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}
