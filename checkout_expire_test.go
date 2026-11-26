// Checkout session expire contract tests.
//
// ExpireSession closes a one-page checkout session. The tests below pin what
// callers depend on: the session ID reaches the expire endpoint unchanged, the
// result is unwrapped from the envelope, an empty ID is rejected before sending,
// and the platform's reason code and aiHint reach the caller on *Error.
package pancake

import (
	"context"
	"errors"
	"testing"
)

const (
	testCheckoutSessionID = "cs_3f2b9c1e-7a4d-4e8b-9c21-5d6f0a1b2c3d"
	expireSessionPath     = "/v1/actions/checkout/session/expire"
)

func TestCheckoutExpireSession_PostsSessionIDAndUnwraps(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{
			"checkoutSessionId": testCheckoutSessionID,
			"status":            "expired",
		}}
	}

	res, err := client.Checkout.ExpireSession(context.Background(), ExpireCheckoutSessionParams{
		CheckoutSessionID: testCheckoutSessionID,
	})
	if err != nil {
		t.Fatalf("expire session: %v", err)
	}
	if res.CheckoutSessionID != testCheckoutSessionID || res.Status != "expired" {
		t.Errorf("result = %+v, want %s / expired", res, testCheckoutSessionID)
	}

	reqs := server.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected one request, got %d", len(reqs))
	}
	if reqs[0].Method != "POST" || reqs[0].Path != expireSessionPath {
		t.Errorf("request = %s %s, want POST %s", reqs[0].Method, reqs[0].Path, expireSessionPath)
	}
	body := requestBody(t, server, "/session/expire")
	if len(body) != 1 || body["checkoutSessionId"] != testCheckoutSessionID {
		t.Errorf("body = %v, want only checkoutSessionId", body)
	}
}

func TestCheckoutExpireSession_SendsIdempotencyKeyWhenGiven(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"checkoutSessionId": testCheckoutSessionID, "status": "expired"}}
	}

	if _, err := client.Checkout.ExpireSession(context.Background(), ExpireCheckoutSessionParams{
		CheckoutSessionID: testCheckoutSessionID,
	}, WithIdempotencyKey("expire-1")); err != nil {
		t.Fatalf("expire session: %v", err)
	}
	if got := server.requests()[0].Headers.Get("X-Idempotency-Key"); got != "expire-1" {
		t.Errorf("X-Idempotency-Key = %q, want expire-1", got)
	}
}

func TestCheckoutExpireSession_ValidatesInput(t *testing.T) {
	client, _, server := newSignedTestClient(t)

	for _, id := range []string{"", "   "} {
		_, err := client.Checkout.ExpireSession(context.Background(), ExpireCheckoutSessionParams{CheckoutSessionID: id})
		var perr *Error
		if !errors.As(err, &perr) {
			t.Fatalf("CheckoutSessionID %q: expected *Error, got %v", id, err)
		}
		if perr.Status != 400 || perr.Errors[0].Layer != ErrorLayerSDK {
			t.Errorf("CheckoutSessionID %q: got status %d layer %q, want 400 sdk", id, perr.Status, perr.Errors[0].Layer)
		}
	}
	if len(server.requests()) != 0 {
		t.Fatalf("expected no request to be sent, got %d", len(server.requests()))
	}
}

func TestCheckoutExpireSession_PropagatesReasonCode(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 409, map[string]any{
			"data":   nil,
			"errors": []map[string]string{{"message": "PAYMENT_IN_FLIGHT", "layer": "order", "aiHint": "Wait for checkout.completed, or retry later."}},
		}
	}

	_, err := client.Checkout.ExpireSession(context.Background(), ExpireCheckoutSessionParams{
		CheckoutSessionID: testCheckoutSessionID,
	})
	var perr *Error
	if !errors.As(err, &perr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if perr.Status != 409 || perr.Errors[0].Message != "PAYMENT_IN_FLIGHT" {
		t.Errorf("got status %d message %q, want 409 PAYMENT_IN_FLIGHT", perr.Status, perr.Errors[0].Message)
	}
	if perr.Errors[0].AIHint == "" {
		t.Error("expected aiHint to be decoded")
	}
}
