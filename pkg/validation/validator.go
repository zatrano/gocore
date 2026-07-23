package validation

import (
	"github.com/go-playground/validator/v10"
)

// Register, ortak doğrulama tag'lerini validator örneğine kaydeder.
func Register(v *validator.Validate) {
	_ = v.RegisterValidation("phone", validatePhone)
	_ = v.RegisterValidation("phone_required", validatePhoneRequired)
}

func validatePhone(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	if s == "" {
		return true
	}
	_, err := NormalizePhone(s)
	return err == nil
}

func validatePhoneRequired(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	if s == "" {
		return false
	}
	_, err := NormalizePhone(s)
	return err == nil
}

// Validator, go-playground/validator ve test double'ları kapsar.
type Validator interface {
	Struct(any) error
}

// Check, önce sanitize sonra struct doğrulaması yapar.
func Check(v Validator, dst any) error {
	if err := SanitizeStruct(dst); err != nil {
		return err
	}
	return v.Struct(dst)
}
