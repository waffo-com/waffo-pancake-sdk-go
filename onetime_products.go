package pancake

import "context"

// OnetimeProductsResource manages one-time (non-subscription) products.
type OnetimeProductsResource struct {
	http *httpClient
}

// Create creates a one-time product with multi-currency pricing.
func (r *OnetimeProductsResource) Create(ctx context.Context, p CreateOnetimeProductParams) (*OnetimeProductResult, error) {
	if err := validateShortID("storeId", p.StoreID, "STO"); err != nil {
		return nil, err
	}
	if err := validateRequired("name", p.Name); err != nil {
		return nil, err
	}
	if err := validatePrices("prices", p.Prices); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[OnetimeProductResult](ctx, r.http, "/v1/actions/onetime-product/create-product", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Update creates a new immutable version of the product and skips when the
// request would not change anything.
func (r *OnetimeProductsResource) Update(ctx context.Context, p UpdateOnetimeProductParams) (*OnetimeProductResult, error) {
	if err := validateShortID("id", p.ID, "PROD"); err != nil {
		return nil, err
	}
	if p.Name != nil {
		if err := validateRequired("name", *p.Name); err != nil {
			return nil, err
		}
	}
	if len(p.Prices) > 0 {
		if err := validatePrices("prices", p.Prices); err != nil {
			return nil, err
		}
	}
	out, warnings, err := postAction[OnetimeProductResult](ctx, r.http, "/v1/actions/onetime-product/update-product", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Publish promotes a one-time product's test version to production.
func (r *OnetimeProductsResource) Publish(ctx context.Context, p PublishOnetimeProductParams) (*OnetimeProductResult, error) {
	if err := validateShortID("id", p.ID, "PROD"); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[OnetimeProductResult](ctx, r.http, "/v1/actions/onetime-product/publish-product", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// UpdateStatus flips a one-time product between active and inactive.
func (r *OnetimeProductsResource) UpdateStatus(ctx context.Context, p UpdateOnetimeStatusParams) (*OnetimeProductResult, error) {
	if err := validateShortID("id", p.ID, "PROD"); err != nil {
		return nil, err
	}
	if err := validateProductStatus(p.Status); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[OnetimeProductResult](ctx, r.http, "/v1/actions/onetime-product/update-status", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

func validateProductStatus(s ProductVersionStatus) error {
	if s != ProductVersionStatusActive && s != ProductVersionStatusInactive {
		return newSDKError("Invalid status: expected one of [active, inactive], got %q", string(s))
	}
	return nil
}
