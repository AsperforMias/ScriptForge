package llm

import "fmt"

type Error struct {
	Kind    string
	Message string
}

func (e Error) Error() string {
	return e.Message
}

func NewUnavailableError(message string) error {
	return Error{
		Kind:    "provider_unavailable",
		Message: message,
	}
}

func NewInvocationError(provider string, err error) error {
	return Error{
		Kind:    "provider_invocation_failed",
		Message: fmt.Sprintf("%s invocation failed: %v", provider, err),
	}
}
