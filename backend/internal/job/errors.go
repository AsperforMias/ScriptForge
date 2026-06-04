package job

import "errors"

type AppError struct {
	Code    string
	Message string
	Details map[string]any
}

func NewAppError(code, message string, details map[string]any) AppError {
	return AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

func (e AppError) Error() string {
	return e.Message
}

func (e AppError) WithMessage(message string) AppError {
	e.Message = message
	return e
}

func AsAppError(err error, target *AppError) bool {
	return errors.As(err, target)
}

var (
	ErrInvalidInput = NewAppError("invalid_input", "invalid input", nil)
	ErrJobNotFound  = NewAppError("job_not_found", "job not found", nil)
	ErrJobNotReady  = NewAppError("job_not_ready", "job result is not ready", nil)
)
