# Webhook Guide

Waffo Pancake sends webhook events to your configured endpoint when payment, subscription, and refund state changes occur. The SDK provides `pancake.VerifyWebhook()` (and `client.Webhooks.Verify()`) to validate signatures and parse events.

## Overview

- **Algorithm**: RSA-SHA256 with environment-specific key pairs
- **Dual environment**: Test and production use separate key pairs; the SDK resolves the correct key automatically
- **Multi-level key loading**: Per-call option → config field → environment variable → built-in hardcoded key
- **Replay protection**: 5-minute timestamp tolerance by default
- **Environment auto-detection**: Tries the production key first, falls back to test
- **Zero I/O**: Verification is a pure local cryptographic operation

## Signature Verification

```
1. Parse X-Waffo-Signature header → t (timestamp) + v1 (Base64 signature)
2. Build signature input: t + "." + rawBody
3. Verify v1 with RSA-SHA256 using the Waffo public key
4. Check timestamp (default 5-minute tolerance to prevent replay attacks)
```

## Usage

### `net/http` (standard library)

```go
package main

import (
    "encoding/json"
    "io"
    "log"
    "net/http"

    pancake "github.com/waffo-com/waffo-pancake-sdk-go"
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    // IMPORTANT: use the raw body — parsing and re-serializing breaks the signature.
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "read body", http.StatusBadRequest)
        return
    }
    sig := r.Header.Get("X-Waffo-Signature")

    event, err := pancake.VerifyWebhook(string(body), sig, nil)
    if err != nil {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }

    // Respond immediately, process asynchronously.
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("OK"))

    // Use event.ID for idempotent deduplication.
    switch event.EventType {
    case string(pancake.WebhookEventTypeOrderCompleted):
        var data pancake.WebhookEventData
        _ = json.Unmarshal(event.Data, &data)
        log.Printf("Order %s completed", data.OrderID)
    case string(pancake.WebhookEventTypeSubscriptionActivated):
        var data pancake.WebhookEventData
        _ = json.Unmarshal(event.Data, &data)
        log.Printf("Subscription activated for %s", data.BuyerEmail)
    case string(pancake.WebhookEventTypeSubscriptionCanceled):
        var data pancake.WebhookEventData
        _ = json.Unmarshal(event.Data, &data)
        log.Printf("Subscription canceled: %s", data.OrderID)
    case string(pancake.WebhookEventTypeRefundSucceeded):
        var data pancake.WebhookEventData
        _ = json.Unmarshal(event.Data, &data)
        log.Printf("Refund %s %s", data.Amount, data.Currency)
    }
}

func main() {
    http.HandleFunc("/webhooks", webhookHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Typed verification (skip the manual `json.Unmarshal`)

```go
event, err := pancake.VerifyWebhookTyped[pancake.WebhookEventData](
    string(body),
    r.Header.Get("X-Waffo-Signature"),
    nil,
)
if err != nil {
    http.Error(w, "Invalid signature", http.StatusUnauthorized)
    return
}
// event.Data is pancake.WebhookEventData (struct, not raw bytes)
log.Printf("Order %s completed", event.Data.OrderID)
```

### chi / gin (third-party routers)

The pattern is identical — read the raw body, pull the header, call `pancake.VerifyWebhook`:

```go
// chi
r.Post("/webhooks", webhookHandler)

// gin
r.POST("/webhooks", func(c *gin.Context) {
    body, _ := io.ReadAll(c.Request.Body)
    sig := c.GetHeader("X-Waffo-Signature")

    event, err := pancake.VerifyWebhook(string(body), sig, nil)
    if err != nil {
        c.String(http.StatusUnauthorized, "Invalid signature")
        return
    }
    c.String(http.StatusOK, "OK")
    _ = event
})
```

### Options

```go
// Specify environment explicitly (skip auto-detection)
event, err := pancake.VerifyWebhook(body, sig, &pancake.VerifyWebhookOptions{
    Environment: pancake.EnvironmentProd,
})

// Disable replay protection (useful for testing) — negative value disables the check
event, err := pancake.VerifyWebhook(body, sig, &pancake.VerifyWebhookOptions{
    ToleranceMS: -1,
})

