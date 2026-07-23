package tabular_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zatrano/gocore/pkg/tabular"
)

func TestWriteCSV(t *testing.T) {
	var buf bytes.Buffer
	err := tabular.WriteCSV(&buf, []string{"email", "name"}, [][]string{
		{"a@b.com", "Ali"},
		{"c@d.com", "Ayşe"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.HasPrefix(got, "\ufeff") {
		t.Fatal("expected UTF-8 BOM")
	}
	if !strings.Contains(got, "email,name") || !strings.Contains(got, "a@b.com,Ali") {
		t.Fatalf("unexpected csv: %q", got)
	}
}

func TestWriteXLSX(t *testing.T) {
	var buf bytes.Buffer
	err := tabular.WriteXLSX(&buf, "Users", []string{"id", "email"}, [][]string{{"1", "a@b.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() < 100 {
		t.Fatalf("xlsx too small: %d", buf.Len())
	}
}

func TestParseFormat(t *testing.T) {
	if tabular.ParseFormat("xlsx") != tabular.FormatXLSX {
		t.Fatal("xlsx")
	}
	if tabular.ParseFormat("") != tabular.FormatCSV {
		t.Fatal("default csv")
	}
}
