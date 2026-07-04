package apperror

import "fmt"

type AppError struct {
	Message    string
	StatusCode int
}

func (e *AppError) Error() string { return e.Message }

func New(message string, statusCode int) *AppError {
	if statusCode == 0 {
		statusCode = 400
	}
	return &AppError{Message: message, StatusCode: statusCode}
}

func BadRequest(message string) *AppError { return New(message, 400) }
func NotFound(message string) *AppError   { return New(message, 404) }

func AsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	if ae, ok := err.(*AppError); ok {
		return ae, true
	}
	return nil, false
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
