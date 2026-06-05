package kori

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validatorOnce sync.Once
	vld           *validator.Validate
)

func getValidator() *validator.Validate {
	validatorOnce.Do(func() {
		vld = validator.New(validator.WithRequiredStructEnabled())

		vld.RegisterTagNameFunc(func(f reflect.StructField) string {
			for _, tag := range []string{"json", "query", "path", "header", "form"} {
				if v := f.Tag.Get(tag); v != "" {
					return strings.Split(v, ",")[0]
				}
			}

			return f.Name
		})
	})

	return vld
}

func validateStruct(v any) error {
	if err := getValidator().Struct(v); err != nil {
		if ve, ok := errors.AsType[validator.ValidationErrors](err); ok {
			details := make([]ValidationDetail, len(ve))

			for i, fe := range ve {
				details[i] = ValidationDetail{
					Field:   fe.Field(),
					Tag:     fe.Tag(),
					Param:   fe.Param(),
					Message: validationMessage(fe),
				}
			}

			return &HTTPError{
				Status:  http.StatusUnprocessableEntity,
				Message: "validation failed",
				Details: details,
			}
		}
		return err
	}
	return nil
}

// ValidationDetail describes a single field validation error.
// Returned as Details in the 422 HTTPError response.
type ValidationDetail struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Param   string `json:"param,omitempty"`
	Message string `json:"message"`
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "min":
		return fe.Field() + " must be at least " + fe.Param()
	case "max":
		return fe.Field() + " must be at most " + fe.Param()
	case "email":
		return fe.Field() + " must be a valid email address"
	case "uuid", "uuid4":
		return fe.Field() + " must be a valid UUID"
	case "oneof":
		return fe.Field() + " must be one of: " + fe.Param()
	case "url":
		return fe.Field() + " must be a valid URL"
	case "gt":
		return fe.Field() + " must be greater than " + fe.Param()
	case "gte":
		return fe.Field() + " must be greater than or equal to " + fe.Param()
	case "lt":
		return fe.Field() + " must be less than " + fe.Param()
	case "lte":
		return fe.Field() + " must be less than or equal to " + fe.Param()
	case "len":
		return fe.Field() + " must have exactly " + fe.Param() + " characters"
	default:
		return fe.Error()
	}
}
