package pancake

import "context"

// AuthResource issues session tokens for buyers.
type AuthResource struct {
	http *httpClient
}

// IssueSessionToken mints a buyer session JWT. Provide either StoreID or
// ProductID — when only ProductID is given the server derives the store from
// the product.
//
// Example:
//
//	tok, err := client.Auth.IssueSessionToken(ctx, pancake.IssueSessionTokenParams{
//	    StoreID:       pancake.Ptr("STO_..."),
//	    BuyerIdentity: "customer@example.com",
//	})
func (r *AuthResource) IssueSessionToken(ctx context.Context, p IssueSessionTokenParams) (*SessionToken, error) {
	if p.StoreID == nil && p.ProductID == nil {
		return nil, newSDKError("Missing required field: provide storeId or productId")
	}
	if p.StoreID != nil {
		if err := validateShortID("storeId", *p.StoreID, "STO"); err != nil {
			return nil, err
		}
	}
	if p.ProductID != nil {
		if err := validateShortID("productId", *p.ProductID, "PROD"); err != nil {
			return nil, err
		}
	}
	if err := validateRequired("buyerIdentity", p.BuyerIdentity); err != nil {
		return nil, err
	}
	var out SessionToken
	if err := r.http.post(ctx, "/v1/actions/auth/issue-session-token", p, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
