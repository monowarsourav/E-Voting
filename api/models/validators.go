package models

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// alphanumDashRegexp matches strings consisting solely of ASCII letters,
// digits, underscore, and hyphen. Used for IDs that flow into logs, URLs,
// and file paths.
var alphanumDashRegexp = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// alphanumDash is a custom validator registered under the tag "alphanumdash".
func alphanumDash(fl validator.FieldLevel) bool {
	return alphanumDashRegexp.MatchString(fl.Field().String())
}

// RegisterValidators registers application-specific validation tags with
// gin's binding validator. Safe to call once at startup.
func RegisterValidators() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return nil
	}
	return v.RegisterValidation("alphanumdash", alphanumDash)
}

// init auto-registers custom validators on import so that tests and other
// callers that skip main() still get a working validator.
func init() {
	_ = RegisterValidators()
}

// voterIDRule is kept private in requests.go; this var suppresses unused
// warnings if callers only use the struct tags directly.
var _ = voterIDRule
