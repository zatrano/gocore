package tabular

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Format, dışa aktarma biçimi.
type Format string

const (
	FormatCSV  Format = "csv"
	FormatXLSX Format = "xlsx"
)

// ParseFormat, query/form değerinden biçim seçer; boşsa csv.
func ParseFormat(raw string) Format {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "xlsx", "excel", "xls":
		return FormatXLSX
	default:
		return FormatCSV
	}
}

// ContentType, HTTP Content-Type.
func (f Format) ContentType() string {
	if f == FormatXLSX {
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	return "text/csv; charset=utf-8"
}

// Extension, dosya uzantısı.
func (f Format) Extension() string {
	if f == FormatXLSX {
		return "xlsx"
	}
	return "csv"
}

// Write, headers + rows'u seçilen biçimde yazar.
func Write(w io.Writer, format Format, sheet string, headers []string, rows [][]string) error {
	if format == FormatXLSX {
		return WriteXLSX(w, sheet, headers, rows)
	}
	return WriteCSV(w, headers, rows)
}

// WriteCSV, UTF-8 BOM'lu CSV yazar (Excel uyumu).
func WriteCSV(w io.Writer, headers []string, rows [][]string) error {
	if _, err := w.Write([]byte("\ufeff")); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(headers); err != nil {
		return err
	}
	for _, row := range rows {
		cells := make([]string, len(headers))
		copy(cells, row)
		if err := cw.Write(cells); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteXLSX, tek sayfalık Excel yazar.
func WriteXLSX(w io.Writer, sheet string, headers []string, rows [][]string) error {
	if sheet == "" {
		sheet = "Sheet1"
	}
	f := excelize.NewFile()
	defer f.Close()
	name := f.GetSheetName(0)
	if err := f.SetSheetName(name, sheet); err != nil {
		return err
	}
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return err
		}
	}
	for r, row := range rows {
		for c := 0; c < len(headers); c++ {
			val := ""
			if c < len(row) {
				val = row[c]
			}
			cell, err := excelize.CoordinatesToCellName(c+1, r+2)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cell, val); err != nil {
				return err
			}
		}
	}
	if _, err := f.WriteTo(w); err != nil {
		return fmt.Errorf("xlsx write: %w", err)
	}
	return nil
}
