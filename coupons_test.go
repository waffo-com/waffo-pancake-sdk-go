package pancake

import (
	"context"
	"encoding/json"
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
			"id": "CPN_AbCdEfGhIjKlMnOpQrStUv", "name": "BF", "status": "active",
			"promotionCodes": []any{
				map[string]any{"id": "PMC_AbCdEfGhIjKlMnOpQrStUv", "couponId": "CPN_AbCdEfGhIjKlMnOpQrStUv", "code": "BF2026"},
				map[string]any{"id": "PMC_BbCdEfGhIjKlMnOpQrStUv", "couponId": "CPN_AbCdEfGhIjKlMnOpQrStUv", "code": "X7K2M9PQ4R"},
			},
		}}}
	}

	res, err := client.Coupons.Create(ctx, CreateCouponParams{
		StoreID:  "STO_AbCdEfGhIjKlMnOpQrStUv",
		Name:     "BF",
		Discount: CouponDiscount{Type: CouponDiscountTypePercentage, PercentOff: Ptr(20.0)},
		PromotionCodes: []CreateCouponPromotionCodeParams{
			{Code: Ptr("bf2026")},
			{Restrictions: &PromotionCodeRestrictions{MaxRedemptions: Ptr(100)}},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Coupon.Name != "BF" || len(res.Coupon.PromotionCodes) != 2 || res.Coupon.PromotionCodes[1].Code != "X7K2M9PQ4R" {
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
	if _, ok := body["code"]; ok {
		t.Errorf("expected no top-level code, got %s", requests[0].Body)
	}
	codes, ok := body["promotionCodes"].([]any)
	if !ok || len(codes) != 2 {
		t.Fatalf("expected 2 promotionCodes, got %s", requests[0].Body)
	}
	if first := codes[0].(map[string]any); first["code"] != "bf2026" {
		t.Errorf("expected first code bf2026, got %v", first["code"])
	}
	if second := codes[1].(map[string]any); second["code"] != nil {
		t.Errorf("expected the second entry to omit code, got %v", second["code"])
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
}
