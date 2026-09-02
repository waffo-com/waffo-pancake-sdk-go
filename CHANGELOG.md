# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.11.0] — 2026-09-02

Subscription period and status now travel on the subscription events only. Matches `@waffo/pancake-ts@0.20.0`.

### Removed

- **BREAKING: `WebhookEventTypeSubscriptionUpdated` (`subscription.updated`)** — the platform has no publisher for it; plan changes are published as `subscription.plan_changed`. Migration: switch the constant to `WebhookEventTypeSubscriptionPlanChanged`. Code that matched on the raw string never fired and can be deleted.
- **`subscription.payment_succeeded` no longer carries `BillingPeriod`, `CurrentPeriodStart`, `CurrentPeriodEnd`, `CanceledAt` or `OrderStatus`.** No struct field changed — those five stay on `WebhookEventData` for the events that do carry them, but they arrive nil on this event. `docs/webhook-guide.md` has the field-by-event table and a per-use-case migration path.

### Added

- **`WebhookEventTypeSubscriptionRenewed` (`subscription.renewed`)** — emitted when the current billing period actually rolls forward, carrying the new period. The first period is not a renewal and does not emit it.
- **`WebhookEventTypeSubscriptionRecovered` (`subscription.recovered`)** — emitted when a retried charge brings a past-due subscription back to active, closing the loop with `subscription.past_due`.
- **`WebhookEventTypeSubscriptionPlanChanged` / `...PlanChangeScheduled` / `...PlanChangeFailed`** — the three plan-change events.

### Changed

- Feature parity target updated to `@waffo/pancake-ts@0.20.x` (`doc.go` and `README.md` still declared `0.18.x`).

## [0.10.0] — 2026-08-18

Subscription products can now charge for the trial period. Matches `@waffo/pancake-ts@0.19.0`.

### Added

- **`PriceInfo.TrialAmount`** — trial period price as a display string, subscription products only. Leave it nil for a free trial. It requires `metadata.trialDays` on the product and must be lower than `Amount`; the API rejects either violation with HTTP 400.
- **`PriceSnapshot`** — the struct `CreateCheckoutSessionParams.PriceSnapshot` accepts. Same fields as `PriceInfo` without `TrialAmount`.

### Changed

- **BREAKING: `CreateCheckoutSessionParams.PriceSnapshot` is `*PriceSnapshot` rather than `*PriceInfo`.** A session-level override replaces the regular period price; the trial price comes from the product version locked into the session, so a `TrialAmount` passed here would be dropped server-side. Go resolves struct types by name, so existing `&PriceInfo{...}` literals stop compiling. Migration: rename the literal to `&PriceSnapshot{...}` — field names and JSON tags are identical.
- Feature parity target updated to `@waffo/pancake-ts@0.19.x`.

## [0.9.0] — 2026-08-08

Customer sessions never reached the API, and webhook retries were rejected as replays. Matches the same two fixes in `@waffo/pancake-ts@0.18.0`.

### Fixed

- **Customer session requests now send `X-Environment`.** A session token carries no environment of its own, so the gateway requires the header next to the Bearer credential and rejects the request with HTTP 400 `Incomplete JWT authentication headers` without it. The header was missing, which made every `CustomerSession` method unusable: `CancelSubscription`, `CancelOnetimeOrder`, `ReactivateSubscription`, `CreateRefundTicket`, `ResubmitRefundTicket`, and `GraphQL.Query`. API Key requests were never affected — the gateway derives their environment from the key.
- **Webhook verification no longer rejects legitimate retries.** The signature timestamp is stamped once, before the first delivery attempt, and retries reuse the original header — so the last retry of a schedule arrives with a timestamp as old as the schedule itself (observed above 31 minutes). Against the old 5-minute window every late retry failed verification as a suspected replay. `VerifyWebhook` now allows timestamps up to **45 minutes** old.

### Added

- **`Config.Environment`** — `EnvironmentTest` or `EnvironmentProd`, the environment customer sessions operate in.
- **`(*Client).CustomerWithEnvironment(token, environment)`** — overrides `Config.Environment` for a single session.
- **`VerifyWebhookOptions.FutureToleranceMS`** — how far in the future a signature timestamp may be, default `60000` (1 minute). Raise it for a receiving server with known clock skew.
- **`DefaultWebhookFutureToleranceMS`** — the exported default for the above.

### Changed

