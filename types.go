package pancake

import "encoding/json"

// -----------------------------------------------------------------------------
// Per-call request options
// -----------------------------------------------------------------------------

// RequestOption customizes a single API call. Pass any number of them as the
// trailing arguments of a write method; passing none is the normal case.
//
// Variadic rather than a trailing *Options pointer (the shape [WebhooksResource.Verify]
// uses) so that adding it to every write method keeps existing call sites compiling.
type RequestOption func(*requestOptions)

// WithIdempotencyKey sends key as X-Idempotency-Key on this call.
//
// Nothing is sent when this option is absent, and the SDK never derives a key —
// so by default a write is not deduplicated and a retry executes it again.
//
// Platform semantics once a key is sent:
//
//   - The first request executes and its 2xx response is cached for 24 hours
//   - A repeat of the same key returns that cached response without re-executing
//   - A repeat while the original is still in flight returns 409
//   - A non-2xx original does not occupy the key; the same key can be retried
//
// The key is the whole cache identity of the request, so uniqueness is the
// caller's to guarantee: at most 256 characters of letters, numbers, hyphens and
// underscores, distinct per logical operation. A malformed key is rejected by the
// gateway with a 400.
func WithIdempotencyKey(key string) RequestOption {
	return func(o *requestOptions) { o.idempotencyKey = key }
}

// requestOptions is the resolved form of the RequestOption list.
type requestOptions struct {
	idempotencyKey string
}

// newRequestOptions applies opts in order and returns the result.
func newRequestOptions(opts []RequestOption) requestOptions {
	var resolved requestOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&resolved)
		}
	}
	return resolved
}

// -----------------------------------------------------------------------------
// Auth
// -----------------------------------------------------------------------------

// IssueSessionTokenParams names the customer for whom a session token should
// be minted. Provide either StoreID or ProductID — when only ProductID is
// given the server derives the store from the product.
type IssueSessionTokenParams struct {
	// BuyerIdentity is encoded into the JWT payload for merchant-side
	// customer identification. Accepts an email or any merchant-provided identifier
	// string. To pre-fill the checkout page's email input use BuyerEmail on
	// Checkout.Authenticated.Create instead.
	BuyerIdentity string  `json:"buyerIdentity"`
	StoreID       *string `json:"storeId,omitempty"`
	ProductID     *string `json:"productId,omitempty"`
}

// SessionToken is the issued JWT plus its absolute expiration timestamp.
type SessionToken struct {
	Token     string   `json:"token"`
	ExpiresAt string   `json:"expiresAt"`
	Warnings  []Notice `json:"warnings,omitempty"`
}

// -----------------------------------------------------------------------------
// Store
// -----------------------------------------------------------------------------

// StoreWebhook is a configured webhook endpoint as stored in store_webhooks.
type StoreWebhook struct {
	ID        string             `json:"id"`
	StoreID   string             `json:"storeId"`
	Channel   WebhookChannel     `json:"channel"`
	URL       string             `json:"url"`
	Events    []WebhookEventType `json:"events"`
	TestMode  bool               `json:"testMode"`
	Secret    *string            `json:"secret"`
	CreatedAt string             `json:"createdAt"`
	UpdatedAt string             `json:"updatedAt"`
}

// AddWebhookParams is the input to Webhooks.Add.
type AddWebhookParams struct {
	StoreID  string             `json:"storeId"`
	Channel  WebhookChannel     `json:"channel"`
	URL      string             `json:"url"`
	Events   []WebhookEventType `json:"events"`
	TestMode bool               `json:"testMode"`
	Secret   *string            `json:"secret,omitempty"`
}

// UpdateWebhookParams is the input to Webhooks.Update. Channel and TestMode
// are immutable; remove and re-add the webhook to change them.
type UpdateWebhookParams struct {
	ID     string             `json:"id"`
	URL    *string            `json:"url,omitempty"`
	Events []WebhookEventType `json:"events,omitempty"`
	Secret *Nullable[string]  `json:"secret,omitempty"`
}

// RemoveWebhookParams is the input to Webhooks.Remove.
type RemoveWebhookParams struct {
	ID string `json:"id"`
}

// NotificationSettings holds the merchant's email and dashboard notification
// preferences. All fields are optional on input (omit to keep server-side
// value); the response always carries the full set.
//
// Merchant-writable are all the Notify* toggles plus EmailUpcomingCharge, the
// buyer-facing renewal reminder — switching that one off silences it for every
// billing period in the store, yearly plans included.
//
// Every other Email* toggle is managed by the PANCAKE platform (admin-only via
// DB): the merchant update-store endpoint silently drops them and names them in
// the response's warnings.
type NotificationSettings struct {
	EmailOrderConfirmation        *bool `json:"emailOrderConfirmation,omitempty"`
	EmailSubscriptionConfirmation *bool `json:"emailSubscriptionConfirmation,omitempty"`
	EmailSubscriptionCycled       *bool `json:"emailSubscriptionCycled,omitempty"`
	EmailSubscriptionCanceled     *bool `json:"emailSubscriptionCanceled,omitempty"`
	EmailSubscriptionRevoked      *bool `json:"emailSubscriptionRevoked,omitempty"`
	EmailSubscriptionPastDue      *bool `json:"emailSubscriptionPastDue,omitempty"`
	EmailTrialStarted             *bool `json:"emailTrialStarted,omitempty"`
	EmailTrialEnding              *bool `json:"emailTrialEnding,omitempty"`
	// EmailUpcomingCharge is the renewal reminder sent to the buyer before a
	// subscription is charged. Unlike the other Email* toggles it is
	// merchant-writable.
	EmailUpcomingCharge *bool `json:"emailUpcomingCharge,omitempty"`
	// EmailSubscriptionPlanChanged is the single toggle shared by the three
	// plan-change customer emails (scheduled / failed / applied).
	EmailSubscriptionPlanChanged *bool `json:"emailSubscriptionPlanChanged,omitempty"`
	// EmailRefundSucceeded is the customer email sent when a refund on the
	// buyer's payment completes.
	EmailRefundSucceeded          *bool `json:"emailRefundSucceeded,omitempty"`
	NotifyNewOrders               *bool `json:"notifyNewOrders,omitempty"`
	NotifyNewSubscriptions        *bool `json:"notifyNewSubscriptions,omitempty"`
	NotifySubscriptionCanceled    *bool `json:"notifySubscriptionCanceled,omitempty"`
	NotifySubscriptionEnded       *bool `json:"notifySubscriptionEnded,omitempty"`
	NotifySubscriptionPastDue     *bool `json:"notifySubscriptionPastDue,omitempty"`
	NotifySubscriptionRenewed     *bool `json:"notifySubscriptionRenewed,omitempty"`
	NotifySubscriptionUncanceled  *bool `json:"notifySubscriptionUncanceled,omitempty"`
	NotifySubscriptionPlanChanged *bool `json:"notifySubscriptionPlanChanged,omitempty"`
	NotifyChargeback              *bool `json:"notifyChargeback,omitempty"`
	// NotifyRefundSucceeded is the merchant notification sent when a refund on
	// a payment completes.
	NotifyRefundSucceeded *bool `json:"notifyRefundSucceeded,omitempty"`
	NotifyPayoutCompleted *bool `json:"notifyPayoutCompleted,omitempty"`
	NotifyPayoutFailed    *bool `json:"notifyPayoutFailed,omitempty"`
}

