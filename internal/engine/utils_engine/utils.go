package utils_engine

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// wrapValidationError turns raw validator output into readable messages.
func WrapValidationError(err error, sourcePath string) error {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		msg := fmt.Sprintf("%s: validation failed:\n", sourcePath)
		for _, e := range verrs {
			msg += fmt.Sprintf("  - %s: %q is not valid (rule: %s: %s)\n",
				e.Namespace(), e.Value(), e.Tag(), e.Param())
		}
		return errors.New(msg)
	}
	return fmt.Errorf("%s: %w", sourcePath, err)
}
