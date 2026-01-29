package errs

import (
	"net/http"
	"strings"
)

type AppError struct {
	Code     int
	Messages []string
	Err      error
}

func (e *AppError) Error() string {
	return strings.Join(e.Messages, ", ")
}

func InternalError(err error) *AppError {
	return &AppError{
		Code:     http.StatusInternalServerError,
		Messages: []string{"Unexpected internal server error occurred"},
		Err:      err,
	}
}

func UserError(message string, code int) *AppError {
	return &AppError{
		Code:     code,
		Messages: []string{message},
	}
}

func UserErrors(messages []string, code int) *AppError {
	return &AppError{
		Code:     code,
		Messages: messages,
	}
}

// common errors that are re-used in different places in the codebase
var NoAccess = UserError("You don't have access to this resource", http.StatusForbidden)
var RiceNotFound = UserError("Rice with provided ID not found", http.StatusNotFound)
var MissingFile = UserError("Required file is missing", http.StatusBadRequest)
