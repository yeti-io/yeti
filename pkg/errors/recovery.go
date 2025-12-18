package errors

import (
	"fmt"
	"runtime/debug"
)

// RecoverPanic recovers from a panic and returns it as an error
// It captures the stack trace for debugging
func RecoverPanic(r interface{}) error {
	if r == nil {
		return nil
	}

	var err error
	switch v := r.(type) {
	case error:
		err = v
	case string:
		err = fmt.Errorf("panic: %s", v)
	default:
		err = fmt.Errorf("panic: %v", v)
	}

	// Include stack trace in error details
	stackTrace := string(debug.Stack())
	return ErrInternal.
		WithCause(err).
		WithDetail("panic", true).
		WithDetail("stack_trace", stackTrace).
		AsFatal() // Panics are always fatal
}

// RecoverPanicWithCallback recovers from a panic and calls a callback with the error
func RecoverPanicWithCallback(r interface{}, callback func(error)) error {
	err := RecoverPanic(r)
	if err != nil && callback != nil {
		callback(err)
	}
	return err
}
