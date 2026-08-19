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
//   - pending  -> canceled (immediate, no PSP call)
//   - active   -> canceling; the PSP cancel is scheduled for the end of the current
//     period, so access continues until then
//   - past_due -> canceling; the PSP cancel is dispatched immediately, since a
//     failed renewal leaves no paid period to preserve
//
// Both active and past_due settle to canceled only once the PSP cancellation
// webhook is received. A past_due cancellation emits no subscription.canceling
// event and cannot be reactivated.
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