- **`Config.Environment` is required for customer sessions.** `Customer(token)` keeps its signature and still performs no I/O; when no environment is available the first session method returns `*Error` (400, `ErrorLayerSDK`) before sending anything. There is no default — guessing would route the call to the other environment. Migration: set `Config.Environment`, or use `CustomerWithEnvironment`.
- **`DefaultWebhookToleranceMS` raised from `300000` to `2700000`, and the window is now asymmetric** — matching the gateway's API Key check, which pairs a wide past-facing window with a tight future-facing one. `ToleranceMS` now means "how far in the past"; the future direction is `FutureToleranceMS`. A negative `ToleranceMS` still disables the check entirely. A captured request stays replayable for longer under the wider window, so keep your handler idempotent on the event `ID` — that, not the window, is the real defense.
- Feature parity target updated to `@waffo/pancake-ts@0.18.x`.

---

## [0.8.0] — 2026-08-03

`SupportEmail` and `Website` were never applied by the update-store endpoint — passing them was silently ignored.

### Removed

- **BREAKING** `UpdateStoreParams.SupportEmail` and `UpdateStoreParams.Website` — the endpoint never wrote these fields, so passing them had no effect. They are derived from ownership verification and are set only by the flows that prove it: email code binding, domain verification, or KYB approval. Both remain readable on `Store`. Migration: drop them from your `Stores.Update` calls; a `NullValuePtr` you were passing to "clear" them was never clearing anything. Matches the same removal in `@waffo/pancake-ts@0.17.0`.

### Fixed

- `docs/api-reference.md` — `Name` is limited to 48 characters, not 100 (100 is the DB column width; the application layer caps at 48) and rejects control characters as of store-service 2026.8.3.

## [0.7.0] — 2026-07-28

Adds per-transaction payment method selection on the hosted checkout page. Brings
feature parity with `@waffo/pancake-ts@0.17.0`.

### Added

- `CashierLanguage` enum + `CreateCheckoutSessionParams.Language` — default language of the hosted checkout page (IETF BCP 47, 22 tags). Catches up with `@waffo/pancake-ts@0.13.0`, which shipped `language` three minor versions ago.
- `PaymentMethod` enum — `PaymentMethodCard` / `PaymentMethodApplePay` / `PaymentMethodGooglePay` / `PaymentMethodWeChat`
- `CreateCheckoutSessionParams.IncludePaymentMethods` — whitelist: offer only these. Every value must be supported by the product type × currency pair (one-time `USD` supports all four, `CNY` supports `wechat`, the other currencies support card / applepay / googlepay); unsupported values are rejected with a 400.
- `CreateCheckoutSessionParams.ExcludePaymentMethods` — blacklist: offer everything the currency supports except these. Values the currency does not offer are ignored, so one blacklist can be reused across currencies. Mutually exclusive with `IncludePaymentMethods`.

### Changed

- Currencies outside the payment method matrix are now rejected at checkout session creation (400) instead of falling through to the provider. Affects one-time `THB` and subscription `CNY`, neither of which has ever produced a successful charge.

---

## [0.6.0] — 2026-07-18

Adds content-safety prompt scanning for AIGC generation. Brings feature parity
with `@waffo/pancake-ts@0.14.0`.

### Added

- `client.ContentSafety.ScanPrompt(ctx, params)` — scan a user prompt before
  AIGC generation; returns a redacted verdict (`Action` = allow / review /
  block, continue only when allow). Stateless (prompt text never stored); fails
  closed to `review` if the safety service is briefly unavailable. POSTs to
  `/v1/actions/verification/scan-prompt`.
- Types: `ScanPromptParams`, `ScanResult`; enums `ScanAction`,
  `ScanReasonCode`, `ScanPolicyCategory`, `ScanSemanticMode`,
  `ScanSemanticStatus`.

## [0.5.0] — 2026-07-14

Renames the "buyer" persona to "customer" across the public API, matching
Waffo terminology (the session-token JWT role is `customer`).

### Changed

- Customer-named identifiers are now the primary public API:
  `CustomerSession`, `CustomerGraphQLResource`, `(*Client).Customer(token)`,
  and `pancake.CustomerGraphQLQuery[T]`. Files `buyer.go` /
  `buyer_http_client.go` are now `customer.go` / `customer_http_client.go`,
  and `examples/buyer` is now `examples/customer`.

### Deprecated

- `BuyerSession` and `BuyerGraphQLResource` remain as type aliases of the
  customer-named types; `(*Client).Buyer(token)` and
  `pancake.BuyerGraphQLQuery[T]` remain as thin wrappers. All carry
  `Deprecated:` godoc markers and are kept for one release.

