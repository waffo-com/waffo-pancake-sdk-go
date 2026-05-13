package pancake

import "context"

// OrdersResource manages orders. Currently only subscription cancellation is
// exposed; one-time order management is performed by buyers via BuyerSession.
type OrdersResource struct {
	http *httpClient
}

// CancelSubscription cancels a subscription order.
//
//   - pending      -> canceled (immediate)
//   - active/past_due -> canceling (PSP cancel; webhook updates the status)
//
// Example:
//
//	res, err := client.Orders.CancelSubscription(ctx, pancake.CancelSubscriptionParams{
//	    OrderID: "ORD_...",
//	})
func (r *OrdersResource) CancelSubscription(ctx context.Context, p CancelSubscriptionParams) (*CancelSubscriptionResult, error) {
	if err := validateShortID("orderId", p.OrderID, "ORD"); err != nil {
		return nil, err
	}
	var out CancelSubscriptionResult
	if err := r.http.post(ctx, "/v1/actions/subscription-order/cancel-order", p, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
