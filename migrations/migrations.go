// Package migrations, SQL migration dosyalarını binary'ye gömer (embed). Bu
// sayede migration'lar uygulamayla birlikte dağıtılır; ayrı dosya kopyalamaya
// gerek kalmaz.
//
// Sıra: 000001_schema → 000002_seed.
package migrations

import "embed"

// FS, tüm .sql migration dosyalarını içeren gömülü dosya sistemidir.
//
//go:embed *.sql
var FS embed.FS
