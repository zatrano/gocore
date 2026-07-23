package shared

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/pkg/validation"
)

// ValidateForm, form bağlar, sanitize eder ve doğrular.
func ValidateForm(v validation.Validator, c fiber.Ctx, dst any) error {
	if err := c.Bind().Form(dst); err != nil {
		return err
	}
	return validation.Check(v, dst)
}

// ValidateBody, JSON gövde bağlar, sanitize eder ve doğrular.
func ValidateBody(v validation.Validator, c fiber.Ctx, dst any) error {
	if err := c.Bind().Body(dst); err != nil {
		return err
	}
	return validation.Check(v, dst)
}

// ValidatorFrom, *validator.Validate'i ortak arayüze uyumlar.
func ValidatorFrom(v *validator.Validate) validation.Validator { return v }
