package model

import (
	"time"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New()
	_ = Validate.RegisterValidation("mmYYYY", validateMonthYear)
}

func validateMonthYear(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	_, err := time.Parse("01-2006", value)
	return err == nil
}
