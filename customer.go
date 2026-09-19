package pancake

import "context"

// CustomerSession exposes the customer self-service API surface backed by a
// session token. Construct via [Client.Customer]. All HTTP methods use Bearer
// authentication.
//
// These requests carry no idempotency key. The API Key client derives one per write
// and the gateway deduplicates on it; customer session actions are outside that
// cache by design, so a write retried after a timeout can execute twice. Guard
// retries on the caller side where a duplicate would matter.
type CustomerSession struct {
	// GraphQL runs customer-scoped GraphQL queries.
	GraphQL *CustomerGraphQLResource

	http *customerHTTPClient
}

func newCustomerSession(h *customerHTTPClient) *CustomerSession {
	return &CustomerSession{
		http:    h,
		GraphQL: &CustomerGraphQLResource{http: h},
	}
}

// CancelSubscription cancels a customer's subscription order.
//
//   - pending  -> canceled
//   - active   -> canceling at the end of the current billing period
//   - past_due -> canceling immediately (the billing period has already lapsed)
//
// The PSP cancellation is confirmed asynchronously in both canceling cases.
func (s *CustomerSession) CancelSubscription(ctx context.Context, p CancelSubscriptionParams) (*CancelSubscriptionResult, error) {
	if err := validateShortID("orderId", p.OrderID, "ORD"); err != nil {
		return nil, err
	}
	out, warnings, err := customerPostAction[CancelSubscriptionResult](ctx, s.http, "/v1/actions/subscription-order/cancel-order", p)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// CancelOnetimeOrder cancels a one-time order whose payment is still pending.
func (s *CustomerSession) CancelOnetimeOrder(ctx context.Context, p CancelOnetimeOrderParams) (*CancelOnetimeOrderResult, error) {
	if err := validateShortID("orderId", p.OrderID, "ORD"); err != nil {
		return nil, err
	}
	out, warnings, err := customerPostAction[CancelOnetimeOrderResult](ctx, s.http, "/v1/actions/onetime-order/cancel-order", p)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// ReactivateSubscription reactivates a subscription currently in the
// "canceling" state.
//
// A subscription that had an unpaid charge at the moment cancellation was
// requested is refused with 400 ("Subscription with an unpaid balance cannot be
// reactivated"), which is distinct from the 400 returned when the order is not
// in the "canceling" state. Cancelling a past_due subscription always falls
// into the former category.
func (s *CustomerSession) ReactivateSubscription(ctx context.Context, p ReactivateSubscriptionParams) (*ReactivateSubscriptionResult, error) {
	if err := validateShortID("orderId", p.OrderID, "ORD"); err != nil {
		return nil, err
	}
	out, warnings, err := customerPostAction[ReactivateSubscriptionResult](ctx, s.http, "/v1/actions/subscription-order/reactivate-order", p)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// CreateRefundTicket submits a refund request for a payment.
func (s *CustomerSession) CreateRefundTicket(ctx context.Context, p CreateRefundTicketParams) (*RefundTicketResult, error) {
	if err := validateShortID("paymentId", p.PaymentID, "PAY"); err != nil {
		return nil, err
	}
	if err := validateRequired("reason", p.Reason); err != nil {
		return nil, err
	}
	if err := validateAmountString("requestedAmount.amount", p.RequestedAmount.Amount); err != nil {
		return nil, err
	}
	if err := validateCurrencyCode("requestedAmount.currency", p.RequestedAmount.Currency); err != nil {
		return nil, err
	}
	if err := validateMaxLength("refundTicketMerchantExternalId", p.RefundTicketMerchantExternalID, 128); err != nil {
		return nil, err
	}
	out, warnings, err := customerPostAction[RefundTicketResult](ctx, s.http, "/v1/actions/refund-ticket/create-ticket", p)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// ResubmitRefundTicket resubmits a previously rejected refund ticket with
// updated details.
func (s *CustomerSession) ResubmitRefundTicket(ctx context.Context, p ResubmitRefundTicketParams) (*RefundTicketResult, error) {
	if err := validateShortID("ticketId", p.TicketID, "TKT"); err != nil {
		return nil, err
	}
	if err := validateShortID("paymentId", p.PaymentID, "PAY"); err != nil {
		return nil, err
	}
	if err := validateRequired("reason", p.Reason); err != nil {
		return nil, err
	}
	if err := validateAmountString("requestedAmount.amount", p.RequestedAmount.Amount); err != nil {
		return nil, err
	}
	if err := validateCurrencyCode("requestedAmount.currency", p.RequestedAmount.Currency); err != nil {
		return nil, err
	}
	out, warnings, err := customerPostAction[RefundTicketResult](ctx, s.http, "/v1/actions/refund-ticket/resubmit-ticket", p)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// CreatePlanChangeSession issues a plan change link for one of the customer's own
// subscriptions — the self-service half of a plan change. Send the customer to the
// returned CheckoutURL, which points at the change confirmation page.
//
// The platform applies three checks to a customer-issued link that it does not apply
// to a merchant-issued one, each answered with 403:
//
//   - Ownership: the subscription must belong to this session's customer (its buyer
//     identity and store must match the token), else "Subscription order does not
//     belong to this credential"
//   - Same group: the target plan must sit in the same product group as the current
//     plan, else "Target plan is not in the same product group as the current plan"
//   - Switch on: that group's rules.selfServicePlanChange must be true, else
//     "Self-service plan change is not enabled for this product group". Open it with
//     SubscriptionProductGroups.Update
//
// A merchant issuing the link with the API Key (Checkout.CreatePlanChangeSession) is
// subject to none of the three and may switch a subscription to any plan, group or not.
//
// The API-Key-only fields are absent from CustomerPlanChangeParams by construction —
// the platform would drop them here without saying so. Like every call on this
// session it carries no idempotency key, so a retry after a timeout can issue a
// second session rather than returning the first.
//
// Example:
//
//	session, err := customer.CreatePlanChangeSession(ctx, pancake.CustomerPlanChangeParams{
//	    OriginOrderID: "ORD_...",
//	    ProductID:     "PROD_...",
//	    Currency:      "USD",
//	})
func (s *CustomerSession) CreatePlanChangeSession(ctx context.Context, p CustomerPlanChangeParams) (*CheckoutSessionResult, error) {
	if err := validateShortID("originOrderId", p.OriginOrderID, "ORD"); err != nil {
		return nil, err
	}
	if err := validateShortID("productId", p.ProductID, "PROD"); err != nil {
		return nil, err
	}
	if err := validateCurrencyCode("currency", p.Currency); err != nil {
		return nil, err
	}
	out, warnings, err := customerPostAction[CheckoutSessionResult](ctx, s.http, "/v1/actions/checkout/create-session", p)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// CustomerGraphQLResource runs customer-scoped GraphQL queries.
type CustomerGraphQLResource struct {
	http *customerHTTPClient
}

// Query executes a GraphQL query scoped to the customer's data.
func (r *CustomerGraphQLResource) Query(ctx context.Context, p GraphQLParams) (*GraphQLResponse, error) {
	if err := validateRequired("query", p.Query); err != nil {
		return nil, err
	}
	_, env, err := r.http.post(ctx, "/v1/graphql", p)
	if err != nil {
		return nil, err
	}
	return &GraphQLResponse{Data: env.Data, Errors: env.Errors, Warnings: env.Warnings}, nil
}

// BuyerSession is an alias for [CustomerSession].
//
// Deprecated: Use CustomerSession instead.
type BuyerSession = CustomerSession

// BuyerGraphQLResource is an alias for [CustomerGraphQLResource].
//
// Deprecated: Use CustomerGraphQLResource instead.
type BuyerGraphQLResource = CustomerGraphQLResource