// Custom tolerance window (10 minutes)
event, err := pancake.VerifyWebhook(body, sig, &pancake.VerifyWebhookOptions{
    ToleranceMS: 600_000,
})
```

## Parameters

| Parameter         | Type                            | Description                                                               |
| ----------------- | ------------------------------- | ------------------------------------------------------------------------- |
| `payload`         | `string`                        | Raw request body string (must be unparsed)                                |
| `signatureHeader` | `string`                        | `X-Waffo-Signature` header value (format: `t=<timestamp>,v1=<signature>`) |
| `opts`            | `*pancake.VerifyWebhookOptions` | Optional configuration; pass `nil` for defaults                           |

### `VerifyWebhookOptions`

| Field         | Type                          | Default          | Description                                                                                                              |
| ------------- | ----------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `Environment` | `pancake.Environment`         | auto-detect      | Which environment's key to resolve. Zero value (`""`) means try prod first then test. Ignored when `PublicKey` is set.   |
| `ToleranceMS` | `int64`                       | `300000` (5 min) | Timestamp tolerance in ms. Set to a **negative** value to skip the timestamp check; zero is treated as the 5-min default |
| `PublicKey`   | `string`                      | —                | Per-call public key override (highest priority, skips all resolution)                                                    |
| `PublicKeys`  | `*pancake.WebhookPublicKeys`  | —                | Config-level key(s) for the resolution chain. Typically injected automatically by `client.Webhooks.Verify()`             |

```go
// WebhookPublicKeys layout:
type WebhookPublicKeys struct {
    Shared string // single key used for both environments
    Test   string
    Prod   string
}
```

## Dual-Environment Public Key Architecture

Waffo Pancake uses **separate RSA key pairs** for test and production environments. Webhook events from test mode are signed with the test private key; production events are signed with the production private key. The SDK must use the corresponding public key to verify each event.

```
                           ┌──────────────────┐
                           │   Waffo Server    │
                           ├──────────────────┤
                           │ Test Private Key  │──sign──→ test webhook events
                           │ Prod Private Key  │──sign──→ prod webhook events
                           └──────────────────┘
                                    │
                                    ▼
                           ┌──────────────────┐
                           │   Your Server     │
                           │   (SDK verify)    │
                           ├──────────────────┤
                           │ Test Public Key   │──verify──→ test webhook events
                           │ Prod Public Key   │──verify──→ prod webhook events
                           └──────────────────┘
```

When `Environment` is set, the SDK uses only the key for that environment. When the zero value is passed, the SDK **auto-detects** by trying the production key first, then falling back to the test key.

### Why Dual Keys?

- **Isolation**: Test and production environments are cryptographically separated. A test key cannot verify a production event and vice versa.
- **Key rotation**: Keys can be rotated independently per environment without affecting the other.
- **Security boundary**: Even if a test private key is compromised, production webhook integrity is unaffected.

## Multi-Level Public Key Resolution

For each environment, the SDK resolves the public key by walking a **6-level fallback chain**. The first non-empty value wins:

```
┌─────────────────────────────────────────────────────────┐
│                     Resolution Chain                     │
│                  (per environment: test/prod)            │
├─────┬───────────────────────────────────────────────────┤
│  1  │  opts.PublicKey (per-call override)               │ ← highest priority
├─────┼───────────────────────────────────────────────────┤
│  2  │  Config.WebhookPublicKey.Test / .Prod             │
│     │  (per-environment config field)                   │
├─────┼───────────────────────────────────────────────────┤
│  3  │  Config.WebhookPublicKey.Shared                   │
│     │  (shared config field for both envs)              │
├─────┼───────────────────────────────────────────────────┤
│  4  │  WAFFO_WEBHOOK_TEST_PUBLIC_KEY (test)             │
│     │  WAFFO_WEBHOOK_PROD_PUBLIC_KEY (prod)             │
│     │  (environment variable, per-env)                  │
├─────┼───────────────────────────────────────────────────┤
│  5  │  WAFFO_WEBHOOK_PUBLIC_KEY                         │
│     │  (environment variable, shared)                   │
├─────┼───────────────────────────────────────────────────┤
│  6  │  Built-in hardcoded key                           │ ← default fallback
│     │  (SDK-embedded Waffo public key)                  │
└─────┴───────────────────────────────────────────────────┘
```

### Level 1 — Per-call Override (`opts.PublicKey`)

The highest priority. When set, the SDK uses this key directly and **skips the entire resolution chain** — config keys, env vars, and built-in keys are all ignored. The `Environment` option is also ignored.

```go
// Use a specific key for this one call
event, err := pancake.VerifyWebhook(body, sig, &pancake.VerifyWebhookOptions{
    PublicKey: "-----BEGIN PUBLIC KEY-----\nMIIBIjAN...",
})

