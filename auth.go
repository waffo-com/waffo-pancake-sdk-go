package pancake

import "context"

// portalSessionToken is the issue-session-token data block for
// purpose "portal".
type portalSessionToken struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	PortalURL string `json:"portalUrl"`
}

// portalSessionTokenBody is the issue-session-token request body for
// purpose "portal". ProductID is never sent: the platform rejects it here.
type portalSessionTokenBody struct {
	StoreID       string `json:"storeId"`
	BuyerIdentity string `json:"buyerIdentity"`
	Purpose       string `json:"purpose"`
}

// AuthResource issues session tokens and customer portal links for customers.
type AuthResource struct {
	http *httpClient
}

// IssueSessionToken mints a customer session JWT. Provide either StoreID or
// ProductID — when only ProductID is given the server derives the store from
// the product.
//
// Example:
//
//	tok, err := client.Auth.IssueSessionToken(ctx, pancake.IssueSessionTokenParams{
//	    StoreID:       pancake.Ptr("STO_..."),
//	    BuyerIdentity: "customer@example.com",
//	})
func (r *AuthResource) IssueSessionToken(ctx context.Context, p IssueSessionTokenParams, opts ...RequestOption) (*SessionToken, error) {
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
	out, warnings, err := postAction[SessionToken](ctx, r.http, "/v1/actions/auth/issue-session-token", p, opts)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// CreateCustomerPortalLink creates a link that signs the customer straight into
// the store's Pancake customer portal, without the portal's email verification
// step.
//
// It calls issue-session-token with purpose "portal" and appends the returned
// token to the returned portal URL as a fragment ("#token=..."), the same way
// Checkout.Authenticated.Create builds CheckoutURL. Fragments are not sent in
// HTTP requests or Referer headers. The portal environment is the environment
// of the API Key this client signs with; there is no environment parameter.
//
// StoreID must be a Store Short ID and BuyerIdentity non-blank; both are
// checked before any request (*Error, 400, ErrorLayerSDK). A platform response
// without portalUrl or token returns *Error with ErrorLayerSDK.
//
// Take StoreID and BuyerIdentity from your backend's authenticated session,
// never from request parameters the browser controls. Redirect to PortalURL
// right away and do not log or cache it — it carries a bearer token.
//
// Example:
//
//	user, ok := requireSignedInUser(r) // your own session check
//	if !ok {
//	    http.Redirect(w, r, "/login", http.StatusFound)
//	    return
//	}
//	link, err := client.Auth.CreateCustomerPortalLink(r.Context(), pancake.CreateCustomerPortalLinkParams{
//	    StoreID:       "STO_...",
//	    BuyerIdentity: user.ID,
//	})
//	if err != nil {
//	    http.Error(w, "portal unavailable", http.StatusBadGateway)
//	    return
//	}
//	w.Header().Set("Cache-Control", "no-store")
//	w.Header().Set("Referrer-Policy", "no-referrer")
//	http.Redirect(w, r, link.PortalURL, http.StatusFound)
func (r *AuthResource) CreateCustomerPortalLink(ctx context.Context, p CreateCustomerPortalLinkParams, opts ...RequestOption) (*CustomerPortalLink, error) {
	if err := validateShortID("storeId", p.StoreID, "STO"); err != nil {
		return nil, err
	}
	if err := validateRequired("buyerIdentity", p.BuyerIdentity); err != nil {
		return nil, err
	}
	body := portalSessionTokenBody{StoreID: p.StoreID, BuyerIdentity: p.BuyerIdentity, Purpose: "portal"}
	// Not postAction: the missing-field errors below carry the response's HTTP
	// status, which postAction does not return.
	status, env, err := r.http.post(ctx, "/v1/actions/auth/issue-session-token", body, newRequestOptions(opts))
	if err != nil {
		return nil, err
	}
	tok, warnings, err := unwrapEnvelope[portalSessionToken](status, env)
	if err != nil {
		return nil, err
	}
	if tok.PortalURL == "" {
		return nil, &Error{
			Status: status,
			Errors: []APIError{{Message: `Missing portalUrl in issue-session-token response for purpose "portal"`, Layer: ErrorLayerSDK}},
		}
	}
	if tok.Token == "" {
		return nil, &Error{
			Status: status,
			Errors: []APIError{{Message: `Missing token in issue-session-token response for purpose "portal"`, Layer: ErrorLayerSDK}},
		}
	}
	return &CustomerPortalLink{
		PortalURL: tok.PortalURL + "#token=" + tok.Token,
		ExpiresAt: tok.ExpiresAt,
		Warnings:  warnings,
	}, nil
}
