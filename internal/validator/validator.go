package validator

import (
	"regexp"
)

// Validator struct holds validation errors.
type Validator struct {
	Errors map[string]string
}

// New creates a new Validator instance.
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// Valid checks if the validator contains any errors.
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError adds an error message for a given field if it doesn't already exist.
func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check adds an error message if a condition is not met.
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// In checks if a value exists in a list of permitted values.
func In(value string, list ...string) bool {
	for _, item := range list {
		if value == item {
			return true
		}
	}
	return false
}

// Matches checks if a value matches a given regular expression pattern.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// MinLength checks if a value has a minimum length.
func MinLength(value string, min int) bool {
	return len(value) >= min
}

// MaxLength checks if a value does not exceed a maximum length.
func MaxLength(value string, max int) bool {
	return len(value) <= max
}

// Unique checks if all values in a slice are unique.
func Unique(values []string) bool {
	seen := make(map[string]bool)
	for _, value := range values {
		if seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

// regexp to check if an email is valid
var EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
