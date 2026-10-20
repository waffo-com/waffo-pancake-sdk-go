// Checkout discount parameter tests.
//
// CouponID applies a locked coupon, PromotionCode prefills a promotion code and
// AllowPromotionCodes shows a code entry field, on new purchases and plan changes.
// CouponID is mutually exclusive with the other two; the SDK rejects those
// combinations before sending, as the platform would with a 400.
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

func TestCheckout_PromotionCode_Forwarded(t *testing.T) {
	ctx := context.Background()
	client, _, srv := newSignedTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"sessionId": "cs", "checkoutUrl": "https://x", "expiresAt": "z", "token": "tok"}}
	}
	base := CreateCheckoutSessionParams{ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv", Currency: "USD", PromotionCode: Ptr("SPRING-20")}
	unlocked := base
	unlocked.AllowPromotionCodes = Ptr(true)

	if _, err := client.Checkout.CreateSession(ctx, base); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if _, err := client.Checkout.Anonymous.Create(ctx, unlocked); err != nil {
		t.Fatalf("Anonymous.Create: %v", err)
	}
	if _, err := client.Checkout.Authenticated.Create(ctx, AuthenticatedCheckoutParams{CreateCheckoutSessionParams: unlocked, BuyerIdentity: "user-1"}); err != nil {
		t.Fatalf("Authenticated.Create: %v", err)
	}

	var sessionBodies []string
	for _, r := range srv.requests() {
		body := string(r.Body)
		if strings.Contains(r.Path, "issue-session-token") {
			if strings.Contains(body, "promotionCode") {
				t.Errorf("expected no promotionCode in issue-session-token body, got %s", body)
			}
			continue
		}
		sessionBodies = append(sessionBodies, body)
	}
	if len(sessionBodies) != 3 {
		t.Fatalf("expected 3 create-session requests, got %d", len(sessionBodies))
	}
	for i, body := range sessionBodies {
		if !strings.Contains(body, `"promotionCode":"SPRING-20"`) {
			t.Errorf("request %d: expected promotionCode in body, got %s", i, body)
		}
	}
	if strings.Contains(sessionBodies[0], "allowPromotionCodes") {
		t.Errorf("expected no allowPromotionCodes on the locked session, got %s", sessionBodies[0])
	}
	if !strings.Contains(sessionBodies[1], `"allowPromotionCodes":true`) {
		t.Errorf("expected allowPromotionCodes on the unlocked session, got %s", sessionBodies[1])
	}
}

func TestCheckout_PromotionCode_Rejected(t *testing.T) {
	ctx := context.Background()
	client, _, srv := newSignedTestClient(t)
	base := CreateCheckoutSessionParams{ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv", Currency: "USD"}

	withCoupon := base
	withCoupon.CouponID = Ptr(testCouponID)
	withCoupon.PromotionCode = Ptr("SPRING-20")
	blank := base
	blank.PromotionCode = Ptr("  ")

	cases := []struct {
		name string
		call func() error
		want string
	}{
		{"CreateSession couponId and promotionCode", func() error {
			_, err := client.Checkout.CreateSession(ctx, withCoupon)
			return err
		}, "couponId and promotionCode are mutually exclusive"},
		{"Anonymous.Create couponId and promotionCode", func() error {
			_, err := client.Checkout.Anonymous.Create(ctx, withCoupon)
			return err
		}, "mutually exclusive"},
		{"Authenticated.Create couponId and promotionCode", func() error {
			_, err := client.Checkout.Authenticated.Create(ctx, AuthenticatedCheckoutParams{CreateCheckoutSessionParams: withCoupon, BuyerIdentity: "user-1"})
			return err
		}, "mutually exclusive"},
		{"CreateSession blank promotionCode", func() error {
			_, err := client.Checkout.CreateSession(ctx, blank)
			return err
		}, "promotionCode cannot be empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertSDKError(t, tc.call(), tc.want)
		})
	}
	if n := len(srv.requests()); n != 0 {
		t.Fatalf("expected no requests, got %d", n)
	}
}

