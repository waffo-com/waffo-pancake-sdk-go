// Checkout session checkoutVersion contract tests.
//
// create-session reports which checkout a session opens ("v2" one-page,
// "v1" legacy). The tests below pin that CreateSession, Anonymous.Create and
// Authenticated.Create all surface it, and that a response without it leaves
// the field empty, which callers treat as "v1".
package pancake

import (
	"context"
	"strings"
	"testing"
)

func checkoutVersionServer(t *testing.T, version string) *Client {
	t.Helper()
	client, _, server := newSignedTestClient(t)
	server.respond = func(req recordedRequest) (int, any) {
		if strings.HasSuffix(req.Path, "/issue-session-token") {
			return 200, map[string]any{"data": map[string]any{"token": "eyJhbGciOi.test.jwt", "expiresAt": "2026-04-02T09:05:00.000Z"}}
		}
		session := map[string]any{
			"sessionId":   testCheckoutSessionID,
			"checkoutUrl": "https://checkout.waffo.ai/store/my-store/checkout/" + testCheckoutSessionID,
			"expiresAt":   "2026-04-02T10:00:00.000Z",
		}
		if version != "" {
			session["checkoutVersion"] = version
		}
		return 200, map[string]any{"data": session}
	}
	return client
}

var checkoutVersionParams = CreateCheckoutSessionParams{ProductID: "PROD_0000000000000000000000", Currency: "USD"}

func TestCheckoutVersion_ReturnedByEveryCreateMethod(t *testing.T) {
	client := checkoutVersionServer(t, "v2")
	ctx := context.Background()

	session, err := client.Checkout.CreateSession(ctx, checkoutVersionParams)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if session.CheckoutVersion != "v2" {
		t.Errorf("CreateSession CheckoutVersion = %q, want v2", session.CheckoutVersion)
	}

	anonymous, err := client.Checkout.Anonymous.Create(ctx, checkoutVersionParams)
	if err != nil {
		t.Fatalf("anonymous create: %v", err)
	}
	if anonymous.CheckoutVersion != "v2" {
		t.Errorf("Anonymous.Create CheckoutVersion = %q, want v2", anonymous.CheckoutVersion)
	}

	authenticated, err := client.Checkout.Authenticated.Create(ctx, AuthenticatedCheckoutParams{
		CreateCheckoutSessionParams: checkoutVersionParams,
		BuyerIdentity:               "user-123",
	})
	if err != nil {
		t.Fatalf("authenticated create: %v", err)
	}
	if authenticated.CheckoutVersion != "v2" {
		t.Errorf("Authenticated.Create CheckoutVersion = %q, want v2", authenticated.CheckoutVersion)
	}
	if !strings.HasSuffix(authenticated.CheckoutURL, "#token=eyJhbGciOi.test.jwt") {
		t.Errorf("Authenticated.Create CheckoutURL = %q, want the token fragment appended", authenticated.CheckoutURL)
	}
}

func TestCheckoutVersion_EmptyWhenPlatformOmitsIt(t *testing.T) {
	client := checkoutVersionServer(t, "")
	ctx := context.Background()

	session, err := client.Checkout.CreateSession(ctx, checkoutVersionParams)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if session.CheckoutVersion != "" {
		t.Errorf("CheckoutVersion = %q, want empty (treated as v1)", session.CheckoutVersion)
	}

	authenticated, err := client.Checkout.Authenticated.Create(ctx, AuthenticatedCheckoutParams{
		CreateCheckoutSessionParams: checkoutVersionParams,
		BuyerIdentity:               "user-123",
	})
	if err != nil {
		t.Fatalf("authenticated create: %v", err)
	}
	if authenticated.CheckoutVersion != "" {
		t.Errorf("Authenticated.Create CheckoutVersion = %q, want empty (treated as v1)", authenticated.CheckoutVersion)
	}
}