// Or via the client instance
event, err := client.Webhooks.Verify(body, sig, &pancake.VerifyWebhookOptions{
    PublicKey: rotatedKey,
})
```

**Use cases**: Key rotation testing, debugging with a known key, temporary override during migration.

### Level 2 — Config Per-Environment Keys (`WebhookPublicKey.Test` / `.Prod`)

Set the `Test` and/or `Prod` fields on `pancake.WebhookPublicKeys`. The SDK picks the key matching the resolved environment.

```go
client, err := pancake.New(pancake.Config{
    MerchantID: "MER_xxx",
    PrivateKey: "...",
    WebhookPublicKey: pancake.WebhookPublicKeys{
        Test: os.Getenv("MY_TEST_PUB_KEY"),
        Prod: os.Getenv("MY_PROD_PUB_KEY"),
    },
})

// Uses test key
event, err := client.Webhooks.Verify(body, sig, &pancake.VerifyWebhookOptions{
    Environment: pancake.EnvironmentTest,
})

// Uses prod key
event, err := client.Webhooks.Verify(body, sig, &pancake.VerifyWebhookOptions{
    Environment: pancake.EnvironmentProd,
})

// Auto-detect: tries prod first, then test
event, err := client.Webhooks.Verify(body, sig, nil)
```

You can provide only one environment — the other falls through to env vars or built-in keys:

```go
WebhookPublicKey: pancake.WebhookPublicKeys{
    Prod: customProdKey,
    // Test: "" → falls through to env var → built-in test key
}
```

### Level 3 — Config Shared Key (`WebhookPublicKey.Shared`)

A single string key applies to **both** environments. Useful when you use the same key pair for test and production (e.g., self-hosted deployments).

```go
client, err := pancake.New(pancake.Config{
    MerchantID: "MER_xxx",
    PrivateKey: "...",
    WebhookPublicKey: pancake.WebhookPublicKeys{
        Shared: os.Getenv("WAFFO_PUB_KEY"), // used for both test and prod
    },
})
```

### Level 4 — Environment Variables (Per-Environment) ⭐ Recommended Migration Path

When no config key is found, the SDK reads from process environment variables via `os.Getenv`:

| Environment | Variable Name                   |
| ----------- | ------------------------------- |
| test        | `WAFFO_WEBHOOK_TEST_PUBLIC_KEY` |
| prod        | `WAFFO_WEBHOOK_PROD_PUBLIC_KEY` |

> **When built-in hardcoded keys become invalid (e.g., Waffo rotates platform keys, or you migrate to a self-hosted deployment), the minimum-effort fix is to set environment variables. No code changes, no redeployment of application code — just update the env vars in your hosting platform (Fly.io, Render, AWS, Docker, etc.) and the SDK picks them up automatically on the next request.**

```bash
# .env, systemd EnvironmentFile, Docker env, AWS Parameter Store, etc.
WAFFO_WEBHOOK_TEST_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----\nMIIBIjAN..."
WAFFO_WEBHOOK_PROD_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----\nMIIBIjAN..."
```

```go
// No code changes needed — same code as before
event, err := pancake.VerifyWebhook(body, sig, nil)
// SDK auto-reads from env vars when built-in keys fail to match

