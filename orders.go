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
//   - pending  -> canceled (immediate; no PSP call)
//   - active   -> canceling (PSP cancel dispatched for the end of the current
//     billing period; the subscription stays usable until then)
//   - past_due -> canceling (PSP cancel dispatched immediately; the billing
//     period has already lapsed, so there is nothing left to use)
//
// In both canceling cases the terminal "canceled" status is written when the
// PSP cancellation webhook arrives, not by this call. A cancellation requested
// while a charge is unpaid — which is always the case for past_due — cannot be
// undone by ReactivateSubscription.
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
