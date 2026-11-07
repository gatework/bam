package bam

import (
	"fmt"
	"net/http"
)

// APIError is an HTTP failure. StatusCode is the actual HTTP status, rather than
// a possibly conflicting status in the response body. Use errors.As to inspect
// Code and StatusCode. Body and credentials are never included in Error().
type APIError struct {
	StatusCode int
	Code       string
	Reason     string
	Message    string
	Method     string
	Path       string
	Err        error // Optional response-body read failure; the HTTP status remains authoritative.
}

func (e *APIError) Error() string {
	// A server message can echo request secrets. It is available explicitly in
	// Message for diagnostics, but deliberately excluded from routine logging.
	return fmt.Sprintf("bam: %s %s: HTTP %d %s", e.Method, e.Path, e.StatusCode, http.StatusText(e.StatusCode))
}

func (e *APIError) Unwrap() error { return e.Err }

// ProtocolError means that a response does not satisfy the JSON contract.
// Mutating requests may already have taken effect; do not retry blindly.
type ProtocolError struct {
	Method string
	Path   string
	Detail string
	Err    error
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("bam: %s %s: %s", e.Method, e.Path, e.Detail)
}

func (e *ProtocolError) Unwrap() error { return e.Err }

// AcceptedError reports HTTP 202 without claiming that a requested mutation has
// completed. Resource contains the complete response, typically a WorkflowRequest.
// Do not retry the mutation; inspect the pending resource instead.
type AcceptedError struct {
	Resource Document
	Location string
}

func (e *AcceptedError) Error() string {
	return "bam: request accepted but not completed (HTTP 202)"
}