### Notes

- The wire contract is unchanged: JSON fields `buyerIdentity`, `buyerEmail`,
  and every other `buyer*` wire field are untouched, as are the Go struct
  fields mapped to them (`BuyerIdentity`, `BuyerEmail`,
  `MerchantProvidedBuyerIdentity`). HTTP paths, headers, and GraphQL query
  text are unchanged.

## [0.4.0] — 2026-06-01

Brings feature parity with `@waffo/pancake-ts@0.10.0`: the full 19-field
`NotificationSettings` schema (8 consumer-email + 11 merchant-notify toggles).

### Added

- `NotificationSettings` gains 11 fields aligning with the platform schema:
  - Consumer email (platform-managed, silently dropped if included):
    `EmailTrialStarted`, `EmailTrialEnding`
  - Merchant notify (merchant-writable): `NotifySubscriptionCanceled`,
    `NotifySubscriptionEnded`, `NotifySubscriptionPastDue`,
    `NotifySubscriptionRenewed`, `NotifySubscriptionUncanceled`,
    `NotifySubscriptionUpdated`, `NotifyChargeback`, `NotifyPayoutCompleted`,
    `NotifyPayoutFailed`

### Notes

Non-breaking and purely additive — all new fields are `*bool` and `omitempty`.
Unlike `@waffo/pancake-ts@0.10.0`, this SDK does **not** introduce a separate
`MerchantWritableNotificationSettings` type because Go cannot express
compile-time field-set narrowing on a single struct. The `Email*` toggles
remain present on `NotificationSettings` for response decoding (the server
returns all 19 fields) but are documented as silently dropped on update.

## [0.3.1] — 2026-05-21

### Fixed

- `CreateCheckoutSessionParams.OrderMerchantExternalID` was published as
  `OrderMerchantExternalId` (lowercase `d`) in v0.3.0, breaking the build for
  any caller that referenced the canonical Go-style `ID` suffix used by every
  other identifier in this package. v0.3.1 restores the correct `ID` casing.
  Wire-level JSON tag is unchanged (`orderMerchantExternalId`).

### Notes

- v0.3.0 is retracted via `go.mod` so the Go toolchain skips it automatically.
  Upgrade directly to v0.3.1.

## [0.3.0] — 2026-05-21

Brings feature parity with `@waffo/pancake-ts@0.9.0`: flat dual-key external-id
fields across write inputs, response entities, and webhook payload. The same
field name now appears at every layer (REST request body / response / webhook
payload / GraphQL types).

### Added

- `CreateCheckoutSessionParams.OrderMerchantExternalID` — order business
  identifier (optional, max 128 chars). Propagates to order/payment records
  and surfaces in webhook payload (`data.orderMerchantExternalId`) and
  GraphQL (`Order.orderMerchantExternalId` / `Payment.orderMerchantExternalId`
  / `Refund.orderMerchantExternalId`).
- `CreateRefundTicketParams.RefundTicketMerchantExternalID` — refund-ticket
  business identifier. Propagates to the executed refund record on PSP
  success and surfaces in webhook payload
  (`data.refundTicketMerchantExternalId`) and GraphQL
  (`RefundTicket.refundTicketMerchantExternalId` /
  `Refund.refundTicketMerchantExternalId`).
- `RefundTicket.RefundTicketMerchantExternalID` — read-side response field
  (immutable across resubmits).
- `WebhookEventData.OrderMerchantExternalID` and
  `WebhookEventData.RefundTicketMerchantExternalID` — both coexist on
  `refund.*` events; order/payment events carry only the order key.

### Notes

Non-breaking; all four new fields are pointer types with `omitempty`. JSON
tags use camelCase keys aligned with `@waffo/pancake-ts` and the webhook
envelope. Visitor / store-slug checkout flows silently drop
`OrderMerchantExternalID` (accepted on the wire, not persisted) because
merchant business identifiers are not meant to be supplied client-side.

## [0.2.0] — 2026-05-17

Brings feature parity with `@waffo/pancake-ts@0.8.0`: unified envelope handling
across REST and GraphQL, with `Notice` warnings surfaced to callers.

### Fixed

- **GraphQL queries actually return data.** Prior versions assumed a
  double-wrapped envelope (`{data:{data,errors,warnings}}`) and stripped one
  layer too many, leaving `GraphQLResponse.Data` as an empty `json.RawMessage`
  regardless of what the server returned. The wire is in fact the standard
  single-layer GraphQL envelope (`{data, errors?, warnings?}`); the SDK now
  returns it verbatim.
