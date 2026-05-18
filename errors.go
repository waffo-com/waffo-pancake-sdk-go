package pancake

import "fmt"

// Notice is a single entry of the call-stack-ordered errors / warnings arrays
// returned by the API. The deepest layer is at index 0; outermost at the last
// index. Same shape across REST and GraphQL — GraphQL-specific fields
// (Locations, Path) are populated only by graphql-js resolver errors.
type Notice struct {
	Message string     `json:"message"`
	Layer   ErrorLayer `json:"layer,omitempty"`
	AIHint  string     `json:"aiHint,omitempty"`
	// GraphQL-only fields. omitempty so REST callers don't see them in output.
	Locations []GraphQLErrorLocation `json:"locations,omitempty"`
	Path      []string               `json:"path,omitempty"`
}

// APIError is the legacy name for {@link Notice}. Kept as a type alias for
// backwards compatibility with existing imports.
//
// Deprecated: use Notice.
type APIError = Notice

// Error is thrown when the API returns a non-success response, and is also the
// error type produced by client-side validation failures (status 400 with
// Layer == ErrorLayerSDK). Use errors.As to extract it from a returned error.
//
//	var perr *pancake.Error
//	if errors.As(err, &perr) {
//	    fmt.Println(perr.Status, perr.Errors[0].Message)
//	}
type Error struct {
	Status int
	Errors []APIError
}

// Error returns the deepest error message in the chain, or a generic message
// when the chain is empty.
func (e *Error) Error() string {
	if len(e.Errors) == 0 {
		return fmt.Sprintf("waffo pancake error (status %d)", e.Status)
	}
	return e.Errors[0].Message
}

// newSDKError builds a client-side validation error mirroring the shape of an
// API error so callers can handle both uniformly.
func newSDKError(format string, args ...any) *Error {
	return &Error{
		Status: 400,
		Errors: []APIError{{Message: fmt.Sprintf(format, args...), Layer: ErrorLayerSDK}},
	}
}
