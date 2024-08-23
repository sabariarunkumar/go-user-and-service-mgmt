package misc

import "github.com/go-playground/validator/v10"

// PayloadValidator validates fields
var PayloadValidator *validator.Validate

// Payload validator specifically used for email validation.
func InitPayloadValidator() {
	PayloadValidator = validator.New()
}
