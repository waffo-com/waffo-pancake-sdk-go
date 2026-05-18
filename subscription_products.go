package pancake

import "context"

// SubscriptionProductsResource manages recurring subscription products.
type SubscriptionProductsResource struct {
	http *httpClient
}

// Create creates a subscription product with billing period and pricing.
func (r *SubscriptionProductsResource) Create(ctx context.Context, p CreateSubscriptionProductParams) (*SubscriptionProductResult, error) {
	if err := validateShortID("storeId", p.StoreID, "STO"); err != nil {
		return nil, err
	}
	if err := validateRequired("name", p.Name); err != nil {
		return nil, err
	}
	if err := validateBillingPeriod(p.BillingPeriod); err != nil {
		return nil, err
	}
	if err := validatePrices("prices", p.Prices); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[SubscriptionProductResult](r.http, ctx, "/v1/actions/subscription-product/create-product", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Update creates a new immutable subscription product version.
func (r *SubscriptionProductsResource) Update(ctx context.Context, p UpdateSubscriptionProductParams) (*SubscriptionProductResult, error) {
	if err := validateShortID("id", p.ID, "PROD"); err != nil {
		return nil, err
	}
	if p.Name != nil {
		if err := validateRequired("name", *p.Name); err != nil {
			return nil, err
		}
	}
	if p.BillingPeriod != nil {
		if err := validateBillingPeriod(*p.BillingPeriod); err != nil {
			return nil, err
		}
	}
	if len(p.Prices) > 0 {
		if err := validatePrices("prices", p.Prices); err != nil {
			return nil, err
		}
	}
	out, warnings, err := postAction[SubscriptionProductResult](r.http, ctx, "/v1/actions/subscription-product/update-product", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Publish promotes a subscription product's test version to production.
func (r *SubscriptionProductsResource) Publish(ctx context.Context, p PublishSubscriptionProductParams) (*SubscriptionProductResult, error) {
	if err := validateShortID("id", p.ID, "PROD"); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[SubscriptionProductResult](r.http, ctx, "/v1/actions/subscription-product/publish-product", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// UpdateStatus flips a subscription product between active and inactive.
func (r *SubscriptionProductsResource) UpdateStatus(ctx context.Context, p UpdateSubscriptionStatusParams) (*SubscriptionProductResult, error) {
	if err := validateShortID("id", p.ID, "PROD"); err != nil {
		return nil, err
	}
	if err := validateProductStatus(p.Status); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[SubscriptionProductResult](r.http, ctx, "/v1/actions/subscription-product/update-status", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

func validateBillingPeriod(bp BillingPeriod) error {
	switch bp {
	case BillingPeriodWeekly, BillingPeriodMonthly, BillingPeriodQuarterly, BillingPeriodYearly:
		return nil
	default:
		return newSDKError("Invalid billingPeriod: expected one of [weekly, monthly, quarterly, yearly], got %q", string(bp))
	}
}
