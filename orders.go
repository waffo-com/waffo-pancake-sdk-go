package pancake

import "context"

// OrdersResource manages orders. Currently only subscription cancellation is
// exposed; one-time order management is performed by customers via
// CustomerSession.
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
	out, warnings, err := postAction[CancelSubscriptionResult](ctx, r.http, "/v1/actions/subscription-order/cancel-order", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}
