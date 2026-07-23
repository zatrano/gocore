package recipients_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/zatrano/gocore/pkg/recipients"
)

func TestParseCSV_HeaderMapping(t *testing.T) {
	in := "email,phone,name,locale\n" +
		"a@b.com,+905551112233,Ali,tr\n" +
		"c@d.com,+905553334455,Ayse,en\n"
	got, err := recipients.ParseCSV(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("satır sayısı = %d, beklenen 2", len(got))
	}
	if got[0].Email != "a@b.com" || got[0].Phone != "+905551112233" || got[0].Name != "Ali" || got[0].Locale != "tr" {
		t.Fatalf("ilk kayıt yanlış: %+v", got[0])
	}
	if got[1].Line != 3 {
		t.Fatalf("ikinci kaydın satır no = %d, beklenen 3", got[1].Line)
	}
}

func TestParseCSV_TurkishHeadersAndBOM(t *testing.T) {
	in := "\ufefftelefon,ad,e-posta\n" +
		"+905551112233,Veli,v@x.com\n"
	got, err := recipients.ParseCSV(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if got[0].Phone != "+905551112233" || got[0].Name != "Veli" || got[0].Email != "v@x.com" {
		t.Fatalf("türkçe başlık eşlemesi yanlış: %+v", got[0])
	}
}

func TestParseCSV_SkipsEmptyRows(t *testing.T) {
	in := "email\n\na@b.com\n\n"
	got, err := recipients.ParseCSV(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("boş satırlar atlanmalı, alınan %d", len(got))
	}
}

func TestParseCSV_NoRecognizedColumns(t *testing.T) {
	in := "foo,bar\n1,2\n"
	_, err := recipients.ParseCSV(strings.NewReader(in))
	if !errors.Is(err, recipients.ErrNoRecognizedColumns) {
		t.Fatalf("ErrNoRecognizedColumns bekleniyordu, alınan: %v", err)
	}
}

func TestParseCSV_EmptyFile(t *testing.T) {
	_, err := recipients.ParseCSV(strings.NewReader(""))
	if !errors.Is(err, recipients.ErrEmptyFile) {
		t.Fatalf("ErrEmptyFile bekleniyordu, alınan: %v", err)
	}
}

func TestParse_UnsupportedFormat(t *testing.T) {
	_, err := recipients.Parse("list.txt", strings.NewReader("x"))
	if !errors.Is(err, recipients.ErrUnsupportedFormat) {
		t.Fatalf("ErrUnsupportedFormat bekleniyordu, alınan: %v", err)
	}
}

func TestParseXLSX_HeaderMapping(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetSheetRow(sheet, "A1", &[]any{"email", "phone", "name"})
	_ = f.SetSheetRow(sheet, "A2", &[]any{"a@b.com", "+905551112233", "Ali"})
	_ = f.SetSheetRow(sheet, "A3", &[]any{"c@d.com", "+905553334455", "Ayse"})

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write xlsx: %v", err)
	}

	got, err := recipients.Parse("list.xlsx", &buf)
	if err != nil {
		t.Fatalf("Parse xlsx: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("satır sayısı = %d, beklenen 2", len(got))
	}
	if got[1].Email != "c@d.com" || got[1].Name != "Ayse" {
		t.Fatalf("ikinci kayıt yanlış: %+v", got[1])
	}
}
