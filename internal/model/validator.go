package model

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// Validate is a global instance of the validator used to check struct tags across the application.
var Validate *validator.Validate

// init initializes the global validator and registers custom validation rules.
func init() {
	Validate = validator.New()
	_ = Validate.RegisterValidation("mmYYYY", validateMonthYear)
}

// validateMonthYear is a custom validation function that ensures a string field
// follows the "MM-YYYY" (month-year) format.
func validateMonthYear(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	_, err := time.Parse("01-2006", value)
	return err == nil
}
