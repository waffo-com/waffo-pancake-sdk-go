package pancake

import "context"

// StoreMerchantsResource manages store membership.
//
// Coming soon — the underlying endpoints currently return HTTP 501.
type StoreMerchantsResource struct {
	http *httpClient
}

// Add invites a merchant to a store with the given role ("admin" or "member").
func (r *StoreMerchantsResource) Add(ctx context.Context, p AddMerchantParams) (*AddMerchantResult, error) {
	if err := validateShortID("storeId", p.StoreID, "STO"); err != nil {
		return nil, err
	}
	if err := validateRequired("email", p.Email); err != nil {
		return nil, err
	}
	if err := validateMerchantRole(p.Role); err != nil {
		return nil, err
	}
	var out AddMerchantResult
	if err := r.http.post(ctx, "/v1/actions/store-merchant/add-merchant", p, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Remove removes a merchant from a store.
func (r *StoreMerchantsResource) Remove(ctx context.Context, p RemoveMerchantParams) (*RemoveMerchantResult, error) {
	if err := validateShortID("storeId", p.StoreID, "STO"); err != nil {
		return nil, err
	}
	if err := validateShortID("merchantId", p.MerchantID, "MER"); err != nil {
		return nil, err
	}
	var out RemoveMerchantResult
	if err := r.http.post(ctx, "/v1/actions/store-merchant/remove-merchant", p, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateRole changes a merchant's role within a store.
func (r *StoreMerchantsResource) UpdateRole(ctx context.Context, p UpdateRoleParams) (*UpdateRoleResult, error) {
	if err := validateShortID("storeId", p.StoreID, "STO"); err != nil {
		return nil, err
	}
	if err := validateShortID("merchantId", p.MerchantID, "MER"); err != nil {
		return nil, err
	}
	if err := validateMerchantRole(p.Role); err != nil {
		return nil, err
	}
	var out UpdateRoleResult
	if err := r.http.post(ctx, "/v1/actions/store-merchant/update-role", p, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func validateMerchantRole(role string) error {
	if role != "admin" && role != "member" {
		return newSDKError("Invalid role: expected one of [admin, member], got %q", role)
	}
	return nil
}
