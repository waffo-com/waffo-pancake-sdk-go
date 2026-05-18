# github.com/waffo-com/waffo-pancake-sdk-go

Go SDK for the Waffo Pancake Merchant of Record (MoR) payment platform.

- Zero runtime dependencies, Go >= 1.22
- Automatic RSA-SHA256 request signing with deterministic idempotency keys
- Full type definitions (14 enums, 40+ structs)
- Webhook verification with embedded public keys (test/prod)
- Feature parity with [`@waffo/pancake-ts@0.7.x`](https://www.npmjs.com/package/@waffo/pancake-ts)

## Installation

```bash
go get github.com/waffo-com/waffo-pancake-sdk-go
```

Requires Go 1.22 or newer.

## Quick Start

> Most merchants create stores and products in the
> [Dashboard](https://pancake.waffo.ai/dashboard). The SDK is primarily used
> for checkout integration — redirecting buyers from your site to the Waffo
> checkout page.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/waffo-com/waffo-pancake-sdk-go"
)

func main() {
    client, err := pancake.New(pancake.Config{
        MerchantID: os.Getenv("WAFFO_MERCHANT_ID"), // "MER_..." Short ID
        PrivateKey: os.Getenv("WAFFO_PRIVATE_KEY"), // RSA PEM
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Create a checkout session — one call handles token + session + URL.
    result, err := client.Checkout.Authenticated.Create(ctx, pancake.AuthenticatedCheckoutParams{
        CreateCheckoutSessionParams: pancake.CreateCheckoutSessionParams{
            ProductID:  "PROD_xxx",
            Currency:   "USD",
            BuyerEmail: pancake.Ptr("customer@example.com"),
        },
        BuyerIdentity: "user-123", // your user's identity
    })
    if err != nil {
        log.Fatal(err)
    }

    // Redirect the buyer to result.CheckoutURL (includes #token=...).
    fmt.Println(result.CheckoutURL)
}
```

## Configuration

| Field              | Type                          | Required | Description                                                       |
| ------------------ | ----------------------------- | -------- | ----------------------------------------------------------------- |
| `MerchantID`       | `string`                      | yes      | Merchant ID in `MER_{base62}` format                              |
| `PrivateKey`       | `string`                      | yes      | RSA private key in PEM format (auto-normalized)                   |
| `BaseURL`          | `string`                      | no       | API base URL override (default: `https://api.waffo.ai`)           |
| `HTTPClient`       | `*http.Client`                | no       | Custom HTTP client (default: `http.DefaultClient`)                |
| `WebhookPublicKey` | `pancake.WebhookPublicKeys`   | no       | Custom webhook public key(s) — overrides the built-in defaults    |

The SDK auto-normalizes PEM input variants: standard PEM, PKCS#1, PKCS#8,
literal `\n` from env vars, Windows line endings, and raw base64.

## Checkout Integration

Two flows, mirroring the TypeScript SDK:

| Mode          | Method                                | Buyer identity     | Form state | Use case                                 |
| ------------- | ------------------------------------- | ------------------ | ---------- | ---------------------------------------- |
| Authenticated | `Checkout.Authenticated.Create(...)`  | Merchant-provided  | Pre-filled | Sites with user accounts (recommended)   |
| Anonymous     | `Checkout.Anonymous.Create(...)`      | None               | Empty      | Template stores, one-time purchase links |

```go
// Authenticated (recommended): identity binds the order to a stable
// merchant-controlled ID even if the buyer changes the email on the form.
res, err := client.Checkout.Authenticated.Create(ctx, pancake.AuthenticatedCheckoutParams{
    CreateCheckoutSessionParams: pancake.CreateCheckoutSessionParams{
        ProductID:  "PROD_...",
        Currency:   "USD",
        BuyerEmail: pancake.Ptr("customer@example.com"),
        PriceSnapshot: &pancake.PriceInfo{
            Amount:      "19.99",
            TaxCategory: pancake.TaxCategoryDigitalGoods,
        },
    },
    BuyerIdentity: "user-123",
})

// Anonymous: buyer fills the form themselves.
res, err := client.Checkout.Anonymous.Create(ctx, pancake.AnonymousCheckoutParams{
    ProductID: "PROD_...",
    Currency:  "USD",
})
```

Opening the URL in a new tab is recommended so buyers can return to your site
without losing page state.

## Webhook Verification

> Use the **raw** request body. Parsing and re-serializing breaks the
> signature.

### Standalone function

```go
import "io"

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    sig := r.Header.Get("X-Waffo-Signature")

    event, err := pancake.VerifyWebhook(string(body), sig, nil)
    if err != nil {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }
    w.WriteHeader(http.StatusOK)

    switch pancake.WebhookEventType(event.EventType) {
    case pancake.WebhookEventTypeOrderCompleted:
        var data pancake.WebhookEventData
        _ = json.Unmarshal(event.Data, &data)
        fmt.Println("Order completed for", data.BuyerEmail)
    }
}
```

### Typed verification

```go
event, err := pancake.VerifyWebhookTyped[pancake.WebhookEventData](
    string(body),
    sig,
    &pancake.VerifyWebhookOptions{Environment: pancake.EnvironmentProd},
)
// event.Data is pancake.WebhookEventData (struct, not raw bytes)
```

### Client-instance method

```go
client, _ := pancake.New(pancake.Config{
    MerchantID: "MER_...",
    PrivateKey: privateKey,
    WebhookPublicKey: pancake.WebhookPublicKeys{
        Test: os.Getenv("WAFFO_TEST_PUB_KEY"),
        Prod: os.Getenv("WAFFO_PROD_PUB_KEY"),
    },
})
event, err := client.Webhooks.Verify(string(body), sig, nil)
```

### Public key resolution chain

For each environment, public keys are resolved in priority order:

1. `VerifyWebhookOptions.PublicKey` — per-call override
2. `Config.WebhookPublicKey` (`Shared` first, then per-env)
3. `WAFFO_WEBHOOK_TEST_PUBLIC_KEY` / `WAFFO_WEBHOOK_PROD_PUBLIC_KEY` env var
4. `WAFFO_WEBHOOK_PUBLIC_KEY` env var
5. Built-in hardcoded PEM key

Replay protection: timestamps outside a 5-minute window are rejected. Set
`VerifyWebhookOptions.ToleranceMS` to a negative value to disable.

## Buyer Self-Service

```go
// Issue a session token on your backend.
tok, err := client.Auth.IssueSessionToken(ctx, pancake.IssueSessionTokenParams{
    StoreID:       pancake.Ptr("STO_..."),
    BuyerIdentity: "user-123",
})

// Hand off to the buyer session.
buyer := client.Buyer(tok.Token)

// Cancel a subscription.
res, err := buyer.CancelSubscription(ctx, pancake.CancelSubscriptionParams{
    OrderID: "ORD_...",
})

// Submit a refund request.
refund, err := buyer.CreateRefundTicket(ctx, pancake.CreateRefundTicketParams{
    PaymentID: "PAY_...",
    Reason:    "Product not as described",
    RequestedAmount: pancake.RequestedAmount{
        Amount:   "29.00",
        Currency: "USD",
    },
})
```

The token is scoped to the issuing store and buyer identity. TTL is 5
minutes and auto-refreshes on each call.

## GraphQL

Both raw and typed access are supported. The raw form is useful for one-off
queries; the typed form is type-safe.

```go
// Raw — Data is json.RawMessage; unmarshal into your own struct.
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query { stores { id name status } }`,
})

var data struct {
    Stores []pancake.Store `json:"stores"`
}
_ = json.Unmarshal(resp.Data, &data)

// Typed — Data is your struct.
type StoresQuery struct {
    Stores []pancake.Store `json:"stores"`
}
typed, err := pancake.GraphQLQuery[StoresQuery](ctx, client, pancake.GraphQLParams{
    Query: `query { stores { id name status } }`,
})
fmt.Println(typed.Data.Stores[0].Name)
```

For buyer-scoped queries, use `buyer.GraphQL.Query` or
`pancake.BuyerGraphQLQuery[T]`.

## Warnings (Migration Notices)

Every successful REST action and GraphQL query may carry a `Warnings` slice alongside the data. Warnings describe non-fatal advisories the server wants you to act on — typically deprecated parameters, fields scheduled for removal, or new APIs you should switch to. Each `Notice` carries `Message` (human-readable), `Layer` (which service produced it), and `AIHint` (a structured migration instruction aimed at LLM consumers).

```go
// REST action — warnings sit on the typed Result struct
res, err := client.Stores.Update(ctx, pancake.UpdateStoreParams{
    ID: "STO_xxx",
    // ... may carry deprecated fields the server warns about
})
if err != nil { /* handle */ }
for _, w := range res.Warnings {
    log.Printf("[%s] %s — %s", w.Layer, w.Message, w.AIHint)
    // e.g. Layer=store, AIHint="Switch to client.Webhooks.Add / Update / Remove"
}

// GraphQL — warnings sit on the envelope alongside Data and Errors
resp, _ := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query { stores { id } }`,
})
for _, w := range resp.Warnings {
    log.Printf("%s — %s", w.Message, w.AIHint)
}
```

**LLM/agent consumers**: always inspect `AIHint` on every warning — it is the canonical migration instruction (Go module path, version, method name, endpoint path) the platform team intends for you to follow when the underlying API evolves.

## Programmatic Store, Product, and Webhook Management

```go
// Stores
storeRes, _ := client.Stores.Create(ctx, pancake.CreateStoreParams{Name: "My Store"})

// Products
prodRes, _ := client.OnetimeProducts.Create(ctx, pancake.CreateOnetimeProductParams{
    StoreID: storeRes.Store.ID,
    Name:    "E-Book",
    Prices: pancake.Prices{
        "USD": {Amount: "29.00", TaxCategory: pancake.TaxCategoryDigitalGoods},
    },
})
_, _ = client.OnetimeProducts.Publish(ctx, pancake.PublishOnetimeProductParams{ID: prodRes.Product.ID})

// Subscription products
subRes, _ := client.SubscriptionProducts.Create(ctx, pancake.CreateSubscriptionProductParams{
    StoreID:       storeRes.Store.ID,
    Name:          "Pro Plan",
    BillingPeriod: pancake.BillingPeriodMonthly,
    Prices: pancake.Prices{
        "USD": {Amount: "9.99", TaxCategory: pancake.TaxCategorySaaS},
    },
})

// Webhooks
_, _ = client.Webhooks.Add(ctx, pancake.AddWebhookParams{
    StoreID:  storeRes.Store.ID,
    Channel:  pancake.WebhookChannelHTTP,
    URL:      "https://example.com/webhooks",
    Events:   []pancake.WebhookEventType{pancake.WebhookEventTypeOrderCompleted},
    TestMode: false,
})
```

To list configured webhooks, query GraphQL `Store.storeWebhooks` via
`client.GraphQL.Query`.

## Error Handling

```go
import "errors"

if _, err := client.Stores.Create(ctx, pancake.CreateStoreParams{Name: ""}); err != nil {
    var perr *pancake.Error
    if errors.As(err, &perr) {
        fmt.Println(perr.Status)            // 400
        fmt.Println(perr.Errors[0].Message) // "name cannot be empty"
        fmt.Println(perr.Errors[0].Layer)   // pancake.ErrorLayerSDK
    }
}
```

Client-side validation failures use `Layer: ErrorLayerSDK` so they can be
distinguished from server-returned errors.

## Resource Reference

| Field                                | Methods                                                                                                                                  |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `client.Auth`                        | `IssueSessionToken`                                                                                                                      |
| `client.Stores`                      | `Create`, `Update`, `Delete`                                                                                                             |
| `client.StoreMerchants`              | `Add`, `Remove`, `UpdateRole` (coming soon — endpoints return 501)                                                                       |
| `client.OnetimeProducts`             | `Create`, `Update`, `Publish`, `UpdateStatus`                                                                                            |
| `client.SubscriptionProducts`        | `Create`, `Update`, `Publish`, `UpdateStatus`                                                                                            |
| `client.SubscriptionProductGroups`   | `Create`, `Update`, `Delete`, `Publish`                                                                                                  |
| `client.Orders`                      | `CancelSubscription`                                                                                                                     |
| `client.Checkout`                    | `CreateSession`, `Anonymous.Create`, `Authenticated.Create`                                                                              |
| `client.GraphQL`                     | `Query` (also `pancake.GraphQLQuery[T]`)                                                                                                 |
| `client.Webhooks`                    | `Add`, `Update`, `Remove`, `Verify` (also `pancake.VerifyWebhook` / `pancake.VerifyWebhookTyped[T]`)                                     |
| `client.Buyer(token)`                | `CancelSubscription`, `CancelOnetimeOrder`, `ReactivateSubscription`, `CreateRefundTicket`, `ResubmitRefundTicket`, `GraphQL.Query`      |

## Optional fields

| TypeScript signature             | Go representation                                                       |
| -------------------------------- | ----------------------------------------------------------------------- |
| `description?: string`           | `Description *string \`json:"description,omitempty"\``                  |
| `logo?: string \| null`          | `Logo *pancake.Nullable[string] \`json:"logo,omitempty"\``              |
| `prices: Record<string, ...>`    | `Prices pancake.Prices`                                                 |

Helpers: `pancake.Ptr(v)`, `pancake.NullValuePtr(v)`, `pancake.ExplicitNullPtr[T]()`.

## Development

```bash
go test ./...
go test -race -cover ./...
go vet ./...
golangci-lint run
```

See `examples/` for runnable sample programs.

## License

MIT
