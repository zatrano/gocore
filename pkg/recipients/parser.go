package recipients

import (
	"encoding/csv"
	"io"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Parse, dosya uzantısına göre CSV veya XLSX ayrıştırır. Başlık satırı zorunludur.
func Parse(filename string, r io.Reader) ([]Row, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".csv":
		return ParseCSV(r)
	case ".xlsx":
		return ParseXLSX(r)
	default:
		return nil, ErrUnsupportedFormat
	}
}

// ParseCSV, CSV içeriğini ayrıştırır. İlk satır başlıktır.
func ParseCSV(r io.Reader) ([]Row, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, ErrEmptyFile
		}
		return nil, err
	}
	idx, ok := newColumnIndex(header)
	if !ok {
		return nil, ErrNoRecognizedColumns
	}

	out := make([]Row, 0, 64)
	line := 1
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line++
		if isEmptyRow(rec) {
			continue
		}
		out = append(out, idx.row(rec, line))
		if len(out) > MaxRows {
			return nil, ErrTooManyRows
		}
	}
	if len(out) == 0 {
		return nil, ErrEmptyFile
	}
	return out, nil
}

// ParseXLSX, Excel dosyasının ilk sayfasını ayrıştırır. İlk satır başlıktır.
func ParseXLSX(r io.Reader) ([]Row, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, ErrEmptyFile
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, ErrEmptyFile
	}

	idx, ok := newColumnIndex(rows[0])
	if !ok {
		return nil, ErrNoRecognizedColumns
	}

	out := make([]Row, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		line := i + 1
		if isEmptyRow(rows[i]) {
			continue
		}
		out = append(out, idx.row(rows[i], line))
		if len(out) > MaxRows {
			return nil, ErrTooManyRows
		}
	}
	if len(out) == 0 {
		return nil, ErrEmptyFile
	}
	return out, nil
}
