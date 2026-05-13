# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
