// Package errors defines the nx-level error taxonomy and the translation
// layer between low-level failures and user-facing messages.
//
// Every error surfaced to the CLI carries a failure Class used for stderr
// diagnostics and maps to exit code 10 per the output-schema contract.
package errors

import (
	"fmt"
)

// Class identifies the failure category reported on stderr.
type Class string

// Failure classes reported on stderr.
const (
	ClassAuth       Class = "auth"       // rejected or missing credentials
	ClassNetwork    Class = "network"    // connection failures, timeouts
	ClassTLS        Class = "tls"        // certificate verification failures
	ClassResponse   Class = "response"   // malformed or unexpected Jenkins/Nexus responses
	ClassCoordinate Class = "coordinate" // unparseable format-specific coordinate
	ClassInstance   Class = "instance"   // unknown alias or unresolvable instance
	ClassFlag       Class = "flag"       // invalid flag or argument values
)

// ExitCode is the process exit code for every nx-level failure.
const ExitCode = 10

// Error is the canonical nx error type.
type Error struct {
	class Class
	msg   string
	cause error
}

// New builds an Error of the given class with a formatted message.
func New(class Class, format string, args ...any) *Error {
	return &Error{class: class, msg: fmt.Sprintf(format, args...)}
}

// Wrap builds an Error of the given class wrapping cause with context.
func Wrap(class Class, cause error, format string, args ...any) *Error {
	if cause == nil {
		return nil
	}
	return &Error{class: class, msg: fmt.Sprintf(format, args...), cause: cause}
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.class, e.msg, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.class, e.msg)
}

// Unwrap exposes the wrapped cause for errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.cause }

// Class reports the failure category.
func (e *Error) Class() Class { return e.class }

// ExitCode implements the exit-code contract; every nx-level failure exits 10.
func (e *Error) ExitCode() int { return ExitCode }
