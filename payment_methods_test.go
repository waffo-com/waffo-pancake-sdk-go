// Tests for the optional ordered payment-methods allow-list on
// CreateCheckoutSessionParams (WAF-508): marshaling, forwarding to
// create-session in the requested order, and client-side format validation.
package pancake

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCreateCheckoutSessionParams_PaymentMethods_Marshal(t *testing.T) {
	p := CreateCheckoutSessionParams{
		ProductID:      "PROD_xxx",
		Currency:       "USD",
		PaymentMethods: []string{"EWALLET", "CREDITCARD"},
	}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"paymentMethods":["EWALLET","CREDITCARD"]`) {
		t.Fatalf("expected paymentMethods in payload preserving order, got %s", out)
	}

	// nil slice should omit the field entirely (backward-compatible default behavior)
	p2 := CreateCheckoutSessionParams{ProductID: "PROD_xxx", Currency: "USD"}
	out2, _ := json.Marshal(p2)
	if strings.Contains(string(out2), "paymentMethods") {
		t.Fatalf("expected omitempty to drop field, got %s", out2)
	}
}

func TestCheckoutAnonymous_ForwardsPaymentMethodsInOrder(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"sessionId": "ses", "checkoutUrl": "https://x/y", "expiresAt": "z"}}
	}

	if _, err := client.Checkout.Anonymous.Create(context.Background(), AnonymousCheckoutParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []string{"APPLEPAY", "GOOGLEPAY", "CREDITCARD"},
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	reqs := server.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	var body map[string]any
	if err := json.Unmarshal(reqs[0].Body, &body); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	methods, ok := body["paymentMethods"].([]any)
	if !ok || len(methods) != 3 {
		t.Fatalf("expected 3 paymentMethods in request body, got %v", body["paymentMethods"])
	}
	if methods[0] != "APPLEPAY" || methods[1] != "GOOGLEPAY" || methods[2] != "CREDITCARD" {
		t.Fatalf("expected paymentMethods order preserved, got %v", methods)
	}
}

func TestCheckoutAuthenticated_ForwardsPaymentMethodsToCreateSessionOnly(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(req recordedRequest) (int, any) {
		if strings.Contains(req.Path, "issue-session-token") {
			return 200, map[string]any{"data": map[string]any{"token": "jwt", "expiresAt": "z"}}
		}
		return 200, map[string]any{"data": map[string]any{"sessionId": "ses", "checkoutUrl": "https://x/y", "expiresAt": "z"}}
	}

	if _, err := client.Checkout.Authenticated.Create(context.Background(), AuthenticatedCheckoutParams{
		CreateCheckoutSessionParams: CreateCheckoutSessionParams{
			ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
			Currency:       "USD",
			PaymentMethods: []string{"CREDITCARD", "DEBITCARD"},
		},
		BuyerIdentity: "user-123",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	for _, req := range server.requests() {
		var body map[string]any
		if err := json.Unmarshal(req.Body, &body); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if strings.Contains(req.Path, "issue-session-token") {
			if _, present := body["paymentMethods"]; present {
				t.Fatalf("paymentMethods must not leak into issue-session-token, got %v", body)
			}
			continue
		}
		methods, ok := body["paymentMethods"].([]any)
		if !ok || len(methods) != 2 || methods[0] != "CREDITCARD" || methods[1] != "DEBITCARD" {
			t.Fatalf("expected paymentMethods=[CREDITCARD,DEBITCARD] on create-session, got %v", body["paymentMethods"])
		}
	}
}

func TestCreateCheckoutSession_RejectsEmptyPaymentMethods(t *testing.T) {
	client, _, _ := newSignedTestClient(t)
	if _, err := client.Checkout.CreateSession(context.Background(), CreateCheckoutSessionParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []string{},
	}); err == nil {
		t.Fatal("expected error for empty paymentMethods slice")
	}
}

func TestCreateCheckoutSession_RejectsDuplicatePaymentMethods(t *testing.T) {
	client, _, _ := newSignedTestClient(t)
	if _, err := client.Checkout.CreateSession(context.Background(), CreateCheckoutSessionParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []string{"CREDITCARD", "CREDITCARD"},
	}); err == nil {
		t.Fatal("expected error for duplicate paymentMethods entries")
	}
}

func TestCreateCheckoutSession_RejectsUnknownPaymentMethod(t *testing.T) {
	client, _, _ := newSignedTestClient(t)
	if _, err := client.Checkout.CreateSession(context.Background(), CreateCheckoutSessionParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []string{"ALIPAY"},
	}); err == nil {
		t.Fatal("expected error for unknown paymentMethods entry")
	}
}

func TestCreateCheckoutSession_AllowsNilPaymentMethods(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"sessionId": "ses", "checkoutUrl": "https://x/y", "expiresAt": "z"}}
	}
	if _, err := client.Checkout.CreateSession(context.Background(), CreateCheckoutSessionParams{
		ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:  "USD",
	}); err != nil {
		t.Fatalf("expected omitted paymentMethods to keep default behavior, got error: %v", err)
	}
}
