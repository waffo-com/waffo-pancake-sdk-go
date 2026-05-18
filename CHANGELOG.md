# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
