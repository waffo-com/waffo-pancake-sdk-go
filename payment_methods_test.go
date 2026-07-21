// Payment method allowlist tests for CreateCheckoutSessionParams.PaymentMethods.
//
// Covers: marshal/omitempty contract, client-side shape validation (mirrors the
// TypeScript SDK and server-side checks), and forwarding/ordering through the
// low-level Checkout.CreateSession and Checkout.Anonymous.Create entry points.
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
		PaymentMethods: []string{"APPLEPAY", "CREDITCARD"},
	}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"paymentMethods":["APPLEPAY","CREDITCARD"]`) {
		t.Fatalf("expected ordered paymentMethods in payload, got %s", out)
	}

	// nil slice should omit the field entirely (backward compatible)
	p2 := CreateCheckoutSessionParams{ProductID: "PROD_xxx", Currency: "USD"}
	out2, err := json.Marshal(p2)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out2), "paymentMethods") {
		t.Fatalf("expected omitempty to drop field, got %s", out2)
	}
}

func TestCreateCheckoutSession_RejectsEmptyPaymentMethods(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	if _, err := client.Checkout.CreateSession(context.Background(), CreateCheckoutSessionParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []string{},
	}); err == nil {
		t.Fatal("expected error for empty paymentMethods array")
	}
	if len(server.requests()) != 0 {
		t.Fatal("expected no network call for client-side validation failure")
	}
}

func TestCreateCheckoutSession_RejectsDuplicatePaymentMethods(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	if _, err := client.Checkout.CreateSession(context.Background(), CreateCheckoutSessionParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []string{"CREDITCARD", "CREDITCARD"},
	}); err == nil {
		t.Fatal("expected error for duplicate paymentMethods entries")
	}
	if len(server.requests()) != 0 {
		t.Fatal("expected no network call for client-side validation failure")
	}
}

func TestCreateCheckoutSession_RejectsEmptyStringPaymentMethodEntry(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	if _, err := client.Checkout.CreateSession(context.Background(), CreateCheckoutSessionParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []string{""},
	}); err == nil {
		t.Fatal("expected error for empty-string paymentMethods entry")
	}
	if len(server.requests()) != 0 {
		t.Fatal("expected no network call for client-side validation failure")
	}
}

func TestCheckoutAnonymous_ForwardsPaymentMethodsInOrder(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{
			"sessionId":   "ses_1",
			"checkoutUrl": "https://pancake.example/checkout/abc",
			"expiresAt":   "2026-05-13T00:45:00Z",
		}}
	}

	_, err := client.Checkout.Anonymous.Create(context.Background(), AnonymousCheckoutParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []string{"GOOGLEPAY", "APPLEPAY", "CREDITCARD"},
	})
	if err != nil {
		t.Fatalf("checkout: %v", err)
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
	if !ok {
		t.Fatalf("expected paymentMethods array in request body, got %v", body["paymentMethods"])
	}
	want := []string{"GOOGLEPAY", "APPLEPAY", "CREDITCARD"}
	if len(methods) != len(want) {
		t.Fatalf("expected %d methods, got %d", len(want), len(methods))
	}
	for i, m := range want {
		if methods[i] != m {
			t.Fatalf("expected paymentMethods[%d]=%s, got %v", i, m, methods[i])
		}
	}
}

func TestCheckoutAnonymous_OmitsPaymentMethodsWhenNotProvided(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{
			"sessionId":   "ses_1",
			"checkoutUrl": "https://pancake.example/checkout/abc",
			"expiresAt":   "2026-05-13T00:45:00Z",
		}}
	}

	_, err := client.Checkout.Anonymous.Create(context.Background(), AnonymousCheckoutParams{
		ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:  "USD",
	})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}

	reqs := server.requests()
	var body map[string]any
	if err := json.Unmarshal(reqs[0].Body, &body); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if _, present := body["paymentMethods"]; present {
		t.Fatalf("expected paymentMethods key absent from request body, got %v", body["paymentMethods"])
	}
}
