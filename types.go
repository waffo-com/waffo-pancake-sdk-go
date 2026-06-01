package pancake

import "encoding/json"

// -----------------------------------------------------------------------------
// Internal HTTP options
// -----------------------------------------------------------------------------

// postOptions tunes a single signed POST request.
type postOptions struct {
	// IdempotencyWindow rotates the deterministic idempotency key by time
	// window (in seconds). Used by checkout session creation.
	IdempotencyWindow int
	// NoIdempotency omits the X-Idempotency-Key header entirely. Set for
	// read-only queries (e.g. GraphQL) so the gateway's 24h idempotency
	// cache does not serve stale data on identical repeat queries.
	NoIdempotency bool
}

// -----------------------------------------------------------------------------
// Auth
// -----------------------------------------------------------------------------

// IssueSessionTokenParams names the buyer for whom a session token should be
// minted. Provide either StoreID or ProductID — when only ProductID is given
// the server derives the store from the product.
type IssueSessionTokenParams struct {
	// BuyerIdentity is encoded into the JWT payload for merchant-side buyer
	// identification. Accepts an email or any merchant-provided identifier
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
// Email* toggles (Email…) are managed by the PANCAKE platform (admin-only via
// DB) and are silently dropped if passed to the merchant update-store endpoint.
// Only Notify* toggles are merchant-writable.
type NotificationSettings struct {
	EmailOrderConfirmation        *bool `json:"emailOrderConfirmation,omitempty"`
	EmailSubscriptionConfirmation *bool `json:"emailSubscriptionConfirmation,omitempty"`
	EmailSubscriptionCycled       *bool `json:"emailSubscriptionCycled,omitempty"`
	EmailSubscriptionCanceled     *bool `json:"emailSubscriptionCanceled,omitempty"`
	EmailSubscriptionRevoked      *bool `json:"emailSubscriptionRevoked,omitempty"`
	EmailSubscriptionPastDue      *bool `json:"emailSubscriptionPastDue,omitempty"`
	EmailTrialStarted             *bool `json:"emailTrialStarted,omitempty"`
	EmailTrialEnding              *bool `json:"emailTrialEnding,omitempty"`
	NotifyNewOrders               *bool `json:"notifyNewOrders,omitempty"`
	NotifyNewSubscriptions        *bool `json:"notifyNewSubscriptions,omitempty"`
	NotifySubscriptionCanceled    *bool `json:"notifySubscriptionCanceled,omitempty"`
	NotifySubscriptionEnded       *bool `json:"notifySubscriptionEnded,omitempty"`
	NotifySubscriptionPastDue     *bool `json:"notifySubscriptionPastDue,omitempty"`
	NotifySubscriptionRenewed     *bool `json:"notifySubscriptionRenewed,omitempty"`
	NotifySubscriptionUncanceled  *bool `json:"notifySubscriptionUncanceled,omitempty"`
	NotifySubscriptionUpdated     *bool `json:"notifySubscriptionUpdated,omitempty"`
	NotifyChargeback              *bool `json:"notifyChargeback,omitempty"`
	NotifyPayoutCompleted         *bool `json:"notifyPayoutCompleted,omitempty"`
	NotifyPayoutFailed            *bool `json:"notifyPayoutFailed,omitempty"`
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
type UpdateStoreParams struct {
	ID                   string                          `json:"id"`
	Name                 *string                         `json:"name,omitempty"`
	Status               *EntityStatus                   `json:"status,omitempty"`
	Logo                 *Nullable[string]               `json:"logo,omitempty"`
	SupportEmail         *Nullable[string]               `json:"supportEmail,omitempty"`
	Website              *Nullable[string]               `json:"website,omitempty"`
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
type PriceInfo struct {
	Amount      string      `json:"amount"`
	TaxCategory TaxCategory `json:"taxCategory"`
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

// GroupRules controls cross-product behavior within a subscription group.
type GroupRules struct {
	SharedTrial bool `json:"sharedTrial"`
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
	StoreID     string      `json:"storeId"`
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	Rules       *GroupRules `json:"rules,omitempty"`
	ProductIDs  []string    `json:"productIds,omitempty"`
}

// UpdateSubscriptionProductGroupParams is the input to
// SubscriptionProductGroups.Update. ProductIDs is a full replacement, not a
// merge.
type UpdateSubscriptionProductGroupParams struct {
	ID          string      `json:"id"`
	Name        *string     `json:"name,omitempty"`
	Description *string     `json:"description,omitempty"`
	Rules       *GroupRules `json:"rules,omitempty"`
	ProductIDs  []string    `json:"productIds,omitempty"`
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
// BuyerSession.CancelSubscription.
type CancelSubscriptionParams struct {
	OrderID string `json:"orderId"`
}

// CancelSubscriptionResult is the response of CancelSubscription.
type CancelSubscriptionResult struct {
	OrderID  string                  `json:"orderId"`
	Status   SubscriptionOrderStatus `json:"status"`
	Warnings []Notice                `json:"warnings,omitempty"`
}

// BillingDetail captures buyer billing information for checkout.
type BillingDetail struct {
	Country      string  `json:"country"`
	IsBusiness   bool    `json:"isBusiness"`
	Postcode     *string `json:"postcode,omitempty"`
	State        *string `json:"state,omitempty"`
	BusinessName *string `json:"businessName,omitempty"`
	TaxID        *string `json:"taxId,omitempty"`
}

// CreateCheckoutSessionParams is the input to Checkout.CreateSession.
type CreateCheckoutSessionParams struct {
	ProductID        string            `json:"productId"`
	Currency         string            `json:"currency"`
	PriceSnapshot    *PriceInfo        `json:"priceSnapshot,omitempty"`
	WithTrial        *bool             `json:"withTrial,omitempty"`
	BuyerEmail       *string           `json:"buyerEmail,omitempty"`
	BillingDetail    *BillingDetail    `json:"billingDetail,omitempty"`
	SuccessURL       *string           `json:"successUrl,omitempty"`
	ExpiresInSeconds *int              `json:"expiresInSeconds,omitempty"`
	DarkMode         *bool             `json:"darkMode,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	// OrderMerchantExternalID is the order-side business identifier (max 128 chars); inherited by orders, payments, refunds.
	OrderMerchantExternalID *string `json:"orderMerchantExternalId,omitempty"`
}

// CheckoutSessionResult is the response of Checkout.CreateSession and
// Checkout.Anonymous.Create.
type CheckoutSessionResult struct {
	SessionID   string   `json:"sessionId"`
	CheckoutURL string   `json:"checkoutUrl"`
	ExpiresAt   string   `json:"expiresAt"`
	Warnings    []Notice `json:"warnings,omitempty"`
}

// -----------------------------------------------------------------------------
// Buyer self-service
// -----------------------------------------------------------------------------

// CancelOnetimeOrderParams is the input to BuyerSession.CancelOnetimeOrder.
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
// BuyerSession.ReactivateSubscription.
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

// CreateRefundTicketParams is the input to BuyerSession.CreateRefundTicket.
type CreateRefundTicketParams struct {
	PaymentID       string          `json:"paymentId"`
	Reason          string          `json:"reason"`
	RequestedAmount RequestedAmount `json:"requestedAmount"`
	Metadata        map[string]any  `json:"metadata,omitempty"`
	// RefundTicketMerchantExternalID is the refund-ticket business identifier (max 128 chars); inherited by the executed refund on PSP success.
	RefundTicketMerchantExternalID *string `json:"refundTicketMerchantExternalId,omitempty"`
}

// ResubmitRefundTicketParams is the input to BuyerSession.ResubmitRefundTicket.
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
	// BuyerIdentity is encoded into the JWT for merchant-side buyer
	// identification. Use BuyerEmail to pre-fill the checkout form's email
	// input; the two fields are independent.
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
// [GraphQLQuery] / [BuyerGraphQLQuery].
type TypedGraphQLResponse[T any] struct {
	Data     T                `json:"data"`
	Errors   []GraphQLError   `json:"errors,omitempty"`
	Warnings []GraphQLWarning `json:"warnings,omitempty"`
}

// -----------------------------------------------------------------------------
// Webhook
// -----------------------------------------------------------------------------

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

	Amount    string   `json:"amount"`
	TaxAmount string   `json:"taxAmount"`
	TaxRate   *float64 `json:"taxRate,omitempty"`
	TaxName   *string  `json:"taxName,omitempty"`
	Subtotal  *string  `json:"subtotal,omitempty"`
	Total     *string  `json:"total,omitempty"`

	ProductName        string            `json:"productName"`
	ProductDescription *string           `json:"productDescription,omitempty"`
	ProductMetadata    map[string]string `json:"productMetadata,omitempty"`

	PaymentID            *string `json:"paymentId,omitempty"`
	PaymentStatus        *string `json:"paymentStatus,omitempty"`
	PaymentMethod        *string `json:"paymentMethod,omitempty"`
	PaymentLast4         *string `json:"paymentLast4,omitempty"`
	PaymentFailureReason *string `json:"paymentFailureReason,omitempty"`
	PaymentDate          *string `json:"paymentDate,omitempty"`

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
	// ToleranceMS is the replay-protection window in milliseconds. Set to a
	// negative value to disable timestamp checking. Zero (default) is treated
	// as the default 5-minute window.
	ToleranceMS int64
	// PublicKey, when non-empty, overrides all resolution chains and is used
	// directly for verification.
	PublicKey string
	// PublicKeys injects config-level keys into the resolution chain.
	// When using Webhooks.Verify this is set automatically from the client
	// config; standalone callers can pass it directly.
	PublicKeys *WebhookPublicKeys
}
