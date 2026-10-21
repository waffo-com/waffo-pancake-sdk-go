package pancake

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// TestCoupons_CreateSendsPromotionCodes checks that Coupons.Create sends the
// promotion codes in the request body, sends no top-level code, and decodes
// the promotion codes returned with the coupon.
func TestCoupons_CreateSendsPromotionCodes(t *testing.T) {
	ctx := context.Background()
	client, _, srv := newSignedTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"coupon": map[string]any{
			"id": "CPN_AbCdEfGhIjKlMnOpQrStUv", "name": "BF", "code": "BF-MAIN", "status": "active",
			"promotionCodes": []any{
				map[string]any{"id": "PMC_AbCdEfGhIjKlMnOpQrStUv", "couponId": "CPN_AbCdEfGhIjKlMnOpQrStUv", "code": "BF2026"},
				map[string]any{"id": "PMC_BbCdEfGhIjKlMnOpQrStUv", "couponId": "CPN_AbCdEfGhIjKlMnOpQrStUv", "code": "X7K2M9PQ4R"},
			},
		}}}
	}

	res, err := client.Coupons.Create(ctx, CreateCouponParams{
		StoreID:  "STO_AbCdEfGhIjKlMnOpQrStUv",
		Name:     "BF",
		Code:     Ptr("bf-main"),
		Discount: CouponDiscount{Type: CouponDiscountTypePercentage, PercentOff: Ptr(20.0)},
		PromotionCodes: []CreateCouponPromotionCodeParams{
			{Code: Ptr("bf2026")},
			{Restrictions: &PromotionCodeRestrictions{MaxRedemptions: Ptr(100), DailyRedemptionLimit: Ptr(1)}},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Coupon.Name != "BF" || res.Coupon.Code != "BF-MAIN" || len(res.Coupon.PromotionCodes) != 2 || res.Coupon.PromotionCodes[1].Code != "X7K2M9PQ4R" {
		t.Errorf("unexpected coupon %+v", res.Coupon)
	}

	requests := srv.requests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	var body map[string]any
	if err := json.Unmarshal(requests[0].Body, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] != "bf-main" {
		t.Errorf("expected the coupon code bf-main, got %s", requests[0].Body)
	}
	codes, ok := body["promotionCodes"].([]any)
	if !ok || len(codes) != 2 {
		t.Fatalf("expected 2 promotionCodes, got %s", requests[0].Body)
	}
	if first := codes[0].(map[string]any); first["code"] != "bf2026" {
		t.Errorf("expected first code bf2026, got %v", first["code"])
	}
	second := codes[1].(map[string]any)
	if second["code"] != nil {
		t.Errorf("expected the second entry to omit code, got %v", second["code"])
	}
	if restrictions, _ := second["restrictions"].(map[string]any); restrictions["dailyRedemptionLimit"] != float64(1) {
		t.Errorf("expected restrictions.dailyRedemptionLimit 1, got %v", second["restrictions"])
	}
}

// TestCoupons_CreateWithoutPromotionCodesOmitsField checks that a coupon
// created without promotion codes sends no promotionCodes field.
func TestCoupons_CreateWithoutPromotionCodesOmitsField(t *testing.T) {
	ctx := context.Background()
	client, _, srv := newSignedTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"coupon": map[string]any{"id": "CPN_AbCdEfGhIjKlMnOpQrStUv", "promotionCodes": []any{}}}}
	}

	res, err := client.Coupons.Create(ctx, CreateCouponParams{
		StoreID:  "STO_AbCdEfGhIjKlMnOpQrStUv",
		Name:     "Partner",
		Discount: CouponDiscount{Type: CouponDiscountTypeFixedAmount, AmountOff: map[string]string{"USD": "5.00"}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(res.Coupon.PromotionCodes) != 0 {
		t.Errorf("expected no promotion codes, got %+v", res.Coupon.PromotionCodes)
	}
	var body map[string]any
	if err := json.Unmarshal(srv.requests()[0].Body, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, ok := body["promotionCodes"]; ok {
		t.Errorf("expected promotionCodes to be omitted, got %s", srv.requests()[0].Body)
	}
	if _, ok := body["code"]; ok {
		t.Errorf("expected code to be omitted so the platform generates one, got %s", srv.requests()[0].Body)
	}
}

// TestCoupons_CodesAreTrimmedBeforeValidation checks that codes with
// surrounding whitespace pass the format check, as the server trims them, that
// duplicates are detected after trimming, and that the trimmed length counts.
func TestCoupons_CodesAreTrimmedBeforeValidation(t *testing.T) {
	ctx := context.Background()
	client, _, srv := newSignedTestClient(t)
	srv.respond = func(r recordedRequest) (int, any) {
		if strings.Contains(r.Path, "promotion-code") {
			return 200, map[string]any{"data": map[string]any{"promotionCode": map[string]any{"id": "PMC_AbCdEfGhIjKlMnOpQrStUv", "code": "KOL-ALICE"}}}
		}
		return 200, map[string]any{"data": map[string]any{"coupon": map[string]any{"id": "CPN_AbCdEfGhIjKlMnOpQrStUv", "promotionCodes": []any{}}}}
	}
	discount := CouponDiscount{Type: CouponDiscountTypePercentage, PercentOff: Ptr(10.0)}

	if _, err := client.Coupons.Create(ctx, CreateCouponParams{
		StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Name: "Trimmed", Discount: discount,
		Code:           Ptr(" bf2026 "),
		PromotionCodes: []CreateCouponPromotionCodeParams{{Code: Ptr("  VIP-01\t")}},
	}); err != nil {
		t.Fatalf("Coupons.Create with padded codes: %v", err)
	}
	if _, err := client.PromotionCodes.Create(ctx, CreatePromotionCodeParams{
		CouponID: "CPN_AbCdEfGhIjKlMnOpQrStUv", Code: Ptr(" kol-alice "),
	}); err != nil {
		t.Fatalf("PromotionCodes.Create with a padded code: %v", err)
	}
	sent := len(srv.requests())

	rejected := []struct {
		name string
		call func() error
	}{
		{"duplicate promotion codes after trimming", func() error {
			_, err := client.Coupons.Create(ctx, CreateCouponParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Name: "X", Discount: discount,
				PromotionCodes: []CreateCouponPromotionCodeParams{{Code: Ptr(" vip-01 ")}, {Code: Ptr("VIP-01")}},
			})
			return err
		}},
		{"promotion code repeating the coupon code after trimming", func() error {
			_, err := client.Coupons.Create(ctx, CreateCouponParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Name: "X", Discount: discount,
				Code: Ptr("bf2026 "), PromotionCodes: []CreateCouponPromotionCodeParams{{Code: Ptr(" BF2026")}},
			})
			return err
		}},
		{"coupon code too short after trimming", func() error {
			_, err := client.Coupons.Create(ctx, CreateCouponParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Name: "X", Discount: discount, Code: Ptr(" ab "),
			})
			return err
		}},
		{"promotion code too short after trimming", func() error {
			_, err := client.PromotionCodes.Create(ctx, CreatePromotionCodeParams{
				CouponID: "CPN_AbCdEfGhIjKlMnOpQrStUv", Code: Ptr(" ab "),
			})
			return err
		}},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			var e *Error
			if err := tc.call(); !errors.As(err, &e) || e.Status != 400 {
				t.Fatalf("expected a 400 SDK error, got %v", err)
			}
		})
	}
	if got := len(srv.requests()); got != sent {
		t.Errorf("expected rejected calls to send nothing, got %d extra requests", got-sent)
	}
}