func TestPlanChange_DiscountFields_Forwarded(t *testing.T) {
	ctx := context.Background()
	client, _, srv := newSignedTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"sessionId": "cs", "checkoutUrl": "https://x/store/s/change/cs", "expiresAt": "z", "token": "tok"}}
	}
	base := CreatePlanChangeSessionParams{OriginOrderID: testOriginOrderID, ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv", Currency: "USD"}
	withCoupon := base
	withCoupon.CouponID = Ptr(testCouponID)
	withCode := base
	withCode.PromotionCode = Ptr("SPRING-20")
	withCode.AllowPromotionCodes = Ptr(true)

	if _, err := client.Checkout.CreatePlanChangeSession(ctx, withCoupon); err != nil {
		t.Fatalf("CreatePlanChangeSession: %v", err)
	}
	if _, err := client.Checkout.Authenticated.CreatePlanChange(ctx, AuthenticatedPlanChangeParams{CreatePlanChangeSessionParams: withCode, BuyerIdentity: "user-1"}); err != nil {
		t.Fatalf("Authenticated.CreatePlanChange: %v", err)
	}

	var sessionBodies []string
	for _, r := range srv.requests() {
		if strings.Contains(r.Path, "create-session") {
			sessionBodies = append(sessionBodies, string(r.Body))
		}
	}
	if len(sessionBodies) != 2 {
		t.Fatalf("expected 2 create-session requests, got %d", len(sessionBodies))
	}
	if !strings.Contains(sessionBodies[0], `"couponId":"`+testCouponID+`"`) {
		t.Errorf("expected couponId in plan change body, got %s", sessionBodies[0])
	}
	if !strings.Contains(sessionBodies[1], `"promotionCode":"SPRING-20"`) || !strings.Contains(sessionBodies[1], `"allowPromotionCodes":true`) {
		t.Errorf("expected promotionCode and allowPromotionCodes in plan change body, got %s", sessionBodies[1])
	}
}

func TestPlanChange_DiscountFields_Rejected(t *testing.T) {
	ctx := context.Background()
	client, _, srv := newSignedTestClient(t)
	base := CreatePlanChangeSessionParams{OriginOrderID: testOriginOrderID, ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv", Currency: "USD"}

	couponAndCode := base
	couponAndCode.CouponID = Ptr(testCouponID)
	couponAndCode.PromotionCode = Ptr("SPRING-20")
	couponAndAllow := base
	couponAndAllow.CouponID = Ptr(testCouponID)
	couponAndAllow.AllowPromotionCodes = Ptr(false)
	wrongPrefix := base
	wrongPrefix.CouponID = Ptr("PMC_AbCdEfGhIjKlMnOpQrStUv")
	blank := base
	blank.PromotionCode = Ptr("")

	cases := []struct {
		name string
		call func() error
		want string
	}{
		{"couponId and promotionCode", func() error {
			_, err := client.Checkout.CreatePlanChangeSession(ctx, couponAndCode)
			return err
		}, "couponId and promotionCode are mutually exclusive"},
		{"couponId and allowPromotionCodes", func() error {
			_, err := client.Checkout.CreatePlanChangeSession(ctx, couponAndAllow)
			return err
		}, "couponId and allowPromotionCodes are mutually exclusive"},
		{"authenticated couponId and allowPromotionCodes", func() error {
			_, err := client.Checkout.Authenticated.CreatePlanChange(ctx, AuthenticatedPlanChangeParams{CreatePlanChangeSessionParams: couponAndAllow, BuyerIdentity: "user-1"})
			return err
		}, "mutually exclusive"},
		{"wrong prefix", func() error {
			_, err := client.Checkout.CreatePlanChangeSession(ctx, wrongPrefix)
			return err
		}, "CPN_ prefix"},
		{"empty promotionCode", func() error {
			_, err := client.Checkout.CreatePlanChangeSession(ctx, blank)
			return err
		}, "promotionCode cannot be empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertSDKError(t, tc.call(), tc.want)
		})
	}
	if n := len(srv.requests()); n != 0 {
		t.Fatalf("expected no requests, got %d", n)
	}
}

// assertSDKError checks that err is a 400 SDK-layer *Error whose message contains want.
func assertSDKError(t *testing.T, err error, want string) {
	t.Helper()
	var perr *Error
	if !errors.As(err, &perr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if perr.Status != 400 || perr.Errors[0].Layer != ErrorLayerSDK {
		t.Fatalf("expected 400 sdk error, got %d %s", perr.Status, perr.Errors[0].Layer)
	}
	if !strings.Contains(perr.Errors[0].Message, want) {
		t.Fatalf("expected message containing %q, got %q", want, perr.Errors[0].Message)
	}
}
