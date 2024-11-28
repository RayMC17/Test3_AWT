package data

import (
	"errors"
)

// custom email error message
var ErrDuplicateEmail = errors.New("duplicate email encountered")

var ErrEditConfilct = errors.New("edit confict")