// Or via client — also zero code change
client, _ := pancake.New(pancake.Config{
    MerchantID: "MER_xxx",
    PrivateKey: "...",
    // No WebhookPublicKey needed — env vars take effect automatically
})
event, err := client.Webhooks.Verify(body, sig, &pancake.VerifyWebhookOptions{
    Environment: pancake.EnvironmentProd,
})
// → reads WAFFO_WEBHOOK_PROD_PUBLIC_KEY
```

**Migration checklist when hardcoded keys expire:**

1. Obtain the new public keys from the Waffo dashboard or your platform admin
2. Set `WAFFO_WEBHOOK_PROD_PUBLIC_KEY` (and `WAFFO_WEBHOOK_TEST_PUBLIC_KEY` if needed) in your environment
3. Done — no code changes, no module upgrade, no redeployment of application code

### Level 5 — Environment Variable (Shared)

A single env var for both environments:

| Variable Name              | Used for           |
| -------------------------- | ------------------ |
| `WAFFO_WEBHOOK_PUBLIC_KEY` | Both test and prod |

```bash
WAFFO_WEBHOOK_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----\nMIIBIjAN..."
```

### Level 6 — Built-in Hardcoded Keys (Default)

If no custom key is found at any level, the SDK uses its embedded Waffo public keys. These are the official Waffo Pancake platform keys and are the default for most users.

**No configuration required** — this is the zero-config default.

```go
// Simplest usage — built-in keys handle everything
event, err := pancake.VerifyWebhook(body, sig, nil)
```

### Resolution Examples

| Scenario            | Config                                                 | Env Var                               | Result (prod)            |
| ------------------- | ------------------------------------------------------ | ------------------------------------- | ------------------------ |
| Default (no config) | —                                                      | —                                     | Built-in prod key        |
| Shared config key   | `WebhookPublicKey: { Shared: "KEY_A" }`                | —                                     | `KEY_A`                  |
| Per-env config      | `WebhookPublicKey: { Prod: "KEY_B" }`                  | —                                     | `KEY_B`                  |
| Env var only        | —                                                      | `WAFFO_WEBHOOK_PROD_PUBLIC_KEY=KEY_C` | `KEY_C`                  |
| Config + env var    | `WebhookPublicKey: { Prod: "KEY_D" }`                  | `WAFFO_WEBHOOK_PROD_PUBLIC_KEY=KEY_E` | `KEY_D` (config wins)    |
| Per-call override   | `WebhookPublicKey: { Prod: "KEY_F" }` + `opts.PublicKey` | —                                   | `opts.PublicKey` wins    |

## Public Key Formats

All public key inputs at every level (config, env vars, per-call) are automatically **normalized** by the SDK. The following formats are accepted:

| Format                  | Example                                                     | Notes                                   |
| ----------------------- | ----------------------------------------------------------- | --------------------------------------- |
| Standard SPKI PEM       | `-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----` | Recommended                             |
| PKCS#1 PEM              | `-----BEGIN RSA PUBLIC KEY-----\n...`                       | Also accepted                           |
| Literal `\n` (env vars) | `"-----BEGIN PUBLIC KEY-----\\nMIIB..."`                    | Common in `.env` files and CI secrets   |
| Windows line endings    | `\r\n`                                                      | Converted to `\n`                       |
| Raw base64 (no headers) | `MIIBIjANBgkqhki...`                                        | Wrapped with SPKI headers automatically |
| Single-line base64      | Header + all base64 on one line + footer                    | Re-wrapped to 64-char lines             |

Normalization is applied **on every call**. Invalid keys produce a descriptive `error` value at verification time (no panics, no eager construction-time validation).

## Two Verification APIs

### Standalone Function — `pancake.VerifyWebhook()`

Best for simple setups where you don't need the SDK client (e.g. a small webhook-only service). Uses env vars and built-in keys by default.

```go
import pancake "github.com/waffo-com/waffo-pancake-sdk-go"

event, err := pancake.VerifyWebhook(body, sig, nil) // built-in keys
event, err := pancake.VerifyWebhook(body, sig, &pancake.VerifyWebhookOptions{
    Environment: pancake.EnvironmentProd,
}) // explicit env
event, err := pancake.VerifyWebhook(body, sig, &pancake.VerifyWebhookOptions{
    PublicKey: customKey,
}) // per-call key
```

There is also a generic helper that unmarshals `event.Data` into a typed struct in one step:

```go
event, err := pancake.VerifyWebhookTyped[pancake.WebhookEventData](body, sig, nil)
// event.Data is pancake.WebhookEventData (no manual json.Unmarshal needed)

// You can also use your own struct if you only care about a subset of fields:
type MyOrderData struct {
    OrderID    string `json:"orderId"`
    BuyerEmail string `json:"buyerEmail"`
}
event, err := pancake.VerifyWebhookTyped[MyOrderData](body, sig, nil)
```

### Client Instance Method — `client.Webhooks.Verify()`

Best when you already have a `*pancake.Client`. Automatically injects config-level keys into the resolution chain.

```go
client, _ := pancake.New(pancake.Config{
    MerchantID: "MER_...",
    PrivateKey: "...",
    WebhookPublicKey: pancake.WebhookPublicKeys{
        Test: testKey,
        Prod: prodKey,
    },
})

