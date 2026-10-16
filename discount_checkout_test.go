// Checkout discount parameter tests.
//
// CreateCheckoutSessionParams.CouponID applies a locked coupon and
// AllowPromotionCodes shows a code entry field. The two are mutually exclusive;
// the SDK rejects the combination before sending, as the platform would with a 400.
package pancake

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const testCouponID = "CPN_AbCdEfGhIjKlMnOpQrStUv"

func TestCreateCheckoutSessionParams_DiscountFields_Marshal(t *testing.T) {
	out, err := json.Marshal(CreateCheckoutSessionParams{ProductID: "PROD_xxx", Currency: "USD", CouponID: Ptr(testCouponID)})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"couponId":"`+testCouponID+`"`) {
		t.Fatalf("expected couponId in payload, got %s", out)
	}

	out, err = json.Marshal(CreateCheckoutSessionParams{ProductID: "PROD_xxx", Currency: "USD", AllowPromotionCodes: Ptr(false)})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"allowPromotionCodes":false`) {
		t.Fatalf("expected explicit allowPromotionCodes=false in payload, got %s", out)
	}

	out, _ = json.Marshal(CreateCheckoutSessionParams{ProductID: "PROD_xxx", Currency: "USD"})
	if strings.Contains(string(out), "couponId") || strings.Contains(string(out), "allowPromotionCodes") {
		t.Fatalf("expected omitempty to drop both discount fields, got %s", out)
	}
}

func TestCheckout_DiscountFields_Forwarded(t *testing.T) {
	ctx := context.Background()
	client, _, srv := newSignedTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"sessionId": "cs", "checkoutUrl": "https://x", "expiresAt": "z", "token": "tok"}}
	}

	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:  "USD",
		CouponID:  Ptr(testCouponID),
	}); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if _, err := client.Checkout.Anonymous.Create(ctx, AnonymousCheckoutParams{
		ProductID:           "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:            "USD",
		AllowPromotionCodes: Ptr(true),
	}); err != nil {
		t.Fatalf("Anonymous.Create: %v", err)
	}

	reqs := srv.requests()
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(reqs))
	}
	if !strings.Contains(string(reqs[0].Body), `"couponId":"`+testCouponID+`"`) {
		t.Errorf("expected couponId in create-session body, got %s", reqs[0].Body)
	}
	if !strings.Contains(string(reqs[1].Body), `"allowPromotionCodes":true`) {
		t.Errorf("expected allowPromotionCodes in create-session body, got %s", reqs[1].Body)
	}
}

func TestCheckout_DiscountFields_Rejected(t *testing.T) {
	ctx := context.Background()
	client, _, srv := newSignedTestClient(t)
	base := CreateCheckoutSessionParams{ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv", Currency: "USD"}

	wrongPrefix := base
	wrongPrefix.CouponID = Ptr("PMC_AbCdEfGhIjKlMnOpQrStUv")
	both := base
	both.CouponID = Ptr(testCouponID)
	both.AllowPromotionCodes = Ptr(false)

	cases := []struct {
		name string
		call func() error
		want string
	}{
		{"CreateSession wrong prefix", func() error {
			_, err := client.Checkout.CreateSession(ctx, wrongPrefix)
			return err
		}, "CPN_ prefix"},
		{"CreateSession both fields", func() error {
			_, err := client.Checkout.CreateSession(ctx, both)
			return err
		}, "mutually exclusive"},
		{"Anonymous.Create both fields", func() error {
			_, err := client.Checkout.Anonymous.Create(ctx, both)
			return err
		}, "mutually exclusive"},
		{"Authenticated.Create both fields", func() error {
			_, err := client.Checkout.Authenticated.Create(ctx, AuthenticatedCheckoutParams{CreateCheckoutSessionParams: both, BuyerIdentity: "user-1"})
			return err
		}, "mutually exclusive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			var perr *Error
			if !errors.As(err, &perr) {
				t.Fatalf("expected *Error, got %v", err)
			}
			if perr.Status != 400 || perr.Errors[0].Layer != ErrorLayerSDK {
				t.Fatalf("expected 400 sdk error, got %d %s", perr.Status, perr.Errors[0].Layer)
			}
			if !strings.Contains(perr.Errors[0].Message, tc.want) {
				t.Fatalf("expected message containing %q, got %q", tc.want, perr.Errors[0].Message)
			}
		})
	}
	if n := len(srv.requests()); n != 0 {
		t.Fatalf("expected no requests, got %d", n)
	}
}
