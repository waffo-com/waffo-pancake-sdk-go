package pancake

// Environment identifies which side of the test/prod boundary a resource
// belongs to.
type Environment string

const (
	EnvironmentTest Environment = "test"
	EnvironmentProd Environment = "prod"
)

// TaxCategory classifies a product for tax computation purposes.
type TaxCategory string

const (
	TaxCategoryDigitalGoods        TaxCategory = "digital_goods"
	TaxCategorySaaS                TaxCategory = "saas"
	TaxCategorySoftware            TaxCategory = "software"
	TaxCategoryEbook               TaxCategory = "ebook"
	TaxCategoryOnlineCourse        TaxCategory = "online_course"
	TaxCategoryConsulting          TaxCategory = "consulting"
	TaxCategoryProfessionalService TaxCategory = "professional_service"
)

// BillingPeriod is the recurrence cadence of a subscription product.
type BillingPeriod string

const (
	BillingPeriodWeekly    BillingPeriod = "weekly"
	BillingPeriodMonthly   BillingPeriod = "monthly"
	BillingPeriodQuarterly BillingPeriod = "quarterly"
	BillingPeriodYearly    BillingPeriod = "yearly"
)

// ProductVersionStatus is the lifecycle state of a product version.
type ProductVersionStatus string

const (
	ProductVersionStatusActive   ProductVersionStatus = "active"
	ProductVersionStatusInactive ProductVersionStatus = "inactive"
)

// EntityStatus is the lifecycle state of a store entity.
type EntityStatus string

const (
	EntityStatusActive    EntityStatus = "active"
	EntityStatusInactive  EntityStatus = "inactive"
	EntityStatusSuspended EntityStatus = "suspended"
)

// StoreRole is a store membership role.
type StoreRole string

const (
	StoreRoleOwner  StoreRole = "owner"
	StoreRoleAdmin  StoreRole = "admin"
	StoreRoleMember StoreRole = "member"
)

// OnetimeOrderStatus is the lifecycle state of a one-time order.
type OnetimeOrderStatus string

const (
	OnetimeOrderStatusPending   OnetimeOrderStatus = "pending"
	OnetimeOrderStatusCompleted OnetimeOrderStatus = "completed"
	OnetimeOrderStatusCanceled  OnetimeOrderStatus = "canceled"
)

// SubscriptionOrderStatus is the lifecycle state of a subscription order.
//
// State machine:
//   - pending   -> active, canceled, closed
//   - active    -> canceling, past_due, canceled, expired
//   - canceling -> active, canceled
//   - past_due  -> active, canceled
//   - closed    -> terminal
//   - canceled  -> terminal
//   - expired   -> terminal
type SubscriptionOrderStatus string

const (
	SubscriptionOrderStatusPending   SubscriptionOrderStatus = "pending"
	SubscriptionOrderStatusActive    SubscriptionOrderStatus = "active"
	SubscriptionOrderStatusCanceling SubscriptionOrderStatus = "canceling"
	SubscriptionOrderStatusPastDue   SubscriptionOrderStatus = "past_due"
	SubscriptionOrderStatusClosed    SubscriptionOrderStatus = "closed"
	SubscriptionOrderStatusCanceled  SubscriptionOrderStatus = "canceled"
	SubscriptionOrderStatusExpired   SubscriptionOrderStatus = "expired"
)

// PaymentStatus is the lifecycle state of a payment.
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCanceled  PaymentStatus = "canceled"
)

// RefundTicketStatus is the lifecycle state of a refund request ticket.
type RefundTicketStatus string

const (
	RefundTicketStatusPending     RefundTicketStatus = "pending"
	RefundTicketStatusUnderReview RefundTicketStatus = "under_review"
	RefundTicketStatusApproved    RefundTicketStatus = "approved"
	RefundTicketStatusRejected    RefundTicketStatus = "rejected"
	RefundTicketStatusReturned    RefundTicketStatus = "returned"
	RefundTicketStatusProcessing  RefundTicketStatus = "processing"
	RefundTicketStatusSucceeded   RefundTicketStatus = "succeeded"
	RefundTicketStatusFailed      RefundTicketStatus = "failed"
	RefundTicketStatusCancelled   RefundTicketStatus = "cancelled"
)

// RefundStatus is the final state of a refund execution.
type RefundStatus string

const (
	RefundStatusSucceeded RefundStatus = "succeeded"
	RefundStatusFailed    RefundStatus = "failed"
)

// MediaType is the kind of a product media asset.
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
)

// ErrorLayer identifies the layer where an API error originated. SDK-side
// validation failures carry the special value [ErrorLayerSDK].
type ErrorLayer string

const (
	ErrorLayerGateway  ErrorLayer = "gateway"
	ErrorLayerUser     ErrorLayer = "user"
	ErrorLayerStore    ErrorLayer = "store"
	ErrorLayerProduct  ErrorLayer = "product"
	ErrorLayerOrder    ErrorLayer = "order"
	ErrorLayerTicket   ErrorLayer = "ticket"
	ErrorLayerGraphQL  ErrorLayer = "graphql"
	ErrorLayerResource ErrorLayer = "resource"
	ErrorLayerEmail    ErrorLayer = "email"
	ErrorLayerSDK      ErrorLayer = "sdk"
)

// WebhookEventType is the kind of business event delivered by a webhook.
type WebhookEventType string

const (
	WebhookEventTypeOrderCompleted               WebhookEventType = "order.completed"
	WebhookEventTypeSubscriptionActivated        WebhookEventType = "subscription.activated"
	WebhookEventTypeSubscriptionPaymentSucceeded WebhookEventType = "subscription.payment_succeeded"
	WebhookEventTypeSubscriptionCanceling        WebhookEventType = "subscription.canceling"
	WebhookEventTypeSubscriptionUncanceled       WebhookEventType = "subscription.uncanceled"
	WebhookEventTypeSubscriptionUpdated          WebhookEventType = "subscription.updated"
	WebhookEventTypeSubscriptionCanceled         WebhookEventType = "subscription.canceled"
	WebhookEventTypeSubscriptionPastDue          WebhookEventType = "subscription.past_due"
	WebhookEventTypeRefundSucceeded              WebhookEventType = "refund.succeeded"
	WebhookEventTypeRefundFailed                 WebhookEventType = "refund.failed"
)

// WebhookChannel is the delivery channel of a configured webhook endpoint.
type WebhookChannel string

const (
	WebhookChannelHTTP     WebhookChannel = "http"
	WebhookChannelFeishu   WebhookChannel = "feishu"
	WebhookChannelDiscord  WebhookChannel = "discord"
	WebhookChannelTelegram WebhookChannel = "telegram"
	WebhookChannelSlack    WebhookChannel = "slack"
)