- **GraphQL queries no longer carry `X-Idempotency-Key`.** Queries are
  read-only; the gateway was caching them for 24h and serving stale snapshots
  on subsequent identical requests. Side-effect-free queries now hit the live
  DB on every call. `GraphQLResource.Query` and `BuyerGraphQLResource.Query`
  set `postOptions.NoIdempotency` internally.
- **REST `warnings` are no longer dropped.** Every REST action endpoint can
  return `warnings: Notice[]` (handbook `command-layer.md`); prior
  `httpClient.post` returned only the unwrapped `data` field, throwing away
  migration `AIHint` notices like `update-store`'s `webhookSettings field
  ignored → Switch to client.Webhooks.Add/Update/Remove`.

### Added

- **`Notice` type** (`{Message, Layer, AIHint, Locations, Path}`) — unified
  shape used by both REST and GraphQL `errors[]` / `warnings[]`. `APIError`,
  `GraphQLError`, and `GraphQLWarning` are kept as type aliases for backwards
  compatibility (see `errors.go`, `types.go`).
- **`Warnings []Notice` field on every `*Result` struct** (`CreateStoreResult`,
  `AddMerchantResult`, `OnetimeProductResult`, `SubscriptionProductResult`,
  `SubscriptionProductGroupResult`, `CancelSubscriptionResult`,
  `CheckoutSessionResult`, `CancelOnetimeOrderResult`,
  `ReactivateSubscriptionResult`, `RefundTicketResult`, `SessionToken`,
  `AddWebhookResult`, `AuthenticatedCheckoutResult`, `RemoveMerchantResult`,
  `UpdateRoleResult`). The `(*Result, error)` signature is unchanged; callers
  read `res.Warnings` to see server advisories.
- **`postOptions.NoIdempotency`** — boolean to suppress the
  `X-Idempotency-Key` header on a per-call basis.
- **`resource_helpers.go`** — package-level generics `postAction[T]` /
  `buyerPostAction[T]` factor out the throw-on-errors + extract-warnings policy
  shared by every resource method.
- **README "Warnings (Migration Notices)" section** with REST + GraphQL Go
  examples and explicit guidance for LLM/agent consumers to act on `AIHint`.

### Changed

- **Transport refactored**: `httpClient.post` / `buyerHTTPClient.post` now
  return `(status, *envelope, error)` without throwing on `errors[]`. Throw /
  unwrap / warnings handling moved to the resource layer via the new
  `postAction[T]` and `buyerPostAction[T]` generics in `resource_helpers.go`.
  `GraphQLResource.Query` / `BuyerGraphQLResource.Query` consume the envelope
  directly and return it verbatim.

### Notes

- Backwards compatible at the call site: `(*Result, error)` signatures are
  unchanged. Existing code keeps compiling; the new `Result.Warnings` slice is
  empty unless the server actually returns warnings.
- Test coverage of the core package raised to ≥ 90% statements.

## [0.1.1] — 2026-05-13

### Notes

- Republishes v0.1.0 under a fresh tag because v0.1.0 was retracted: the tag
  was force-updated to a new commit during the brief window between the first
  successful `go get` and the sum.golang.org cache settling, leaving a hash
  mismatch. `go.mod` carries a `retract v0.1.0` directive so the Go toolchain
  skips it automatically.
- No code changes vs. v0.1.0 — identical SDK content, just on a stable tag.

## [0.1.0] — 2026-05-13 (retracted)

### Added

- Initial release. Feature parity with `@waffo/pancake-ts@0.7.x`.
- `Client` with auto-signed (RSA-SHA256) requests and deterministic idempotency keys.
- 11 resource namespaces: `Auth`, `Stores`, `StoreMerchants`, `OnetimeProducts`,
  `SubscriptionProducts`, `SubscriptionProductGroups`, `Orders`, `Checkout`,
  `GraphQL`, `Webhooks`, plus the `Buyer(token)` factory for buyer self-service.
- `Checkout.Authenticated.Create` runs the token + session calls concurrently and
  appends `#token=<jwt>` to the returned URL.
- Webhook verification with built-in test/prod public keys, four-level fallback
  resolution chain, and 5-minute replay tolerance.
- Generic helpers `pancake.GraphQLQuery[T]`, `pancake.BuyerGraphQLQuery[T]`, and
  `pancake.VerifyWebhookTyped[T]` for type-safe payloads.
- Zero runtime dependencies.

### Retracted

Hash mismatch with sum.golang.org. Use v0.1.1 instead.