event, err := client.Webhooks.Verify(body, sig, nil) // auto-detect with config keys
event, err := client.Webhooks.Verify(body, sig, &pancake.VerifyWebhookOptions{
    Environment: pancake.EnvironmentTest,
}) // explicit env
event, err := client.Webhooks.Verify(body, sig, &pancake.VerifyWebhookOptions{
    PublicKey: oneOff,
}) // per-call override
```

Both APIs share the same underlying verification logic and resolution chain.

## Event Payload

### `pancake.WebhookEvent`

The verified envelope returned by `VerifyWebhook`. `Data` is held as `json.RawMessage` so callers can unmarshal into either the canonical `WebhookEventData` struct or a custom struct (via `VerifyWebhookTyped`).

| Field       | Type                  | Description                                                         |
| ----------- | --------------------- | ------------------------------------------------------------------- |
| `ID`        | `string`              | Delivery record unique ID (UUID) — use for idempotent deduplication |
| `Timestamp` | `string`              | Event timestamp (ISO 8601 UTC)                                      |
| `EventType` | `string`              | Event type (e.g. `"order.completed"`)                               |
| `EventID`   | `string`              | Business event ID (e.g. payment ID)                                 |
| `StoreID`   | `string`              | Store ID the event belongs to                                       |
| `StoreName` | `string`              | Store name                                                          |
| `Mode`      | `pancake.Environment` | Environment (`"test"` or `"prod"`)                                  |
| `Data`      | `json.RawMessage`     | Raw event payload (unmarshal into `WebhookEventData` or your own struct) |

### `pancake.TypedWebhookEvent[T]`

The typed envelope produced by `VerifyWebhookTyped[T]`. Same fields as `WebhookEvent` except `Data T` is the caller-provided struct, fully unmarshaled.

### `pancake.WebhookEventData`

All events include the **Order**, **Amount**, and **Product** sections. Additional sections are conditionally present based on event type.

**Order fields** (always present):

| Field                            | Type                | Required | Description                                                                                                                                              |
| -------------------------------- | ------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `OrderID`                        | `string`            | Yes      | Associated order ID                                                                                                                                      |
| `OrderStatus`                    | `*string`           | No       | Order status (e.g., `"completed"`, `"active"`, `"canceling"`)                                                                                            |
| `BuyerEmail`                     | `string`            | Yes      | Customer email address                                                                                                                                      |
| `MerchantProvidedBuyerIdentity`  | `*string`           | No       | Merchant-provided customer identity from checkout session                                                                                                   |
| `OrderMerchantExternalID`        | `*string`           | No       | Order business-side identifier set at checkout creation (max 128 chars). Present on order / payment / subscription events and on refund events (inherited from the originating order). |
| `RefundTicketMerchantExternalID` | `*string`           | No       | Refund-ticket business-side identifier set at refund-ticket creation. **Only present on `refund.*` events**; coexists with `OrderMerchantExternalID` on the same refund payload. |
| `Currency`                       | `string`            | Yes      | Currency code (ISO 4217)                                                                                                                                 |
| `BillingDetail`                  | `map[string]any`    | No       | Billing/shipping address (structured map)                                                                                                                |
| `OrderMetadata`                  | `map[string]string` | No       | Order-level metadata from checkout session (flat key-value pairs)                                                                                        |

**Amount fields** (always present):

| Field       | Type       | Required | Description                                                                       |
| ----------- | ---------- | -------- | --------------------------------------------------------------------------------- |
| `Amount`    | `string`   | Yes      | Amount in display format (e.g., `"29.00"` for $29.00 USD, `"4500"` for ¥4500 JPY) |
| `TaxAmount` | `string`   | Yes      | Tax amount in display format (e.g., `"2.90"`)                                     |
| `TaxRate`   | `*float64` | No       | Tax rate as decimal (e.g., `0.1` for 10%)                                         |
| `TaxName`   | `*string`  | No       | Tax name (e.g., `"Consumption Tax"`)                                              |
| `Subtotal`  | `*string`  | No       | Subtotal as display string (before tax)                                           |
| `Total`     | `*string`  | No       | Total as display string (after tax)                                               |

**Product fields** (always present):

| Field                | Type                | Required | Description                                                   |
| -------------------- | ------------------- | -------- | ------------------------------------------------------------- |
| `ProductName`        | `string`            | Yes      | Product name                                                  |
| `ProductDescription` | `*string`           | No       | Product description                                           |
| `ProductMetadata`    | `map[string]string` | No       | Product-level metadata set when creating/updating the product |

**Payment fields** (present for `order.completed`, `subscription.payment_succeeded`):

| Field                  | Type      | Description                                        |
| ---------------------- | --------- | -------------------------------------------------- |
| `PaymentID`            | `*string` | Payment ID                                         |
| `PaymentStatus`        | `*string` | Payment status (e.g., `"succeeded"`, `"failed"`)   |
| `PaymentMethod`        | `*string` | Payment method type (e.g., `"card"`)               |
| `PaymentLast4`         | `*string` | Last 4 digits of payment instrument                |
| `PaymentFailureReason` | `*string` | Payment failure reason (present when failed)       |
| `PaymentDate`          | `*string` | Payment date (ISO 8601 date, e.g., `"2026-04-18"`) |

**Subscription fields** (present for `subscription.*` events):

| Field                | Type      | Description                                                           |
| -------------------- | --------- | --------------------------------------------------------------------- |
| `BillingPeriod`      | `*string` | Billing period: `"weekly"`, `"monthly"`, `"quarterly"`, `"yearly"`    |
| `CurrentPeriodStart` | `*string` | Current billing period start date (ISO 8601)                          |
| `CurrentPeriodEnd`   | `*string` | Current billing period end date (ISO 8601)                            |
| `CanceledAt`         | `*string` | Subscription cancellation timestamp (ISO 8601, present when canceled) |

**Refund fields** (present for `refund.succeeded`, `refund.failed`):

| Field             | Type      | Description                               |
| ----------------- | --------- | ----------------------------------------- |
| `RefundStatus`    | `*string` | Refund status (`"succeeded"`, `"failed"`) |
| `RefundReason`    | `*string` | Refund reason                             |
| `RefundCreatedAt` | `*string` | Refund creation timestamp (ISO 8601)      |

## Event Types

Use `pancake.WebhookEventType` constants. When matching against `event.EventType` (a plain `string`), convert with `string(...)`:

```go
switch event.EventType {
case string(pancake.WebhookEventTypeOrderCompleted):
    // ...
case string(pancake.WebhookEventTypeRefundSucceeded):
    // ...
}
```

| Constant                                            | String                           | Trigger                                                         |
| --------------------------------------------------- | -------------------------------- | --------------------------------------------------------------- |
| `WebhookEventTypeOrderCompleted`                    | `order.completed`                | One-time order first payment succeeded                          |
| `WebhookEventTypeSubscriptionActivated`             | `subscription.activated`         | New subscription activated                                      |
| `WebhookEventTypeSubscriptionPaymentSucceeded`      | `subscription.payment_succeeded` | Subscription renewal payment succeeded                          |
| `WebhookEventTypeSubscriptionCanceling`             | `subscription.canceling`         | Customer initiated cancellation (expires at end of billing period) |
| `WebhookEventTypeSubscriptionUncanceled`            | `subscription.uncanceled`        | Customer withdrew cancellation request                             |
| `WebhookEventTypeSubscriptionUpdated`               | `subscription.updated`           | Subscription product changed (upgrade/downgrade)                |
| `WebhookEventTypeSubscriptionCanceled`              | `subscription.canceled`          | Subscription fully terminated                                   |
| `WebhookEventTypeSubscriptionPastDue`               | `subscription.past_due`          | Renewal payment failed (past due)                               |
| `WebhookEventTypeRefundSucceeded`                   | `refund.succeeded`               | Refund completed successfully                                   |
| `WebhookEventTypeRefundFailed`                      | `refund.failed`                  | Refund failed                                                   |

## Key Rotation & Migration

### Scenario: Built-in hardcoded keys are no longer valid

This can happen when Waffo rotates its platform key pair, or when you switch to a self-hosted deployment with custom keys.

**Minimum-effort fix — set environment variables (zero code change):**

```bash
# Just add these to your hosting environment:
WAFFO_WEBHOOK_PROD_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----\nMIIBIjAN..."
WAFFO_WEBHOOK_TEST_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----\nMIIBIjAN..."
```

The SDK automatically checks env vars before falling back to hardcoded keys. Your existing `pancake.VerifyWebhook(body, sig, nil)` or `client.Webhooks.Verify(body, sig, nil)` calls continue to work without any code change.

### Scenario: Gradual key rotation

When rotating keys, the old key remains valid for a transition period:

```go
// During transition: both old and new keys work
// The SDK auto-detect tries multiple keys, so both will be accepted

