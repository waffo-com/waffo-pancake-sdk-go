package pancake

import "context"

// CustomerSession exposes the customer self-service API surface backed by a
// session token. Construct via [Client.Customer]. All HTTP methods use Bearer
// authentication.
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
//   - active   -> canceling, effective at the end of the current period
//   - past_due -> canceling, effective immediately
//
// In both canceling cases the PSP cancellation is confirmed asynchronously.
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
// "canceling" state. A subscription that was canceled while a renewal payment
// was outstanding cannot be reactivated and returns 400.
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
