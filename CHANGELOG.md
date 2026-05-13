# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] — Unreleased

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
