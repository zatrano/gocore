package validation

import (
	"reflect"
	"strings"
)

const sanitizeTag = "sanitize"

// SanitizeStruct, `sanitize:"email|phone"` etiketli string alanları yerinde normalize eder.
func SanitizeStruct(v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return nil
	}
	var inv InvalidFields
	sanitizeValue(val.Elem(), &inv)
	if len(inv) > 0 {
		return inv
	}
	return nil
}

func sanitizeValue(val reflect.Value, inv *InvalidFields) {
	switch val.Kind() {
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			field := val.Type().Field(i)
			if field.PkgPath != "" {
				continue
			}
			fv := val.Field(i)
			if tag := field.Tag.Get(sanitizeTag); tag != "" && fv.Kind() == reflect.String && fv.CanSet() {
				normalized, tagName, err := normalizeByTag(tag, fv.String())
				if err != nil {
					*inv = append(*inv, InvalidField{Field: fieldName(field), Tag: tagName})
					continue
				}
				fv.SetString(normalized)
				continue
			}
			sanitizeValue(fv, inv)
		}
	case reflect.Pointer:
		if !val.IsNil() {
			sanitizeValue(val.Elem(), inv)
		}
	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			sanitizeValue(val.Index(i), inv)
		}
	}
}

func fieldName(field reflect.StructField) string {
	if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
		name, _, _ := strings.Cut(tag, ",")
		if name != "" {
			return name
		}
	}
	if tag := field.Tag.Get("form"); tag != "" && tag != "-" {
		return tag
	}
	return field.Name
}

func normalizeByTag(tag, raw string) (string, string, error) {
	switch tag {
	case "email":
		out, err := NormalizeEmail(raw)
		return out, "email", err
	case "phone":
		out, err := NormalizePhone(raw)
		return out, "phone", err
	default:
		return raw, tag, nil
	}
}
