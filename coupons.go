package pancake

import "context"

// CouponsResource manages coupons (discount codes). A coupon's Code becomes its
// default promotion code, the code buyers type. Only the name of a coupon can
// change; deactivation is permanent.
type CouponsResource struct {
	http *httpClient
}

// Create creates a coupon. The returned status is active once the payment
// provider confirms, or pending while confirmation is in progress (it resolves
// automatically). Retry with the same idempotency key: without it a retry is
// rejected with 409 because the first request already took the code.
//
// Example:
//
//	res, err := client.Coupons.Create(ctx, pancake.CreateCouponParams{
//		StoreID:  "STO_xxx",
//		Name:     "Black Friday 20% off",
//		Code:     "BF2026",
//		Discount: pancake.CouponDiscount{Type: pancake.CouponDiscountTypePercentage, PercentOff: pancake.Ptr(20.0)},
//	}, pancake.WithIdempotencyKey("coupon-bf2026"))
func (r *CouponsResource) Create(ctx context.Context, p CreateCouponParams, opts ...RequestOption) (*CouponResult, error) {
	if err := validateShortID("storeId", p.StoreID, "STO"); err != nil {
		return nil, err
	}
	if err := validateRequired("name", p.Name); err != nil {
		return nil, err
	}
	if err := validateMaxLength("name", &p.Name, 64); err != nil {
		return nil, err
	}
	if err := validateDiscountCode("code", p.Code); err != nil {
		return nil, err
	}
	switch p.Discount.Type {
	case CouponDiscountTypePercentage:
		if p.Discount.PercentOff == nil {
			return nil, newSDKError("Missing required field: discount.percentOff")
		}
	case CouponDiscountTypeFixedAmount:
		if len(p.Discount.AmountOff) == 0 {
			return nil, newSDKError("Missing required field: discount.amountOff")
		}
		for currency, amount := range p.Discount.AmountOff {
			if err := validateCurrencyCode("discount.amountOff (key)", currency); err != nil {
				return nil, err
			}
			if err := validateAmountString("discount.amountOff."+currency, amount); err != nil {
				return nil, err
			}
		}
	default:
		return nil, newSDKError("Invalid discount.type: expected one of [percentage, fixed_amount], got %q", p.Discount.Type)
	}
	if p.Duration != nil && *p.Duration == CouponDurationRepeating {
		if p.DurationInPeriods == nil {
			return nil, newSDKError("Missing required field: durationInPeriods")
		}
		if err := validatePositiveInt("durationInPeriods", *p.DurationInPeriods); err != nil {
			return nil, err
		}
	}
	for _, productID := range p.ProductIDs {
		if err := validateShortID("productIds[]", productID, "PROD"); err != nil {
			return nil, err
		}
	}
	out, warnings, err := postAction[CouponResult](ctx, r.http, "/v1/actions/coupon/create-coupon", p, opts)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Update renames a coupon. Create a new coupon to change the discount.
//
// Example:
//
//	res, err := client.Coupons.Update(ctx, pancake.UpdateCouponParams{ID: "CPN_xxx", Name: "Black Friday (extended)"})
func (r *CouponsResource) Update(ctx context.Context, p UpdateCouponParams, opts ...RequestOption) (*CouponResult, error) {
	if err := validateShortID("id", p.ID, "CPN"); err != nil {
		return nil, err
	}
	if err := validateRequired("name", p.Name); err != nil {
		return nil, err
	}
	if err := validateMaxLength("name", &p.Name, 64); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[CouponResult](ctx, r.http, "/v1/actions/coupon/update-coupon", p, opts)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Deactivate deactivates a coupon permanently. All its promotion codes are
// deactivated and their codes released. Discounts already attached to
// subscriptions continue until the subscription's next plan change. Calling it
// again returns the coupon unchanged.
//
// Example:
//
//	res, err := client.Coupons.Deactivate(ctx, pancake.DeactivateCouponParams{ID: "CPN_xxx"})
func (r *CouponsResource) Deactivate(ctx context.Context, p DeactivateCouponParams, opts ...RequestOption) (*CouponResult, error) {
	if err := validateShortID("id", p.ID, "CPN"); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[CouponResult](ctx, r.http, "/v1/actions/coupon/deactivate-coupon", p, opts)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}
