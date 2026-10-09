package pancake

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const testPortalStoreID = "STO_AbCdEfGhIjKlMnOpQrStUv"

func portalResponder(data map[string]any) func(recordedRequest) (int, any) {
	return func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": data}
	}
}

func TestAuthCreateCustomerPortalLink_AppendsTokenFragment(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	portalURL := "https://pancake.waffo.ai/consumer/portal/store/" + testPortalStoreID
	server.respond = portalResponder(map[string]any{
		"token":     "PORTAL_JWT",
		"expiresAt": "2026-05-13T01:15:00Z",
		"portalUrl": portalURL,
	})

	link, err := client.Auth.CreateCustomerPortalLink(context.Background(), CreateCustomerPortalLinkParams{
		StoreID:       testPortalStoreID,
		BuyerIdentity: "merchant-customer-123",
	})
	if err != nil {
		t.Fatalf("create portal link: %v", err)
	}
	if link.PortalURL != portalURL+"#token=PORTAL_JWT" {
		t.Errorf("PortalURL = %q", link.PortalURL)
	}
	if link.ExpiresAt != "2026-05-13T01:15:00Z" {
		t.Errorf("ExpiresAt = %q", link.ExpiresAt)
	}

	body := requestBody(t, server, "/v1/actions/auth/issue-session-token")
	want := map[string]any{"storeId": testPortalStoreID, "buyerIdentity": "merchant-customer-123", "purpose": "portal"}
	if len(body) != len(want) {
		t.Fatalf("unexpected body fields: %v", body)
	}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("body[%s] = %v want %v", k, body[k], v)
		}
	}
}

func TestAuthCreateCustomerPortalLink_ForwardsIdempotencyKeyAndWarnings(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{
			"data":     map[string]any{"token": "T", "expiresAt": "z", "portalUrl": "https://p.example/consumer/portal/store/" + testPortalStoreID},
			"warnings": []map[string]string{{"message": "heads up", "layer": "user"}},
		}
	}

	link, err := client.Auth.CreateCustomerPortalLink(context.Background(), CreateCustomerPortalLinkParams{
		StoreID:       testPortalStoreID,
		BuyerIdentity: "u-1",
	}, WithIdempotencyKey("portal-link-1"))
	if err != nil {
		t.Fatalf("create portal link: %v", err)
	}
	if len(link.Warnings) != 1 || link.Warnings[0].Message != "heads up" {
		t.Errorf("Warnings = %+v", link.Warnings)
	}
	if got := server.requests()[0].Headers.Get("X-Idempotency-Key"); got != "portal-link-1" {
		t.Errorf("X-Idempotency-Key = %q", got)
	}
}

func TestAuthCreateCustomerPortalLink_MissingPortalURL(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = portalResponder(map[string]any{"token": "T", "expiresAt": "z"})

	_, err := client.Auth.CreateCustomerPortalLink(context.Background(), CreateCustomerPortalLinkParams{
		StoreID:       testPortalStoreID,
		BuyerIdentity: "u-1",
	})
	var perr *Error
	if !errors.As(err, &perr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if perr.Errors[0].Layer != ErrorLayerSDK {
		t.Errorf("Layer = %q want sdk", perr.Errors[0].Layer)
	}
}

func TestAuthCreateCustomerPortalLink_MissingToken(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = portalResponder(map[string]any{
		"token":     "",
		"expiresAt": "z",
		"portalUrl": "https://p.example/consumer/portal/store/" + testPortalStoreID,
	})

	link, err := client.Auth.CreateCustomerPortalLink(context.Background(), CreateCustomerPortalLinkParams{
		StoreID:       testPortalStoreID,
		BuyerIdentity: "u-1",
	})
	if link != nil {
		t.Errorf("expected no link, got %+v", link)
	}
	var perr *Error
	if !errors.As(err, &perr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if perr.Status != 200 || perr.Errors[0].Layer != ErrorLayerSDK || !strings.Contains(perr.Errors[0].Message, "token") {
		t.Errorf("unexpected error: %+v", perr)
	}
}

func TestAuthCreateCustomerPortalLink_APIError(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 400, map[string]any{"data": nil, "errors": []map[string]string{{"message": "productId is not allowed", "layer": "user"}}}
	}

	_, err := client.Auth.CreateCustomerPortalLink(context.Background(), CreateCustomerPortalLinkParams{
		StoreID:       testPortalStoreID,
		BuyerIdentity: "u-1",
	})
	var perr *Error
	if !errors.As(err, &perr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if perr.Status != 400 || perr.Errors[0].Layer != ErrorLayerUser {
		t.Errorf("unexpected error: %+v", perr)
	}
}

func TestAuthCreateCustomerPortalLink_ValidatesBeforeSending(t *testing.T) {
	client, _, server := newSignedTestClient(t)

	cases := map[string]CreateCustomerPortalLinkParams{
		"missing storeId":     {BuyerIdentity: "u-1"},
		"non-store short ID":  {StoreID: "PROD_AbCdEfGhIjKlMnOpQrStUv", BuyerIdentity: "u-1"},
		"malformed storeId":   {StoreID: "bad", BuyerIdentity: "u-1"},
		"empty buyerIdentity": {StoreID: testPortalStoreID},
		"blank buyerIdentity": {StoreID: testPortalStoreID, BuyerIdentity: "  "},
	}
	for name, p := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := client.Auth.CreateCustomerPortalLink(context.Background(), p)
			var perr *Error
			if !errors.As(err, &perr) || perr.Status != 400 || perr.Errors[0].Layer != ErrorLayerSDK {
				t.Fatalf("expected SDK 400, got %v", err)
			}
		})
	}
	if n := len(server.requests()); n != 0 {
		t.Errorf("expected no requests, got %d", n)
	}
}
