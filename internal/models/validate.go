package models

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

func newValidator() *validator.Validate {
	validate := validator.New()
	validate.RegisterValidation("regexp", func(fl validator.FieldLevel) bool {
		pattern := fl.Param()
		matched, err := regexp.MatchString(pattern, fl.Field().String())
		return err == nil && matched
	})
	return validate
}
