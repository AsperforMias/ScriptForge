package llm

import (
	"errors"
	"fmt"
)

type Error struct {
	Kind    string
	Message string
	Debug   *DebugSnapshot
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
	return NewInvocationErrorWithDebug(provider, err, nil)
}

func NewInvocationErrorWithDebug(provider string, err error, debug *DebugSnapshot) error {
	return Error{
		Kind:    "provider_invocation_failed",
		Message: fmt.Sprintf("%s invocation failed: %v", provider, err),
		Debug:   debug,
	}
}

func DebugFromError(err error) *DebugSnapshot {
	var providerErr Error
	if errors.As(err, &providerErr) {
		return providerErr.Debug
	}
	return nil
}
