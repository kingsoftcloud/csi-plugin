package util

import (
	"fmt"
)

type ErrorResponse struct {
	StatusCode int //Status Code of HTTP Response
	Message    string
}

// An Error represents a custom error for Appengine API failure response
type Error struct {
	AppError ErrorResponse `json:"apperror"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("Appengine API Error: Status Code: %d Message: %s", e.AppError.StatusCode, e.AppError.Message)
}
