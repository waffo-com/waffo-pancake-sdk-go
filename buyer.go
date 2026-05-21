package pancake

import "context"

// BuyerSession exposes the buyer self-service API surface backed by a session
// token. Construct via [Client.Buyer]. All HTTP methods use Bearer
// authentication.
type BuyerSession struct {
	// GraphQL runs buyer-scoped GraphQL queries.
	GraphQL *BuyerGraphQLResource

	http *buyerHTTPClient
}

func newBuyerSession(h *buyerHTTPClient) *BuyerSession {
	return &BuyerSession{
		http:    h,
		GraphQL: &BuyerGraphQLResource{http: h},
	}
}

// CancelSubscription cancels a buyer's subscription order.
//
//   - pending -> canceled
//   - active  -> canceling (the PSP cancellation is confirmed asynchronously)
func (s *BuyerSession) CancelSubscription(ctx context.Context, p CancelSubscriptionParams) (*CancelSubscriptionResult, error) {
	if err := validateShortID("orderId", p.OrderID, "ORD"); err != nil {
		return nil, err
	}
	out, warnings, err := buyerPostAction[CancelSubscriptionResult](ctx, s.http, "/v1/actions/subscription-order/cancel-order", p)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// CancelOnetimeOrder cancels a one-time order whose payment is still pending.
func (s *BuyerSession) CancelOnetimeOrder(ctx context.Context, p CancelOnetimeOrderParams) (*CancelOnetimeOrderResult, error) {
	if err := validateShortID("orderId", p.OrderID, "ORD"); err != nil {
		return nil, err
	}
	out, warnings, err := buyerPostAction[CancelOnetimeOrderResult](ctx, s.http, "/v1/actions/onetime-order/cancel-order", p)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// ReactivateSubscription reactivates a subscription currently in the
// "canceling" state.
func (s *BuyerSession) ReactivateSubscription(ctx context.Context, p ReactivateSubscriptionParams) (*ReactivateSubscriptionResult, error) {
	if err := validateShortID("orderId", p.OrderID, "ORD"); err != nil {
		return nil, err
	}
	out, warnings, err := buyerPostAction[ReactivateSubscriptionResult](ctx, s.http, "/v1/actions/subscription-order/reactivate-order", p)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// CreateRefundTicket submits a refund request for a payment.
func (s *BuyerSession) CreateRefundTicket(ctx context.Context, p CreateRefundTicketParams) (*RefundTicketResult, error) {
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
	out, warnings, err := buyerPostAction[RefundTicketResult](ctx, s.http, "/v1/actions/refund-ticket/create-ticket", p)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// ResubmitRefundTicket resubmits a previously rejected refund ticket with
// updated details.
func (s *BuyerSession) ResubmitRefundTicket(ctx context.Context, p ResubmitRefundTicketParams) (*RefundTicketResult, error) {
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
	out, warnings, err := buyerPostAction[RefundTicketResult](ctx, s.http, "/v1/actions/refund-ticket/resubmit-ticket", p)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// BuyerGraphQLResource runs buyer-scoped GraphQL queries.
type BuyerGraphQLResource struct {
	http *buyerHTTPClient
}

// Query executes a GraphQL query scoped to the buyer's data.
func (r *BuyerGraphQLResource) Query(ctx context.Context, p GraphQLParams) (*GraphQLResponse, error) {
	if err := validateRequired("query", p.Query); err != nil {
		return nil, err
	}
	_, env, err := r.http.post(ctx, "/v1/graphql", p)
	if err != nil {
		return nil, err
	}
	return &GraphQLResponse{Data: env.Data, Errors: env.Errors, Warnings: env.Warnings}, nil
}
