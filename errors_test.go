package pancake

import (
	"errors"
	"testing"
)

func TestError_ErrorReturnsDeepestMessage(t *testing.T) {
	e := &Error{
		Status: 400,
		Errors: []APIError{
			{Message: "deepest layer", Layer: ErrorLayerStore},
			{Message: "outermost layer", Layer: ErrorLayerGateway},
		},
	}
	if got := e.Error(); got != "deepest layer" {
		t.Fatalf("Error() = %q, want %q", got, "deepest layer")
	}
}

func TestError_ErrorOnEmptyChain(t *testing.T) {
	e := &Error{Status: 500}
	if got := e.Error(); got == "" {
		t.Fatal("expected non-empty fallback message")
	}
}

func TestError_ErrorsAsExtraction(t *testing.T) {
	var err error = &Error{Status: 401, Errors: []APIError{{Message: "no", Layer: ErrorLayerGateway}}}
	var perr *Error
	if !errors.As(err, &perr) {
		t.Fatal("errors.As failed to extract *Error")
	}
	if perr.Status != 401 {
		t.Fatalf("Status = %d, want 401", perr.Status)
	}
}

func TestNewSDKError(t *testing.T) {
	err := newSDKError("bad %s", "input")
	if err.Status != 400 {
		t.Fatalf("Status = %d, want 400", err.Status)
	}
	if err.Errors[0].Layer != ErrorLayerSDK {
		t.Fatalf("Layer = %q, want sdk", err.Errors[0].Layer)
	}
	if err.Errors[0].Message != "bad input" {
		t.Fatalf("Message = %q, want %q", err.Errors[0].Message, "bad input")
	}
}