// CheckoutThemeSettings holds checkout page styling for a single theme.
type CheckoutThemeSettings struct {
	CheckoutLogo            *string `json:"checkoutLogo"`
	CheckoutColorPrimary    string  `json:"checkoutColorPrimary"`
	CheckoutColorBackground string  `json:"checkoutColorBackground"`
	CheckoutColorCard       string  `json:"checkoutColorCard"`
	CheckoutColorText       string  `json:"checkoutColorText"`
	CheckoutBorderRadius    string  `json:"checkoutBorderRadius"`
}

// CheckoutSettings holds light and dark theme settings for the checkout page.
type CheckoutSettings struct {
	DefaultDarkMode bool                  `json:"defaultDarkMode"`
	Light           CheckoutThemeSettings `json:"light"`
	Dark            CheckoutThemeSettings `json:"dark"`
}

// Store is the store entity returned by Stores.Create / Update / Delete.
type Store struct {
	ID                   string                `json:"id"`
	Name                 string                `json:"name"`
	Status               EntityStatus          `json:"status"`
	Logo                 *string               `json:"logo"`
	SupportEmail         *string               `json:"supportEmail"`
	Website              *string               `json:"website"`
	Slug                 *string               `json:"slug"`
	ProdEnabled          bool                  `json:"prodEnabled"`
	NotificationSettings *NotificationSettings `json:"notificationSettings"`
	CheckoutSettings     *CheckoutSettings     `json:"checkoutSettings"`
	DeletedAt            *string               `json:"deletedAt"`
	CreatedAt            string                `json:"createdAt"`
	UpdatedAt            string                `json:"updatedAt"`
}

// CreateStoreParams is the input to Stores.Create.
type CreateStoreParams struct {
	Name string `json:"name"`
}

// UpdateStoreParams is the input to Stores.Update. Settings objects accept
// partial updates — omitted sub-fields keep their existing values, an explicit
// null clears the whole group.
//
// SupportEmail and Website are not writable here. They are derived from
// ownership verification and are set only by the flows that prove it: email
// code binding and domain verification, or KYB approval. Read them back from
// Store.
type UpdateStoreParams struct {
	ID                   string                          `json:"id"`
	Name                 *string                         `json:"name,omitempty"`
	Status               *EntityStatus                   `json:"status,omitempty"`
	Logo                 *Nullable[string]               `json:"logo,omitempty"`
	NotificationSettings *Nullable[NotificationSettings] `json:"notificationSettings,omitempty"`
	CheckoutSettings     *Nullable[CheckoutSettings]     `json:"checkoutSettings,omitempty"`
}

// DeleteStoreParams is the input to Stores.Delete (soft delete).
type DeleteStoreParams struct {
	ID string `json:"id"`
}

// CreateStoreResult wraps the response of Stores.Create / Update / Delete.
type CreateStoreResult struct {
	Store    Store    `json:"store"`
	Warnings []Notice `json:"warnings,omitempty"`
}

// UpdateStoreResult mirrors CreateStoreResult; aliased for ergonomics.
type UpdateStoreResult = CreateStoreResult

// DeleteStoreResult mirrors CreateStoreResult; aliased for ergonomics.
type DeleteStoreResult = CreateStoreResult

// -----------------------------------------------------------------------------
// Store Merchant (coming soon — endpoints return 501)
// -----------------------------------------------------------------------------

// AddMerchantParams is the input to StoreMerchants.Add.
type AddMerchantParams struct {
	StoreID string `json:"storeId"`
	Email   string `json:"email"`
	Role    string `json:"role"`
}

// AddMerchantResult is the response of StoreMerchants.Add.
type AddMerchantResult struct {
	StoreID    string   `json:"storeId"`
	MerchantID string   `json:"merchantId"`
	Email      string   `json:"email"`
	Role       string   `json:"role"`
	Status     string   `json:"status"`
	AddedAt    string   `json:"addedAt"`
	Warnings   []Notice `json:"warnings,omitempty"`
}

// RemoveMerchantParams is the input to StoreMerchants.Remove.
type RemoveMerchantParams struct {
	StoreID    string `json:"storeId"`
	MerchantID string `json:"merchantId"`
}

// RemoveMerchantResult is the response of StoreMerchants.Remove.
type RemoveMerchantResult struct {
	Message   string   `json:"message"`
	RemovedAt string   `json:"removedAt"`
	Warnings  []Notice `json:"warnings,omitempty"`
}

