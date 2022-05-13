package validate

import (
	"github.com/go-playground/validator/v10"

	"github.com/sergeii/practikum-go-gophermart/pkg/validation"
)

func LuhnNumber(fl validator.FieldLevel) bool {
	maybeLuhn, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}
	return validation.CheckLuhnNumber(maybeLuhn)
}
