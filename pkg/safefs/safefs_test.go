package safefs_test

import (
	"strings"
	"testing"

	"github.com/zatrano/gocore/pkg/safefs"
)

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"rapor.pdf", "rapor.pdf", false},
		{"../../etc/passwd", "passwd", false},
		{"..\\..\\windows\\system32\\cmd.exe", "cmd.exe", false},
		{"my file (1).png", "my_file_1_.png", false},
		{"...", "", true},
		{"", "", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got, err := safefs.SanitizeFilename(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("hata bekleniyordu (in=%q)", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("beklenmeyen hata: %v", err)
			}
			if got != tc.want {
				t.Errorf("SanitizeFilename(%q) = %q, beklenen %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSafeJoin_BlocksTraversal(t *testing.T) {
	t.Parallel()
	base := "/srv/data"
	if _, err := safefs.SafeJoin(base, "../../../etc/passwd"); err == nil {
		t.Error("path traversal engellenmedi")
	}
	got, err := safefs.SafeJoin(base, "sub/file.txt")
	if err != nil {
		t.Fatalf("geçerli yol reddedildi: %v", err)
	}
	if !strings.Contains(filepathSlash(got), "sub/file.txt") {
		t.Errorf("beklenmeyen yol: %s", got)
	}
}

// filepathSlash, platformdan bağımsız karşılaştırma için ayraçları normalize eder.
func filepathSlash(p string) string { return strings.ReplaceAll(p, "\\", "/") }
