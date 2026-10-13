package pancake

import "context"

// PromotionCodesResource manages promotion codes: buyer-facing codes that point
// to a coupon and can add restrictions on top of the coupon's own rules.
type PromotionCodesResource struct {
	http *httpClient
}

// Create creates a promotion code for a coupon. A random code is generated
// when Code is nil.
//
// Example:
//
//	res, err := client.PromotionCodes.Create(ctx, pancake.CreatePromotionCodeParams{
//		CouponID: "CPN_xxx",
//		Code:     pancake.Ptr("KOL-ALICE"),
//		Restrictions: &pancake.PromotionCodeRestrictions{
//			MaxRedemptions: pancake.Ptr(100),
//			MinimumAmount:  map[string]string{"USD": "50.00"},
//		},
//	})
func (r *PromotionCodesResource) Create(ctx context.Context, p CreatePromotionCodeParams, opts ...RequestOption) (*PromotionCodeResult, error) {
	if err := validateShortID("couponId", p.CouponID, "CPN"); err != nil {
		return nil, err
	}
	if p.Code != nil {
		if err := validateDiscountCode("code", *p.Code); err != nil {
			return nil, err
		}
	}
	out, warnings, err := postAction[PromotionCodeResult](ctx, r.http, "/v1/actions/promotion-code/create-promotion-code", p, opts)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Generate generates 1-500 random promotion codes for a coupon in one call.
// Always retry with the same idempotency key: codes are random, so a retry
// without it generates another batch.
//
// Example:
//
//	res, err := client.PromotionCodes.Generate(ctx, pancake.GeneratePromotionCodesParams{
//		CouponID: "CPN_xxx",
//		Count:    100,
//		Prefix:   pancake.Ptr("KOL-"),
//	}, pancake.WithIdempotencyKey("kol-batch-2026-11"))
func (r *PromotionCodesResource) Generate(ctx context.Context, p GeneratePromotionCodesParams, opts ...RequestOption) (*GeneratePromotionCodesResult, error) {
	if err := validateShortID("couponId", p.CouponID, "CPN"); err != nil {
		return nil, err
	}
	if p.Count < 1 || p.Count > 500 {
		return nil, newSDKError("Invalid count: expected integer between 1 and 500, got %d", p.Count)
	}
	if err := validateMaxLength("prefix", p.Prefix, 12); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[GeneratePromotionCodesResult](ctx, r.http, "/v1/actions/promotion-code/generate-promotion-codes", p, opts)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Deactivate deactivates a promotion code permanently and releases its code.
// The coupon's own code can only be deactivated together with the coupon.
//
// Example:
//
//	res, err := client.PromotionCodes.Deactivate(ctx, pancake.DeactivatePromotionCodeParams{ID: "PMC_xxx"})
func (r *PromotionCodesResource) Deactivate(ctx context.Context, p DeactivatePromotionCodeParams, opts ...RequestOption) (*PromotionCodeResult, error) {
	if err := validateShortID("id", p.ID, "PMC"); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[PromotionCodeResult](ctx, r.http, "/v1/actions/promotion-code/deactivate-promotion-code", p, opts)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}