// UpdateRoleParams is the input to StoreMerchants.UpdateRole.
type UpdateRoleParams struct {
	StoreID    string `json:"storeId"`
	MerchantID string `json:"merchantId"`
	Role       string `json:"role"`
}

// UpdateRoleResult is the response of StoreMerchants.UpdateRole.
type UpdateRoleResult struct {
	StoreID    string   `json:"storeId"`
	MerchantID string   `json:"merchantId"`
	Role       string   `json:"role"`
	UpdatedAt  string   `json:"updatedAt"`
	Warnings   []Notice `json:"warnings,omitempty"`
}

// -----------------------------------------------------------------------------
// Product — shared types
// -----------------------------------------------------------------------------

// PriceInfo is the per-currency price expressed in display units (for example
// "9.99" for USD, "1000" for JPY).
//
// TrialAmount applies to subscription products only. Leave it nil for a free
// trial; when set it requires metadata.trialDays on the product and must be
// lower than Amount.
type PriceInfo struct {
	Amount      string      `json:"amount"`
	TaxCategory TaxCategory `json:"taxCategory"`
	TrialAmount *string     `json:"trialAmount,omitempty"`
}

// Prices is the multi-currency price map keyed by ISO 4217 currency code.
type Prices map[string]PriceInfo

// MediaItem is a single image or video attached to a product.
type MediaItem struct {
	Type      MediaType `json:"type"`
	URL       string    `json:"url"`
	Alt       *string   `json:"alt,omitempty"`
	Thumbnail *string   `json:"thumbnail,omitempty"`
}

// -----------------------------------------------------------------------------
// Onetime Product
// -----------------------------------------------------------------------------

// OnetimeProductDetail is the API shape of a one-time product.
type OnetimeProductDetail struct {
	ID          string               `json:"id"`
	StoreID     string               `json:"storeId"`
	Name        string               `json:"name"`
	Description *string              `json:"description"`
	Prices      Prices               `json:"prices"`
	Media       []MediaItem          `json:"media"`
	SuccessURL  *string              `json:"successUrl"`
	Metadata    map[string]any       `json:"metadata"`
	Status      ProductVersionStatus `json:"status"`
	CreatedAt   string               `json:"createdAt"`
	UpdatedAt   string               `json:"updatedAt"`
}