// After transition: update the env var to the new key
// Old signed events will fail verification — this is expected
```

### Choosing the right level

| Situation                   | Recommended Level  | Why                                     |
| --------------------------- | ------------------ | --------------------------------------- |
| Standard Waffo Pancake user | Level 6 (default)  | Built-in keys just work, zero config    |
| Built-in keys expired       | Level 4 (env var)  | No code changes, set env var and done   |
| Self-hosted deployment      | Level 2/3 (config) | Custom keys are part of your app config |
| Testing a new key           | Level 1 (per-call) | One-off override, no permanent change   |
| CI/CD with different keys   | Level 4 (env var)  | Each environment sets its own env var   |

## Error Handling

`VerifyWebhook` returns plain `error` values (created with `errors.New` / `fmt.Errorf`) for all signature, timestamp, and key-resolution failures — there are no sentinel error variables to compare against. In practice you only need to distinguish "did verification succeed" from "did it fail", so a single `err != nil` check is enough:

```go
event, err := pancake.VerifyWebhook(string(body), sig, nil)
if err != nil {
    // All cases — bad signature, expired timestamp, malformed header, missing key —
    // are surfaced as an error. Log err.Error() for diagnostics, return 401.
    log.Printf("webhook verify failed: %v", err)
    http.Error(w, "Invalid signature", http.StatusUnauthorized)
    return
}
```

If you need to differentiate a *transport* error (e.g., the SDK could not parse a configured PEM) from a *signature* error during debugging, inspect the error message — message prefixes are stable:

- `missing X-Waffo-Signature header`
- `malformed X-Waffo-Signature header: missing t or v1`
- `invalid timestamp in X-Waffo-Signature header`
- `webhook timestamp outside tolerance window (possible replay attack)`
- `invalid webhook signature (custom key)`
- `invalid webhook signature (<env> key)`
- `invalid webhook signature (tried both prod and test keys)`
- `decode webhook event: ...`

Note that `*pancake.Error` (the typed API error used by REST/GraphQL calls) is **not** produced by `VerifyWebhook` — `errors.As(err, &perr)` will always be false on webhook verification failures.

## Retry Mechanism

When delivery fails (non-2xx response or timeout), the system automatically retries using **exponential backoff** (managed by the underlying message queue). Default: 3 retries.

| Delivery Status | Description                               |
| --------------- | ----------------------------------------- |
| `pending`       | Created, waiting for delivery or retrying |
| `success`       | Delivery successful (server returned 2xx) |
| `failed`        | All retries exhausted, final failure      |

You can view each delivery's status, HTTP status code, and response content in the dashboard's Webhook logs.

> **Note**: The same business event (same `EventType` + `EventID`) creates only one delivery record and won't be duplicated. However, the same delivery may arrive multiple times due to retries — always deduplicate using `event.ID`.

## Best Practices

1. **Respond quickly** — Return 200 immediately and process the event asynchronously (e.g., enqueue it onto a channel or goroutine). Waffo retries on timeout.
2. **Deduplicate** — Use `event.ID` (delivery record UUID) as an idempotency key to handle redeliveries.
3. **Verify all events** — Always call `pancake.VerifyWebhook` before processing. Never trust unverified payloads.
4. **Use raw body** — The signature is computed over the raw request body. `io.ReadAll(r.Body)` before any JSON decoding; do not let middleware decode-and-re-encode the body.
5. **Specify environment when known** — If your endpoint only receives test or prod events, pass `Environment: pancake.EnvironmentTest` or `pancake.EnvironmentProd` to skip unnecessary key attempts and get clearer error messages.
6. **Use env vars for secrets** — Prefer `WAFFO_WEBHOOK_PROD_PUBLIC_KEY` env vars over hardcoding keys in source code. The SDK reads them automatically via `os.Getenv`.
7. **Key rotation** — During rotation, temporarily use `opts.PublicKey` per-call to test the new key, then update config/env vars once confirmed.
