package pancake

import (
	"context"
	"sync"
)

// CheckoutResource creates checkout sessions and offers two convenience
// sub-resources (Anonymous and Authenticated) for the most common flows.
// CreatePlanChangeSession covers the third flow — changing the plan of an
// existing subscription. Anonymous has no plan change counterpart: a Store Slug
// session is an anonymous credential with no subscription to attribute the change
// to, and the platform answers 403 for it.
type CheckoutResource struct {
	// Anonymous creates checkout sessions without a customer identity.
	Anonymous *CheckoutAnonymousResource
	// Authenticated creates checkout sessions that include a customer-session
	// token for post-purchase self-service.
	Authenticated *CheckoutAuthenticatedResource

	http *httpClient
}

func newCheckoutResource(h *httpClient) *CheckoutResource {
	return &CheckoutResource{
		http:          h,
		Anonymous:     &CheckoutAnonymousResource{http: h},
		Authenticated: &CheckoutAuthenticatedResource{http: h},
	}
}

// CreateSession is the low-level checkout-session endpoint. Most callers
// should prefer Checkout.Anonymous.Create or Checkout.Authenticated.Create.
//
// Example:
//
//	session, err := client.Checkout.CreateSession(ctx, pancake.CreateCheckoutSessionParams{
//	    ProductID: "PROD_...",
//	    Currency:  "USD",
//	})
//	// Redirect the customer to session.CheckoutURL.
func (r *CheckoutResource) CreateSession(ctx context.Context, p CreateCheckoutSessionParams) (*CheckoutSessionResult, error) {
	if err := validateCheckoutCommon(&p); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[CheckoutSessionResult](ctx, r.http, "/v1/actions/checkout/create-session", p, &postOptions{IdempotencyWindow: 60})
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// CreatePlanChangeSession issues a link that changes an existing subscription to
// another plan. It hits the same create-session endpoint as CreateSession, in the
// mode selected by OriginOrderID, and the returned CheckoutURL points at the change
// confirmation page (…/store/{slug}/change/{sessionId}) instead of the cashier.
//
// ChangeAmount, ChangeCreditAmount and WithTrial are merchant-credential only; the
// platform silently drops them for any other credential. ChangeAmount and
// ChangeCreditAmount are mutually exclusive and the platform rejects both-at-once
// with a 400 — this method forwards what the caller passes.
//
// Example:
//
//	session, err := client.Checkout.CreatePlanChangeSession(ctx, pancake.CreatePlanChangeSessionParams{
//	    OriginOrderID:      "ORD_...",
//	    ProductID:          "PROD_...",
//	    Currency:           "USD",
//	    ChangeTiming:       pancake.Ptr(pancake.ChangeTimingImmediate),
//	    ChangeCreditAmount: pancake.Ptr("8.00"),
//	})
//	// Send the customer to session.CheckoutURL.
func (r *CheckoutResource) CreatePlanChangeSession(ctx context.Context, p CreatePlanChangeSessionParams) (*CheckoutSessionResult, error) {
	if err := validatePlanChangeCommon(&p); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[CheckoutSessionResult](ctx, r.http, "/v1/actions/checkout/create-session", p, &postOptions{IdempotencyWindow: 60})
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// CheckoutAnonymousResource creates checkout sessions without a customer
// identity. The form on the checkout page is left blank unless BuyerEmail or
// BillingDetail are supplied.
type CheckoutAnonymousResource struct {
	http *httpClient
}

// Create creates an anonymous checkout session.
//
// Example:
//
//	res, err := client.Checkout.Anonymous.Create(ctx, pancake.AnonymousCheckoutParams{
//	    ProductID: "PROD_...",
//	    Currency:  "USD",
//	})
func (r *CheckoutAnonymousResource) Create(ctx context.Context, p AnonymousCheckoutParams) (*CheckoutSessionResult, error) {
	if err := validateCheckoutCommon(&p); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[CheckoutSessionResult](ctx, r.http, "/v1/actions/checkout/create-session", p, &postOptions{IdempotencyWindow: 60})
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// CheckoutAuthenticatedResource creates checkout sessions with a merchant-
// provided customer identity. It issues a session token in parallel with the
// session creation and appends "#token=..." to the returned URL. Create issues a
// new-purchase link, CreatePlanChange a plan change link.
type CheckoutAuthenticatedResource struct {
	http *httpClient
}

// Create issues a customer session token and creates a checkout session
// concurrently, then returns a unified result whose CheckoutURL carries the
// token as a URL fragment.
//
// BuyerIdentity is sent only to the issue-session-token endpoint; every other
// field is forwarded to the create-session endpoint unchanged.
//
// Example:
//
//	res, err := client.Checkout.Authenticated.Create(ctx, pancake.AuthenticatedCheckoutParams{
//	    CreateCheckoutSessionParams: pancake.CreateCheckoutSessionParams{
//	        ProductID:  "PROD_...",
//	        Currency:   "USD",
//	        BuyerEmail: pancake.Ptr("customer@example.com"),
//	    },
//	    BuyerIdentity: "user-123",
//	})
func (r *CheckoutAuthenticatedResource) Create(ctx context.Context, p AuthenticatedCheckoutParams) (*AuthenticatedCheckoutResult, error) {
	if err := validateCheckoutCommon(&p.CreateCheckoutSessionParams); err != nil {
		return nil, err
	}
	if err := validateRequired("buyerIdentity", p.BuyerIdentity); err != nil {
		return nil, err
	}
	return r.issueTokenAndSession(ctx, p.ProductID, p.BuyerIdentity, p.CreateCheckoutSessionParams)
}

// CreatePlanChange issues a plan change link for an existing subscription together
// with a customer session token, so the customer reaches the change confirmation
// page (…/store/{slug}/change/{sessionId}) already signed in. BuyerIdentity is sent
// only to the issue-session-token endpoint; every other field goes to create-session.
//
// The request is signed with the merchant API Key, so ChangeAmount,
// ChangeCreditAmount and WithTrial are honored here; a customer credential calling
// the endpoint directly would have them silently dropped. ChangeAmount and
// ChangeCreditAmount remain mutually exclusive — the platform answers 400 when both
// are sent.
//
// Example:
//
//	res, err := client.Checkout.Authenticated.CreatePlanChange(ctx, pancake.AuthenticatedPlanChangeParams{
//	    CreatePlanChangeSessionParams: pancake.CreatePlanChangeSessionParams{
//	        OriginOrderID: "ORD_...",
//	        ProductID:     "PROD_...",
//	        Currency:      "USD",
//	        ChangeTiming:  pancake.Ptr(pancake.ChangeTimingNextPeriod),
//	    },
//	    BuyerIdentity: "user-123",
//	})
func (r *CheckoutAuthenticatedResource) CreatePlanChange(ctx context.Context, p AuthenticatedPlanChangeParams) (*AuthenticatedCheckoutResult, error) {
	if err := validatePlanChangeCommon(&p.CreatePlanChangeSessionParams); err != nil {
		return nil, err
	}
	if err := validateRequired("buyerIdentity", p.BuyerIdentity); err != nil {
		return nil, err
	}
	return r.issueTokenAndSession(ctx, p.ProductID, p.BuyerIdentity, p.CreatePlanChangeSessionParams)
}

// issueTokenAndSession issues a session token and creates a checkout session
// concurrently, then merges both responses into one result. sessionBody is the
// create-session payload, which differs between a new purchase and a plan change.
func (r *CheckoutAuthenticatedResource) issueTokenAndSession(
	ctx context.Context,
	productID, buyerIdentity string,
	sessionBody any,
) (*AuthenticatedCheckoutResult, error) {
	tokenBody := struct {
		ProductID     string `json:"productId"`
		BuyerIdentity string `json:"buyerIdentity"`
	}{
		ProductID:     productID,
		BuyerIdentity: buyerIdentity,
	}

	var (
		tok          *SessionToken
		session      *CheckoutSessionResult
		tokWarnings  []Notice
		sessWarnings []Notice
		errTok       error
		errSess      error
		wg           sync.WaitGroup
	)

	wg.Add(2)
	opts := &postOptions{IdempotencyWindow: 60}
	go func() {
		defer wg.Done()
		tok, tokWarnings, errTok = postAction[SessionToken](ctx, r.http, "/v1/actions/auth/issue-session-token", tokenBody, opts)
	}()
	go func() {
		defer wg.Done()
		session, sessWarnings, errSess = postAction[CheckoutSessionResult](ctx, r.http, "/v1/actions/checkout/create-session", sessionBody, opts)
	}()
	wg.Wait()

	if errTok != nil {
		return nil, errTok
	}
	if errSess != nil {
		return nil, errSess
	}

	warnings := append(append([]Notice(nil), tokWarnings...), sessWarnings...)
	return &AuthenticatedCheckoutResult{
		SessionID:      session.SessionID,
		CheckoutURL:    session.CheckoutURL + "#token=" + tok.Token,
		ExpiresAt:      session.ExpiresAt,
		Token:          tok.Token,
		TokenExpiresAt: tok.ExpiresAt,
		Warnings:       warnings,
	}, nil
}