// CreateOnetimeProductParams is the input to OnetimeProducts.Create.
type CreateOnetimeProductParams struct {
	StoreID     string         `json:"storeId"`
	Name        string         `json:"name"`
	Prices      Prices         `json:"prices"`
	Description *string        `json:"description,omitempty"`
	Media       []MediaItem    `json:"media,omitempty"`
	SuccessURL  *string        `json:"successUrl,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// UpdateOnetimeProductParams is the input to OnetimeProducts.Update. The
// server creates a new immutable version and skips when nothing has changed.
type UpdateOnetimeProductParams struct {
	ID          string         `json:"id"`
	Name        *string        `json:"name,omitempty"`
	Prices      Prices         `json:"prices,omitempty"`
	Description *string        `json:"description,omitempty"`
	Media       []MediaItem    `json:"media,omitempty"`
	SuccessURL  *string        `json:"successUrl,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// PublishOnetimeProductParams promotes the test version to production.
type PublishOnetimeProductParams struct {
	ID string `json:"id"`
}

// UpdateOnetimeStatusParams flips between active and inactive.
type UpdateOnetimeStatusParams struct {
	ID     string               `json:"id"`
	Status ProductVersionStatus `json:"status"`
}

// OnetimeProductResult wraps the response of one-time product endpoints.
type OnetimeProductResult struct {
	Product  OnetimeProductDetail `json:"product"`
	Warnings []Notice             `json:"warnings,omitempty"`
}

// -----------------------------------------------------------------------------
// Subscription Product
// -----------------------------------------------------------------------------

// SubscriptionProductDetail is the API shape of a subscription product.
type SubscriptionProductDetail struct {
	ID            string               `json:"id"`
	StoreID       string               `json:"storeId"`
	Name          string               `json:"name"`
	Description   *string              `json:"description"`
	BillingPeriod BillingPeriod        `json:"billingPeriod"`
	Prices        Prices               `json:"prices"`
	Media         []MediaItem          `json:"media"`
	SuccessURL    *string              `json:"successUrl"`
	Metadata      map[string]any       `json:"metadata"`
	Status        ProductVersionStatus `json:"status"`
	CreatedAt     string               `json:"createdAt"`
	UpdatedAt     string               `json:"updatedAt"`
}

// CreateSubscriptionProductParams is the input to SubscriptionProducts.Create.
type CreateSubscriptionProductParams struct {
	StoreID       string         `json:"storeId"`
	Name          string         `json:"name"`
	BillingPeriod BillingPeriod  `json:"billingPeriod"`
	Prices        Prices         `json:"prices"`
	Description   *string        `json:"description,omitempty"`
	Media         []MediaItem    `json:"media,omitempty"`
	SuccessURL    *string        `json:"successUrl,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// UpdateSubscriptionProductParams is the input to SubscriptionProducts.Update.
type UpdateSubscriptionProductParams struct {
	ID            string         `json:"id"`
	Name          *string        `json:"name,omitempty"`
	BillingPeriod *BillingPeriod `json:"billingPeriod,omitempty"`
	Prices        Prices         `json:"prices,omitempty"`
	Description   *string        `json:"description,omitempty"`
	Media         []MediaItem    `json:"media,omitempty"`
	SuccessURL    *string        `json:"successUrl,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// PublishSubscriptionProductParams promotes the test version to production.
type PublishSubscriptionProductParams struct {
	ID string `json:"id"`
}

// UpdateSubscriptionStatusParams flips between active and inactive.
type UpdateSubscriptionStatusParams struct {
	ID     string               `json:"id"`
	Status ProductVersionStatus `json:"status"`
}

// SubscriptionProductResult wraps the response of subscription endpoints.
type SubscriptionProductResult struct {
	Product  SubscriptionProductDetail `json:"product"`
	Warnings []Notice                  `json:"warnings,omitempty"`
}

// -----------------------------------------------------------------------------
// Subscription Product Group
// -----------------------------------------------------------------------------

// GroupRules is the rules object as returned on a group entity. Every switch is
// always present — the platform reports an unset switch as false rather than
// omitting it.
type GroupRules struct {
	SharedTrial bool `json:"sharedTrial"`
	// SelfServicePlanChange controls whether customers may switch plans within this
	// group on their own from the customer portal. A customer-credential plan change
	// link is rejected with 403 while this is off; merchant-issued links are unaffected.
	SelfServicePlanChange bool `json:"selfServicePlanChange"`
}

// GroupRulesInput is the rules object accepted on group create and update. Every
// switch is optional and the platform merges them field by field: sending one
// switch leaves the other at its stored value (unlike ProductIDs, which is a full
// replacement). A switch left out on create is created off.
type GroupRulesInput struct {
	SharedTrial           *bool `json:"sharedTrial,omitempty"`
	SelfServicePlanChange *bool `json:"selfServicePlanChange,omitempty"`
}

// SubscriptionProductGroup is the API shape of a subscription product group.
type SubscriptionProductGroup struct {
	ID          string      `json:"id"`
	StoreID     string      `json:"storeId"`
	Name        string      `json:"name"`
	Description *string     `json:"description"`
	Rules       GroupRules  `json:"rules"`
	ProductIDs  []string    `json:"productIds"`
	Environment Environment `json:"environment"`
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
}

// CreateSubscriptionProductGroupParams is the input to
// SubscriptionProductGroups.Create.
type CreateSubscriptionProductGroupParams struct {
	StoreID     string           `json:"storeId"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	Rules       *GroupRulesInput `json:"rules,omitempty"`
	ProductIDs  []string         `json:"productIds,omitempty"`
}

// UpdateSubscriptionProductGroupParams is the input to
// SubscriptionProductGroups.Update. ProductIDs is a full replacement, not a
// merge; Rules is the opposite — it is merged switch by switch.
type UpdateSubscriptionProductGroupParams struct {
	ID          string           `json:"id"`
	Name        *string          `json:"name,omitempty"`
	Description *string          `json:"description,omitempty"`
	Rules       *GroupRulesInput `json:"rules,omitempty"`
	ProductIDs  []string         `json:"productIds,omitempty"`
}

// DeleteSubscriptionProductGroupParams is the input to
// SubscriptionProductGroups.Delete (hard delete).
type DeleteSubscriptionProductGroupParams struct {
	ID string `json:"id"`
}

// PublishSubscriptionProductGroupParams promotes a test group to production.
type PublishSubscriptionProductGroupParams struct {
	ID string `json:"id"`
}

// SubscriptionProductGroupResult wraps the response of group endpoints.
type SubscriptionProductGroupResult struct {
	Group    SubscriptionProductGroup `json:"group"`
	Warnings []Notice                 `json:"warnings,omitempty"`
}

// -----------------------------------------------------------------------------
// Order
// -----------------------------------------------------------------------------

// CancelSubscriptionParams is the input to Orders.CancelSubscription and to
// CustomerSession.CancelSubscription.
type CancelSubscriptionParams struct {
	OrderID string `json:"orderId"`
}

// CancelSubscriptionResult is the response of CancelSubscription.
type CancelSubscriptionResult struct {
	OrderID  string                  `json:"orderId"`
	Status   SubscriptionOrderStatus `json:"status"`
	Warnings []Notice                `json:"warnings,omitempty"`
}

// BillingDetail captures customer billing information for checkout.
type BillingDetail struct {
	Country      string  `json:"country"`
	IsBusiness   bool    `json:"isBusiness"`
	Postcode     *string `json:"postcode,omitempty"`
	State        *string `json:"state,omitempty"`
	BusinessName *string `json:"businessName,omitempty"`
	TaxID        *string `json:"taxId,omitempty"`
}

// PriceSnapshot overrides the product price for a single checkout session and is
// accepted with API Key authentication only.
//
// For subscription products it replaces the regular period price; the trial
// price comes from the product version locked into the session.
type PriceSnapshot struct {
	Amount      string      `json:"amount"`
	TaxCategory TaxCategory `json:"taxCategory"`
}

// CreateCheckoutSessionParams is the input to Checkout.CreateSession.
type CreateCheckoutSessionParams struct {
	ProductID     string         `json:"productId"`
	Currency      string         `json:"currency"`
	PriceSnapshot *PriceSnapshot `json:"priceSnapshot,omitempty"`
	WithTrial     *bool          `json:"withTrial,omitempty"`
	BuyerEmail    *string        `json:"buyerEmail,omitempty"`
	// BillingDetail couples the cashier to the order's billing country: it then offers only that
	// country's payment market and the customer cannot switch. The country that applies is the one on
	// the finished order, not the one sent here; a country outside the payment markets Waffo covers
	// applies no restriction. Leave nil to keep the cashier unrestricted.
	BillingDetail    *BillingDetail    `json:"billingDetail,omitempty"`
	SuccessURL       *string           `json:"successUrl,omitempty"`
	ExpiresInSeconds *int              `json:"expiresInSeconds,omitempty"`
	DarkMode         *bool             `json:"darkMode,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	// OrderMerchantExternalID is the order-side business identifier (max 128 chars); inherited by orders, payments, refunds.
	OrderMerchantExternalID *string `json:"orderMerchantExternalId,omitempty"`
	// Language is the default language of the hosted checkout page (IETF BCP 47).
	// The customer can switch it on the page; omit to let the provider infer.
	Language *CashierLanguage `json:"language,omitempty"`
	// IncludePaymentMethods is a whitelist — offer only these methods. Every value must be supported
	// by the product type x currency pair, otherwise the request is rejected. Mutually exclusive with
	// ExcludePaymentMethods; omit both to offer every method the currency supports.
	IncludePaymentMethods []PaymentMethod `json:"includePaymentMethods,omitempty"`
	// ExcludePaymentMethods is a blacklist — offer everything the currency supports except these.
	// Values the currency does not offer are ignored, so one blacklist can be reused across currencies.
	// Mutually exclusive with IncludePaymentMethods.
	ExcludePaymentMethods []PaymentMethod `json:"excludePaymentMethods,omitempty"`
}

// CheckoutSessionResult is the response of Checkout.CreateSession and
// Checkout.Anonymous.Create.
type CheckoutSessionResult struct {
	SessionID   string   `json:"sessionId"`
	CheckoutURL string   `json:"checkoutUrl"`
	ExpiresAt   string   `json:"expiresAt"`
	Warnings    []Notice `json:"warnings,omitempty"`
}

// CreatePlanChangeSessionParams is the input to Checkout.CreatePlanChangeSession.
//
// A plan change is the same create-session endpoint in a different mode, and
// OriginOrderID is what switches it — which is why it is a plain required field
// here instead of an optional one on CreateCheckoutSessionParams. The returned
// CheckoutURL points at the change confirmation page
// (…/store/{slug}/change/{sessionId}), where the customer confirms the change.
//
// BuyerEmail and BillingDetail have no counterpart here: in plan change mode the
// customer email, billing details and tax all come from the origin subscription.
type CreatePlanChangeSessionParams struct {
	// OriginOrderID is the subscription being changed (Short ID, ORD_xxx). Required —
	// its presence is what puts the request into plan change mode.
	OriginOrderID string `json:"originOrderId"`
	// ProductID is the target plan — a subscription product other than the current one.
	ProductID string `json:"productId"`
	Currency  string `json:"currency"`
	// ChangeTiming is when the new plan takes effect. Leave nil to let the platform
	// derive it from the change direction; the derived tier is not echoed back.
	ChangeTiming *ChangeTiming `json:"changeTiming,omitempty"`
	// ChangeAmount is what to actually charge for this change, as a display string,
	// tax inclusive (e.g. "12.00") — the "charge this much" form. Mutually exclusive
	// with ChangeCreditAmount: the same number means the opposite thing under each, so
	// sending both is rejected with a 400 rather than one being picked. Merchant
	// credentials only — any other credential has it silently dropped by the platform,
	// the same way PriceSnapshot is on a new purchase.
	ChangeAmount *string `json:"changeAmount,omitempty"`
	// ChangeCreditAmount is how much to credit against this change, as a display
	// string, tax inclusive (e.g. "8.00") — the "credit this much" form, same unit and
	// tax basis as ChangeAmount. Mutually exclusive with ChangeAmount (sending both is
	// a 400) and merchant credentials only.
	ChangeCreditAmount *string `json:"changeCreditAmount,omitempty"`
	// WithTrial is the trial toggle for the target plan. Three-state and identical to a
	// new purchase: true grants one, false withholds one, nil follows the product
	// group's rule. Merchant credentials only — silently dropped otherwise.
	WithTrial *bool `json:"withTrial,omitempty"`
	// PriceSnapshot overrides the target plan's price for this session.
	PriceSnapshot    *PriceSnapshot    `json:"priceSnapshot,omitempty"`
	SuccessURL       *string           `json:"successUrl,omitempty"`
	ExpiresInSeconds *int              `json:"expiresInSeconds,omitempty"`
	DarkMode         *bool             `json:"darkMode,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	// OrderMerchantExternalID is the order-side business identifier (max 128 chars);
	// inherited by orders, payments, refunds.
	OrderMerchantExternalID *string `json:"orderMerchantExternalId,omitempty"`
	// Language is the default language of the confirmation page (IETF BCP 47).
	Language *CashierLanguage `json:"language,omitempty"`
	// IncludePaymentMethods is a whitelist, mutually exclusive with ExcludePaymentMethods.
	IncludePaymentMethods []PaymentMethod `json:"includePaymentMethods,omitempty"`
	// ExcludePaymentMethods is a blacklist, mutually exclusive with IncludePaymentMethods.
	ExcludePaymentMethods []PaymentMethod `json:"excludePaymentMethods,omitempty"`
}

// -----------------------------------------------------------------------------
// Customer self-service
// -----------------------------------------------------------------------------

// CancelOnetimeOrderParams is the input to CustomerSession.CancelOnetimeOrder.
type CancelOnetimeOrderParams struct {
	OrderID string `json:"orderId"`
}

// CancelOnetimeOrderResult is the response of CancelOnetimeOrder.
type CancelOnetimeOrderResult struct {
	OrderID  string   `json:"orderId"`
	Status   string   `json:"status"`
	Warnings []Notice `json:"warnings,omitempty"`
}

// ReactivateSubscriptionParams is the input to
// CustomerSession.ReactivateSubscription.
type ReactivateSubscriptionParams struct {
	OrderID string `json:"orderId"`
}

// ReactivateSubscriptionResult is the response of ReactivateSubscription.
type ReactivateSubscriptionResult struct {
	OrderID  string   `json:"orderId"`
	Status   string   `json:"status"`
	Warnings []Notice `json:"warnings,omitempty"`
}

// RequestedAmount specifies the amount and currency for a refund request.
type RequestedAmount struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// RefundTicketVersionData holds the per-submission data of a refund ticket.
type RefundTicketVersionData struct {
	Reason          string           `json:"reason"`
	RequestedAmount *RequestedAmount `json:"requestedAmount"`
}

// CreateRefundTicketParams is the input to CustomerSession.CreateRefundTicket.
type CreateRefundTicketParams struct {
	PaymentID       string          `json:"paymentId"`
	Reason          string          `json:"reason"`
	RequestedAmount RequestedAmount `json:"requestedAmount"`
	Metadata        map[string]any  `json:"metadata,omitempty"`
	// RefundTicketMerchantExternalID is the refund-ticket business identifier (max 128 chars); inherited by the executed refund on PSP success.
	RefundTicketMerchantExternalID *string `json:"refundTicketMerchantExternalId,omitempty"`
}

// ResubmitRefundTicketParams is the input to
// CustomerSession.ResubmitRefundTicket.
type ResubmitRefundTicketParams struct {
	TicketID        string          `json:"ticketId"`
	PaymentID       string          `json:"paymentId"`
	Reason          string          `json:"reason"`
	RequestedAmount RequestedAmount `json:"requestedAmount"`
}

// RefundTicket is the entity returned by refund ticket create / resubmit.
type RefundTicket struct {
	ID               string                   `json:"id"`
	Type             string                   `json:"type"`
	Status           string                   `json:"status"`
	SubjectID        string                   `json:"subjectId"`
	SubmitterID      string                   `json:"submitterId"`
	SubmitterType    string                   `json:"submitterType"`
	CurrentVersionID *string                  `json:"currentVersionId"`
	ReviewerID       *string                  `json:"reviewerId"`
	ReviewedAt       *string                  `json:"reviewedAt"`
	ReviewNote       *string                  `json:"reviewNote"`
	RejectReason     *string                  `json:"rejectReason"`
	ExecutedAt       *string                  `json:"executedAt"`
	Metadata         map[string]any           `json:"metadata"`
	VersionNumber    *int                     `json:"versionNumber"`
	VersionData      *RefundTicketVersionData `json:"versionData"`
	// RefundTicketMerchantExternalID is the refund-ticket business identifier (max 128 chars, immutable across resubmits).
	RefundTicketMerchantExternalID *string `json:"refundTicketMerchantExternalId"`
	CreatedAt                      string  `json:"createdAt"`
	UpdatedAt                      string  `json:"updatedAt"`
}

// RefundTicketResult wraps the refund ticket response envelope.
type RefundTicketResult struct {
	Ticket   RefundTicket `json:"ticket"`
	Warnings []Notice     `json:"warnings,omitempty"`
}

// -----------------------------------------------------------------------------
// Checkout — convenience wrappers
// -----------------------------------------------------------------------------

// AnonymousCheckoutParams is the input to Checkout.Anonymous.Create. The
// checkout form is left blank unless BuyerEmail or BillingDetail are supplied.
type AnonymousCheckoutParams = CreateCheckoutSessionParams

// AuthenticatedCheckoutParams is the input to Checkout.Authenticated.Create.
// It extends CreateCheckoutSessionParams with BuyerIdentity, which is routed
// to the issue-session-token endpoint while the remaining fields go to the
// create-session endpoint.
type AuthenticatedCheckoutParams struct {
	CreateCheckoutSessionParams
	// BuyerIdentity is encoded into the JWT for merchant-side customer
	// identification. Use BuyerEmail to pre-fill the checkout form's email
	// input; the two fields are independent.
	BuyerIdentity string `json:"-"`
}

// CustomerPlanChangeParams is the input to
// CustomerSession.CreatePlanChangeSession.
//
// Same endpoint as CreatePlanChangeSessionParams, reached with a customer session
// token instead of the merchant API Key. The credential is what narrows the field
// set: a customer-session request carries no merchant id, so the platform treats
// every API-Key-only field as absent and silently drops it — no error says so.
// Those fields are therefore left off this struct rather than accepted and ignored:
// ChangeAmount, ChangeCreditAmount, WithTrial, PriceSnapshot, ExpiresInSeconds,
// Metadata, OrderMerchantExternalID, IncludePaymentMethods and ExcludePaymentMethods.
// BuyerEmail and BillingDetail are absent for the same reason they are on the
// merchant struct — plan change mode takes both from the origin subscription.
type CustomerPlanChangeParams struct {
	// OriginOrderID is the customer's own subscription being changed (Short ID, ORD_xxx).
	OriginOrderID string `json:"originOrderId"`
	// ProductID is the target plan — must sit in the same product group as the current plan.
	ProductID string `json:"productId"`
	// Currency must match the origin subscription.
	Currency string `json:"currency"`
	// ChangeTiming is when the new plan takes effect. Leave nil to let the platform
	// derive it from the change direction.
	ChangeTiming *ChangeTiming `json:"changeTiming,omitempty"`
	SuccessURL   *string       `json:"successUrl,omitempty"`
	DarkMode     *bool         `json:"darkMode,omitempty"`
	// Language is the default language of the confirmation page (IETF BCP 47).
	Language *CashierLanguage `json:"language,omitempty"`
}

// AuthenticatedPlanChangeParams is the input to
// Checkout.Authenticated.CreatePlanChange. Same split as
// AuthenticatedCheckoutParams: BuyerIdentity is routed to the issue-session-token
// endpoint while the remaining fields go to the create-session endpoint.
type AuthenticatedPlanChangeParams struct {
	CreatePlanChangeSessionParams
	// BuyerIdentity is encoded into the JWT for merchant-side customer
	// identification, so the customer reaches the confirmation page signed in.
	BuyerIdentity string `json:"-"`
}

// AuthenticatedCheckoutResult is the response of Checkout.Authenticated.Create
// — session and token data merged into one struct.
type AuthenticatedCheckoutResult struct {
	SessionID      string   `json:"sessionId"`
	CheckoutURL    string   `json:"checkoutUrl"`
	ExpiresAt      string   `json:"expiresAt"`
	Token          string   `json:"token"`
	TokenExpiresAt string   `json:"tokenExpiresAt"`
	Warnings       []Notice `json:"warnings,omitempty"`
}

// -----------------------------------------------------------------------------
// GraphQL
// -----------------------------------------------------------------------------

// GraphQLParams is a GraphQL query string with optional variables.
type GraphQLParams struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// GraphQLErrorLocation marks a position in the query string for diagnostics.
type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// GraphQLError is the legacy name for {@link Notice}. Same shape (the unified
// Notice already includes Locations / Path for graphql-js errors). Kept as a
// type alias for backwards compatibility with existing imports.
//
// Deprecated: use Notice.
type GraphQLError = Notice

// GraphQLWarning is the legacy name for {@link Notice}. Kept as a type alias.
//
// Deprecated: use Notice.
type GraphQLWarning = Notice

// GraphQLResponse is the untyped GraphQL response — Data is left as raw bytes
// so callers can unmarshal into whichever struct they prefer.
type GraphQLResponse struct {
	Data     json.RawMessage `json:"data"`
	Errors   []Notice        `json:"errors,omitempty"`
	Warnings []Notice        `json:"warnings,omitempty"`
}

// envelope is the standard wire envelope { data, errors?, warnings? } produced
// by both REST writes and GraphQL queries (see handbook command-layer.md).
type envelope struct {
	Data     json.RawMessage `json:"data"`
	Errors   []Notice        `json:"errors,omitempty"`
	Warnings []Notice        `json:"warnings,omitempty"`
}

// TypedGraphQLResponse is the typed GraphQL response produced by
// [GraphQLQuery] / [CustomerGraphQLQuery].
type TypedGraphQLResponse[T any] struct {
	Data     T                `json:"data"`
	Errors   []GraphQLError   `json:"errors,omitempty"`
	Warnings []GraphQLWarning `json:"warnings,omitempty"`
}

// -----------------------------------------------------------------------------
// Webhook
// -----------------------------------------------------------------------------

// WebhookAmountBreakdown is the amount breakdown of the object a webhook event
// refers to: the list price behind a charge, the original payment behind a refund,
// or the plan price behind a subscription status change.
//
// Money fields are display strings, already converted from minor units ("29.00" for
// USD, "4500" for JPY). The block never uses the name Amount — that name belongs to
// the event's own figure at the top level.
type WebhookAmountBreakdown struct {
	// Total is the amount after tax.
	Total string `json:"total"`
	// Subtotal is the amount before tax; nil when the snapshot carries no subtotal.
	Subtotal *string `json:"subtotal,omitempty"`
	// TaxAmount is the tax portion.
	TaxAmount string `json:"taxAmount"`
	// TaxRate is the tax rate as a percentage number (10 means 10%).
	TaxRate *float64 `json:"taxRate,omitempty"`
	// TaxName is the tax name (for example "Consumption Tax").
	TaxName *string `json:"taxName,omitempty"`
}

// WebhookEventData is the canonical payload shape of a webhook event. Many
// fields are conditional on the event type — for example refund.* events
// populate the refund.* fields while leaving the subscription.* fields nil.
type WebhookEventData struct {
	OrderID                       string  `json:"orderId"`
	OrderStatus                   *string `json:"orderStatus,omitempty"`
	BuyerEmail                    string  `json:"buyerEmail"`
	MerchantProvidedBuyerIdentity *string `json:"merchantProvidedBuyerIdentity,omitempty"`
	// OrderMerchantExternalID is the order business identifier; present on order/payment + refund events (inherited from order).
	OrderMerchantExternalID *string `json:"orderMerchantExternalId,omitempty"`
	// RefundTicketMerchantExternalID is the refund-ticket business identifier; only on refund.* events.
	RefundTicketMerchantExternalID *string           `json:"refundTicketMerchantExternalId,omitempty"`
	Currency                       string            `json:"currency"`
	BillingDetail                  map[string]any    `json:"billingDetail,omitempty"`
	OrderMetadata                  map[string]string `json:"orderMetadata,omitempty"`

	// ChargedAmount is the amount the payment channel actually charged, as a display
	// string. Present only on payment events (order.completed,
	// subscription.payment_succeeded), and only when the channel reported an amount —
	// nil otherwise, never zero-filled. It differs from ListPrice when a prorated
	// credit or a zero-amount card check applies.
	ChargedAmount *string `json:"chargedAmount,omitempty"`
	// RefundedAmount is the amount the payment channel actually refunded, as a display
	// string. Present only on refund events (refund.succeeded, refund.failed), and only
	// when the channel reported an amount — nil otherwise, never zero-filled.
	RefundedAmount *string `json:"refundedAmount,omitempty"`

	// Deprecated: the same name carries a different subject per event family. Use
	// ChargedAmount on payment events, RefundedAmount on refund events, and
	// PlanPrice.Total on subscription status events. Removal is no earlier than 12
	// months away and ships with the next major version. On payment events this field
	// reports the amount actually charged; see the webhook API reference for the
	// effective date of that change.
	Amount string `json:"amount"`
	// Deprecated: use ListPrice.TaxAmount on payment events, OriginalPayment.TaxAmount
	// on refund events, and PlanPrice.TaxAmount on subscription status events. Removal
	// is no earlier than 12 months away and ships with the next major version.
	TaxAmount string `json:"taxAmount"`
	// Deprecated: use ListPrice.TaxRate on payment events, OriginalPayment.TaxRate on
	// refund events, and PlanPrice.TaxRate on subscription status events. Removal is no
	// earlier than 12 months away and ships with the next major version.
	TaxRate *float64 `json:"taxRate,omitempty"`
	// Deprecated: use ListPrice.TaxName on payment events, OriginalPayment.TaxName on
	// refund events, and PlanPrice.TaxName on subscription status events. Removal is no
	// earlier than 12 months away and ships with the next major version.
	TaxName *string `json:"taxName,omitempty"`
	// Deprecated: use ListPrice.Subtotal on payment events, OriginalPayment.Subtotal on
	// refund events, and PlanPrice.Subtotal on subscription status events. Removal is no
	// earlier than 12 months away and ships with the next major version.
	Subtotal *string `json:"subtotal,omitempty"`
	// Deprecated: use ListPrice.Total on payment events, OriginalPayment.Total on refund
	// events, and PlanPrice.Total on subscription status events. Removal is no earlier
	// than 12 months away and ships with the next major version.
	Total *string `json:"total,omitempty"`

	// ListPrice is the list price snapshot taken when the order was placed. Present
	// only on payment events; nil when the order carries no amount snapshot.
	ListPrice *WebhookAmountBreakdown `json:"listPrice,omitempty"`
	// OriginalPayment is the payment being refunded, as it was originally charged.
	// Present only on refund events; nil when that payment carries no amount snapshot.
	OriginalPayment *WebhookAmountBreakdown `json:"originalPayment,omitempty"`
	// PlanPrice is the subscription's list price for the current phase. Present only on
	// subscription status events (every subscription.* event except
	// subscription.payment_succeeded); nil when the subscription carries no price snapshot.
	PlanPrice *WebhookAmountBreakdown `json:"planPrice,omitempty"`

	ProductName        string            `json:"productName"`
	ProductDescription *string           `json:"productDescription,omitempty"`
	ProductMetadata    map[string]string `json:"productMetadata,omitempty"`

	PaymentID            *string `json:"paymentId,omitempty"`
	PaymentStatus        *string `json:"paymentStatus,omitempty"`
	PaymentMethod        *string `json:"paymentMethod,omitempty"`
	PaymentLast4         *string `json:"paymentLast4,omitempty"`
	PaymentFailureReason *string `json:"paymentFailureReason,omitempty"`
	PaymentDate          *string `json:"paymentDate,omitempty"`

	// PeriodNumber is the billing period this event refers to, as reported by the payment channel.
	PeriodNumber *int `json:"periodNumber,omitempty"`

	BillingPeriod      *string `json:"billingPeriod,omitempty"`
	CurrentPeriodStart *string `json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   *string `json:"currentPeriodEnd,omitempty"`
	CanceledAt         *string `json:"canceledAt,omitempty"`

	RefundStatus    *string `json:"refundStatus,omitempty"`
	RefundReason    *string `json:"refundReason,omitempty"`
	RefundCreatedAt *string `json:"refundCreatedAt,omitempty"`
}

// WebhookEvent is the verified envelope returned by VerifyWebhook. Data is
// kept as raw bytes so callers may unmarshal it into either WebhookEventData
// or a custom struct via VerifyWebhookTyped.
type WebhookEvent struct {
	ID        string          `json:"id"`
	Timestamp string          `json:"timestamp"`
	EventType string          `json:"eventType"`
	EventID   string          `json:"eventId"`
	StoreID   string          `json:"storeId"`
	StoreName string          `json:"storeName"`
	Mode      Environment     `json:"mode"`
	Data      json.RawMessage `json:"data"`
}

// TypedWebhookEvent is the typed envelope produced by VerifyWebhookTyped.
type TypedWebhookEvent[T any] struct {
	ID        string      `json:"id"`
	Timestamp string      `json:"timestamp"`
	EventType string      `json:"eventType"`
	EventID   string      `json:"eventId"`
	StoreID   string      `json:"storeId"`
	StoreName string      `json:"storeName"`
	Mode      Environment `json:"mode"`
	Data      T           `json:"data"`
}

// WebhookPublicKeys configures the public key(s) used to verify webhook
// signatures. A single string is shared between test and prod; the struct
// variant is used to set keys per environment.
type WebhookPublicKeys struct {
	// Shared is the single key used for both environments. Mutually exclusive
	// with Test/Prod.
	Shared string
	Test   string
	Prod   string
}

// IsZero reports whether no keys are configured.
func (k WebhookPublicKeys) IsZero() bool {
	return k.Shared == "" && k.Test == "" && k.Prod == ""
}

// VerifyWebhookOptions tunes [VerifyWebhook]. The zero value is valid.
type VerifyWebhookOptions struct {
	// Environment forces verification against the named environment's key.
	// When zero, both prod and test keys are tried (prod first).
	Environment Environment
	// ToleranceMS is how far in the past a signature timestamp may be, in
	// milliseconds. Set to a negative value to disable timestamp checking
	// entirely (which also disables FutureToleranceMS). Zero selects
	// DefaultWebhookToleranceMS, which covers the full delivery retry schedule:
	// the timestamp is stamped before the first attempt and retries reuse it.
	ToleranceMS int64
	// FutureToleranceMS is how far in the future a signature timestamp may be,
	// in milliseconds. Only clock skew on the receiving server puts a timestamp
	// ahead of now, so this stays tight. Zero selects
	// DefaultWebhookFutureToleranceMS. Ignored when ToleranceMS is negative.
	FutureToleranceMS int64
	// PublicKey, when non-empty, overrides all resolution chains and is used
	// directly for verification.
	PublicKey string
	// PublicKeys injects config-level keys into the resolution chain.
	// When using Webhooks.Verify this is set automatically from the client
	// config; standalone callers can pass it directly.
	PublicKeys *WebhookPublicKeys
}

// -----------------------------------------------------------------------------
// Content Safety
// -----------------------------------------------------------------------------

// ScanPromptParams is the input to ContentSafety.ScanPrompt.
type ScanPromptParams struct {
	Prompt   string           `json:"prompt"`
	Locale   string           `json:"locale,omitempty"`
	Semantic ScanSemanticMode `json:"semantic,omitempty"`
}

// ScanResult is the redacted verdict of ContentSafety.ScanPrompt — no scores,
// thresholds, or keyword text. Continue to generation only when Action is
// [ScanActionAllow].
type ScanResult struct {
	Action            ScanAction           `json:"action"`
	ReasonCode        ScanReasonCode       `json:"reasonCode"`
	MatchedCategories []ScanPolicyCategory `json:"matchedCategories"`
	RequestID         string               `json:"requestId"`
	SemanticStatus    ScanSemanticStatus   `json:"semanticStatus"`
	Warnings          []Notice             `json:"warnings,omitempty"`
}
