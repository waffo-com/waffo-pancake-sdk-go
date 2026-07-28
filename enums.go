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

// CashierLanguage is the default language of the hosted checkout page (IETF BCP 47).
//
// The customer can still switch language on the page. Language x currency mismatches
// are rejected by the payment provider (A0026), not by this SDK.
type CashierLanguage string

const (
	CashierLanguageEn       CashierLanguage = "en"
	CashierLanguagePtBR     CashierLanguage = "pt-BR"
	CashierLanguageEsMX     CashierLanguage = "es-MX"
	CashierLanguageIDID     CashierLanguage = "id-ID" // Indonesian
	CashierLanguageViVN     CashierLanguage = "vi-VN"
	CashierLanguageRuRU     CashierLanguage = "ru-RU"
	CashierLanguageEnKE     CashierLanguage = "en-KE"
	CashierLanguageEsPE     CashierLanguage = "es-PE"
	CashierLanguageEsCO     CashierLanguage = "es-CO"
	CashierLanguageEsCL     CashierLanguage = "es-CL"
	CashierLanguageZhHantTW CashierLanguage = "zh-Hant-TW"
	CashierLanguageZhHantHK CashierLanguage = "zh-Hant-HK"
	CashierLanguageThTH     CashierLanguage = "th-TH"
	CashierLanguageJaJP     CashierLanguage = "ja-JP"
	CashierLanguageEnNG     CashierLanguage = "en-NG"
	CashierLanguageKoKR     CashierLanguage = "ko-KR"
	CashierLanguageEnHK     CashierLanguage = "en-HK"
	CashierLanguageZhHansHK CashierLanguage = "zh-Hans-HK"
	CashierLanguagePlPL     CashierLanguage = "pl-PL"
	CashierLanguageTrTR     CashierLanguage = "tr-TR"
	CashierLanguageZhHans   CashierLanguage = "zh-Hans"
	CashierLanguageMsMY     CashierLanguage = "ms-MY"
)

// PaymentMethod is a payment method offered on the hosted checkout page.
//
// Availability depends on the product type x currency pair. One-time: USD supports
// all four; EUR, GBP, HKD and JPY support card, applepay and googlepay; CNY supports
// wechat. Subscription: USD, EUR, GBP, HKD and JPY support card, applepay and
// googlepay. Currencies outside this matrix cannot be charged at all — checkout
// session creation is rejected with a 400.
type PaymentMethod string

const (
	PaymentMethodCard      PaymentMethod = "card"
	PaymentMethodApplePay  PaymentMethod = "applepay"
	PaymentMethodGooglePay PaymentMethod = "googlepay"
	PaymentMethodWeChat    PaymentMethod = "wechat"
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

// ScanAction is the final content-safety verdict — continue to generation only
// when [ScanActionAllow].
type ScanAction string

const (
	ScanActionAllow  ScanAction = "allow"
	ScanActionReview ScanAction = "review"
	ScanActionBlock  ScanAction = "block"
)

// ScanReasonCode is the stable machine-readable reason for a scan verdict.
type ScanReasonCode string

const (
	ScanReasonCodeAllowed           ScanReasonCode = "allowed"
	ScanReasonCodeReviewRequired    ScanReasonCode = "review_required"
	ScanReasonCodeRestrictedContent ScanReasonCode = "restricted_content"
	ScanReasonCodeServiceDegraded   ScanReasonCode = "service_degraded"
)

// ScanPolicyCategory is a matched content-safety policy category.
type ScanPolicyCategory string

const (
	ScanPolicyCategoryCsamMinor                   ScanPolicyCategory = "csam_minor"
	ScanPolicyCategorySexualViolenceNonconsensual ScanPolicyCategory = "sexual_violence_nonconsensual"
	ScanPolicyCategoryUndressTransform            ScanPolicyCategory = "undress_transform"
	ScanPolicyCategoryFaceSwapIdentity            ScanPolicyCategory = "face_swap_identity"
	ScanPolicyCategoryBestialityRestricted        ScanPolicyCategory = "bestiality_restricted"
	ScanPolicyCategoryAdultNsfw                   ScanPolicyCategory = "adult_nsfw"
)

// ScanSemanticMode controls how the external semantic channel participates in
// a scan.
type ScanSemanticMode string

const (
	ScanSemanticModeOff     ScanSemanticMode = "off"
	ScanSemanticModeShadow  ScanSemanticMode = "shadow"
	ScanSemanticModeEnforce ScanSemanticMode = "enforce"
)

// ScanSemanticStatus reports whether/how the semantic channel contributed to a
// scan.
type ScanSemanticStatus string

const (
	ScanSemanticStatusDisabled          ScanSemanticStatus = "disabled"
	ScanSemanticStatusScored            ScanSemanticStatus = "scored"
	ScanSemanticStatusShadowScored      ScanSemanticStatus = "shadow_scored"
	ScanSemanticStatusSkippedRulesBlock ScanSemanticStatus = "skipped_rules_block"
	ScanSemanticStatusSkippedBudget     ScanSemanticStatus = "skipped_budget"
	ScanSemanticStatusProviderTimeout   ScanSemanticStatus = "provider_timeout"
	ScanSemanticStatusProviderError     ScanSemanticStatus = "provider_error"
)
