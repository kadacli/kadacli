package utils_engine

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/go-playground/validator/v10"
)

// wrapValidationError turns raw validator output into readable messages.
func WrapValidationError(err error, sourcePath string) error {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		msg := fmt.Sprintf("%s: Validation failed:\n", sourcePath)
		for _, e := range verrs {
			rule := e.Tag()
			if e.Param() != "" {
				rule += ": " + e.Param()
			}
			msg += fmt.Sprintf("  - %s: %s is not valid (rule: %s)\n",
				e.Namespace(), formatFieldValue(e.Value()), rule)
		}
		return errors.New(msg)
	}
	return fmt.Errorf("%s: %w", sourcePath, err)
}

// formatFieldValue renders the offending value for a message. Strings are
// quoted; everything else uses its natural form — %q on an int renders it as
// a rune (500 becomes 'Ǵ'), which is unreadable in a validation error.
func formatFieldValue(v any) string {
	if s, ok := v.(string); ok {
		return strconv.Quote(s)
	}
	return fmt.Sprintf("%v", v)
}
