# API Reference

Complete reference for all `github.com/waffo-com/waffo-pancake-sdk-go` resources, parameters, and return types.

> **Conventions**:
>
> - All amounts are in the **smallest currency unit** (e.g. 999 = $9.99 USD, 4500 = ¥4500 JPY)
> - All timestamps are **ISO 8601 UTC** strings
> - Product updates follow **immutable versioning** — only provided fields are updated (omitted fields are preserved), each update creates a new version, skipped if content is unchanged
> - The **publish** flow promotes a test version to production
> - Every method takes `ctx context.Context` first and returns `(*Result, error)`
> - Every result struct carries a `Warnings []pancake.Notice` slice (the v0.2.0 unified envelope) — inspect it for non-fatal advisories and migration hints
> - Optional scalar fields are `*T` (use `pancake.Ptr(v)` to construct); tri-state nullable fields are `*pancake.Nullable[T]` (use `pancake.NullValuePtr(v)` / `pancake.ExplicitNullPtr[T]()`)
> - Every write method accepts trailing `pancake.RequestOption` values; the only one today is `pancake.WithIdempotencyKey`. No key is sent unless you pass it — see [Idempotency](#idempotency)

---

## Auth

### `client.Auth.IssueSessionToken(ctx, params)`

Issue a customer session token (JWT) for storefront authentication.

```go
// With StoreID
tok, err := client.Auth.IssueSessionToken(ctx, pancake.IssueSessionTokenParams{
    StoreID:       pancake.Ptr("STO_xxx"),
    BuyerIdentity: "customer@example.com",
})

// With ProductID (server derives StoreID from the product)
tok, err := client.Auth.IssueSessionToken(ctx, pancake.IssueSessionTokenParams{
    ProductID:     pancake.Ptr("PROD_xxx"),
    BuyerIdentity: "customer@example.com",
})
fmt.Println(tok.Token, tok.ExpiresAt)
```

**Parameters `IssueSessionTokenParams`**:

| Field           | Type      | Required | Description                                                                                      |
| --------------- | --------- | -------- | ------------------------------------------------------------------------------------------------ |
| `StoreID`       | `*string` | No       | Store ID (at least one of `StoreID` / `ProductID` required)                                      |
| `ProductID`     | `*string` | No       | Product ID (at least one of `StoreID` / `ProductID` required; server derives store from product) |
| `BuyerIdentity` | `string`  | Yes      | Customer identity (email or merchant-defined identifier)                                            |

**Returns `*SessionToken`**:

| Field       | Type              | Description           |
| ----------- | ----------------- | --------------------- |
| `Token`     | `string`          | JWT token string      |
| `ExpiresAt` | `string`          | Token expiration time |
| `Warnings`  | `[]pancake.Notice` | Migration notices     |

### `client.Auth.CreateCustomerPortalLink(ctx, params)`

Create a link that signs a customer into your store's Pancake customer portal without email verification. Calls `issue-session-token` with `purpose: "portal"` and appends the returned token to the portal URL as a fragment (`#token=...`).

```go
link, err := client.Auth.CreateCustomerPortalLink(ctx, pancake.CreateCustomerPortalLinkParams{
    StoreID:       "STO_xxx",
    BuyerIdentity: user.ID, // from YOUR authenticated session
})
// 302-redirect the browser to link.PortalURL
// => "https://pancake.waffo.ai/consumer/portal/store/STO_xxx#token=eyJ..."
```

**Parameters `CreateCustomerPortalLinkParams`**:

| Field           | Type     | Required | Description                                                                                    |
| --------------- | -------- | -------- | ---------------------------------------------------------------------------------------------- |
| `StoreID`       | `string` | Yes      | Store ID (`STO_xxx`)                                                                           |
| `BuyerIdentity` | `string` | Yes      | Customer identity in your system — the same value used for `Checkout.Authenticated.Create`     |

**Returns `*CustomerPortalLink`**:

| Field       | Type               | Description                                                             |
| ----------- | ------------------ | ----------------------------------------------------------------------- |
| `PortalURL` | `string`           | Portal URL with the session token appended as a fragment (`#token=...`) |
| `ExpiresAt` | `string`           | Session token expiration time                                           |
| `Warnings`  | `[]pancake.Notice` | Migration notices                                                       |

- **Environment** is the API Key's environment (test key → test portal, prod key → prod portal); it is not a parameter.
- **Security**: `StoreID` and `BuyerIdentity` must come from your backend's authenticated session, never from browser input. `PortalURL` carries a bearer token — redirect immediately, send `Cache-Control: no-store` and `Referrer-Policy: no-referrer` on the redirect, and do not log it.
- **Errors**: invalid `StoreID` / empty or blank `BuyerIdentity` return `*pancake.Error` (400, `ErrorLayerSDK`) before any request; a response without `portalUrl` or `token` returns `*pancake.Error` with `ErrorLayerSDK`.

---

## Stores

### `client.Stores.Create(ctx, params)`

Create a store. The URL slug is auto-generated from the name.

```go
res, err := client.Stores.Create(ctx, pancake.CreateStoreParams{Name: "My Store"})
fmt.Println(res.Store.ID)
```

**Parameters `CreateStoreParams`**:

| Field  | Type     | Required | Description                                         |
| ------ | -------- | -------- | --------------------------------------------------- |
| `Name` | `string` | Yes      | Store name (1–48 characters, trimmed automatically) |

**Returns `*CreateStoreResult`**: `{ Store, Warnings }`

### `client.Stores.Update(ctx, params)`

Update store settings including notification preferences and checkout page styling. Webhook endpoints are managed separately via `client.Webhooks.Add` / `Update` / `Remove`.

```go
res, err := client.Stores.Update(ctx, pancake.UpdateStoreParams{
    ID:   "STO_xxx",
    Name: pancake.Ptr("Updated Name"),
    // Merchant-writable: all the Notify* toggles plus EmailUpcomingCharge. Every
    // other Email* toggle is platform-managed and silently dropped server-side if
    // included.
    NotificationSettings: pancake.NullValuePtr(pancake.NotificationSettings{
        NotifyNewOrders:            pancake.Ptr(true),
        NotifyNewSubscriptions:     pancake.Ptr(false),
        NotifySubscriptionCanceled: pancake.Ptr(true),
        NotifyChargeback:           pancake.Ptr(true),
        NotifyRefundSucceeded:      pancake.Ptr(true),
        // Stop reminding this store's buyers before a renewal is charged.
        EmailUpcomingCharge:        pancake.Ptr(false),
    }),
    CheckoutSettings: pancake.NullValuePtr(pancake.CheckoutSettings{
        Light: pancake.CheckoutThemeSettings{
            CheckoutColorPrimary:    "#000000",
            CheckoutColorBackground: "#ffffff",
            CheckoutColorCard:       "#f5f5f5",
            CheckoutColorText:       "#000000",
            CheckoutBorderRadius:    "8px",
        },
        Dark: pancake.CheckoutThemeSettings{
            CheckoutColorPrimary:    "#ffffff",
            CheckoutColorBackground: "#1a1a1a",
            CheckoutColorCard:       "#2a2a2a",
            CheckoutColorText:       "#ffffff",
            CheckoutBorderRadius:    "8px",
        },
    }),
})
```

**Parameters `UpdateStoreParams`**:

| Field                  | Type                                     | Required | Description                                                              |
| ---------------------- | ---------------------------------------- | -------- | ------------------------------------------------------------------------ |
| `ID`                   | `string`                                 | Yes      | Store ID                                                                 |
| `Name`                 | `*string`                                | No       | Store name (1–48 characters, no control characters)                      |
| `Status`               | `*pancake.EntityStatus`                  | No       | Store status                                                             |
| `Logo`                 | `*pancake.Nullable[string]`              | No       | Logo (Base64 encoded image); `ExplicitNullPtr[string]()` clears          |
| `NotificationSettings` | `*pancake.Nullable[NotificationSettings]` | No       | Notification preferences. Merchant-writable: the `Notify*` toggles plus `EmailUpcomingCharge`; the other `Email*` toggles are platform-managed and dropped with a `warnings` entry |
| `CheckoutSettings`     | `*pancake.Nullable[CheckoutSettings]`    | No       | Checkout page theme (light/dark)                                         |

> `SupportEmail` and `Website` are not writable through this endpoint. They are derived from ownership verification and are set only by the flows that prove it: email code binding and domain verification, or KYB approval. Both remain readable on `pancake.Store`.

> Webhook endpoints are managed via `client.Webhooks.Add` / `Update` / `Remove`. The `webhookSettings` field on the TypeScript SDK is deprecated and not exposed on the Go SDK.

**Returns `*UpdateStoreResult`** (alias of `CreateStoreResult`): `{ Store, Warnings }`

### `client.Stores.Delete(ctx, params)`

Soft-delete a store. Only the store owner can perform this operation.

```go
res, err := client.Stores.Delete(ctx, pancake.DeleteStoreParams{ID: "STO_xxx"})
```

**Parameters `DeleteStoreParams`**:

| Field | Type     | Required | Description |
| ----- | -------- | -------- | ----------- |
| `ID`  | `string` | Yes      | Store ID    |

**Returns `*DeleteStoreResult`** (alias of `CreateStoreResult`): `{ Store, Warnings }`

---

## Store Merchants

> Coming soon — endpoints currently return 501.

### `client.StoreMerchants.Add(ctx, params)`

Add a merchant to a store with a specified role.

```go
res, err := client.StoreMerchants.Add(ctx, pancake.AddMerchantParams{
    StoreID: "STO_xxx",
    Email:   "member@example.com",
    Role:    "admin",
})
```

**Parameters `AddMerchantParams`**:

| Field     | Type     | Required | Description                  |
| --------- | -------- | -------- | ---------------------------- |
| `StoreID` | `string` | Yes      | Store ID                     |
| `Email`   | `string` | Yes      | Merchant email               |
| `Role`    | `string` | Yes      | `"admin"` or `"member"`      |

**Returns `*AddMerchantResult`**:

| Field        | Type               | Description          |
| ------------ | ------------------ | -------------------- |
| `StoreID`    | `string`           | Store ID             |
| `MerchantID` | `string`           | Merchant ID          |
| `Email`      | `string`           | Merchant email       |
| `Role`       | `string`           | Assigned role        |
| `Status`     | `string`           | Membership status    |
| `AddedAt`    | `string`           | Timestamp when added |
| `Warnings`   | `[]pancake.Notice` | Migration notices    |

### `client.StoreMerchants.Remove(ctx, params)`

Remove a merchant from a store.

```go
res, err := client.StoreMerchants.Remove(ctx, pancake.RemoveMerchantParams{
    StoreID:    "STO_xxx",
    MerchantID: "MER_xxx",
})
```

**Parameters `RemoveMerchantParams`**:

| Field        | Type     | Required | Description |
| ------------ | -------- | -------- | ----------- |
| `StoreID`    | `string` | Yes      | Store ID    |
| `MerchantID` | `string` | Yes      | Merchant ID |

**Returns `*RemoveMerchantResult`**:

| Field       | Type               | Description            |
| ----------- | ------------------ | ---------------------- |
| `Message`   | `string`           | Operation message      |
| `RemovedAt` | `string`           | Timestamp when removed |
| `Warnings`  | `[]pancake.Notice` | Migration notices      |

### `client.StoreMerchants.UpdateRole(ctx, params)`

Update a merchant's role within a store.

```go
res, err := client.StoreMerchants.UpdateRole(ctx, pancake.UpdateRoleParams{
    StoreID:    "STO_xxx",
    MerchantID: "MER_xxx",
    Role:       "member",
})
```

**Parameters `UpdateRoleParams`**:

| Field        | Type     | Required | Description             |
| ------------ | -------- | -------- | ----------------------- |
| `StoreID`    | `string` | Yes      | Store ID                |
| `MerchantID` | `string` | Yes      | Merchant ID             |
| `Role`       | `string` | Yes      | `"admin"` or `"member"` |

**Returns `*UpdateRoleResult`**:

| Field        | Type               | Description            |
| ------------ | ------------------ | ---------------------- |
| `StoreID`    | `string`           | Store ID               |
| `MerchantID` | `string`           | Merchant ID            |
| `Role`       | `string`           | Updated role           |
| `UpdatedAt`  | `string`           | Timestamp when updated |
| `Warnings`   | `[]pancake.Notice` | Migration notices      |

---

## Onetime Products

### `client.OnetimeProducts.Create(ctx, params)`

Create a one-time product with multi-currency pricing.

```go
res, err := client.OnetimeProducts.Create(ctx, pancake.CreateOnetimeProductParams{
    StoreID:     "STO_xxx",
    Name:        "E-Book: TypeScript Handbook",
    Description: pancake.Ptr("Complete TypeScript guide for developers"),
    Prices: pancake.Prices{
        "USD": {Amount: "29.00", TaxCategory: pancake.TaxCategoryDigitalGoods},
        "EUR": {Amount: "27.00", TaxCategory: pancake.TaxCategoryDigitalGoods},
        "JPY": {Amount: "4500", TaxCategory: pancake.TaxCategoryDigitalGoods},
    },
    Media: []pancake.MediaItem{
        {Type: pancake.MediaTypeImage, URL: "https://example.com/cover.jpg", Alt: pancake.Ptr("Book cover")},
    },
    SuccessURL: pancake.Ptr("https://example.com/thank-you"),
    Metadata:   map[string]any{"sku": "ebook-ts-001"},
})
```

**Parameters `CreateOnetimeProductParams`**:

| Field         | Type                | Required | Description                                           |
| ------------- | ------------------- | -------- | ----------------------------------------------------- |
| `StoreID`     | `string`            | Yes      | Store ID                                              |
| `Name`        | `string`            | Yes      | Product name                                          |
| `Prices`      | `pancake.Prices`    | Yes      | Multi-currency prices (`map[string]PriceInfo`)        |
| `Description` | `*string`           | No       | Product description                                   |
| `Media`       | `[]pancake.MediaItem` | No     | Media assets (images, videos)                         |
| `SuccessURL`  | `*string`           | No       | Redirect URL after successful payment                 |
| `Metadata`    | `map[string]any`    | No       | Custom metadata                                       |

**Returns `*OnetimeProductResult`**: `{ Product, Warnings }`

### `client.OnetimeProducts.Update(ctx, params)`

Update a one-time product. Creates a new immutable version; skips if content is unchanged.

> Only `ID` is required. Omitted fields keep their current values.

```go
// Update only the name
res, err := client.OnetimeProducts.Update(ctx, pancake.UpdateOnetimeProductParams{
    ID:   "PROD_xxx",
    Name: pancake.Ptr("E-Book: TypeScript Handbook v2"),
})

// Update only the prices
res2, err := client.OnetimeProducts.Update(ctx, pancake.UpdateOnetimeProductParams{
    ID: "PROD_xxx",
    Prices: pancake.Prices{
        "USD": {Amount: "39.00", TaxCategory: pancake.TaxCategoryDigitalGoods},
    },
})
```

**Parameters `UpdateOnetimeProductParams`**:

| Field         | Type                  | Required | Description                           |
| ------------- | --------------------- | -------- | ------------------------------------- |
| `ID`          | `string`              | Yes      | Product ID                            |
| `Name`        | `*string`             | No       | Product name                          |
| `Prices`      | `pancake.Prices`      | No       | Multi-currency prices                 |
| `Description` | `*string`             | No       | Product description                   |
| `Media`       | `[]pancake.MediaItem` | No       | Media assets                          |
| `SuccessURL`  | `*string`             | No       | Redirect URL after successful payment |
| `Metadata`    | `map[string]any`      | No       | Custom metadata                       |

**Returns `*OnetimeProductResult`**: `{ Product, Warnings }`

### `client.OnetimeProducts.Publish(ctx, params)`

Publish the test version to production.

```go
res, err := client.OnetimeProducts.Publish(ctx, pancake.PublishOnetimeProductParams{ID: "PROD_xxx"})
```

**Parameters `PublishOnetimeProductParams`**:

| Field | Type     | Required | Description |
| ----- | -------- | -------- | ----------- |
| `ID`  | `string` | Yes      | Product ID  |

**Returns `*OnetimeProductResult`**: `{ Product, Warnings }`

### `client.OnetimeProducts.UpdateStatus(ctx, params)`

Activate or deactivate a product.

```go
res, err := client.OnetimeProducts.UpdateStatus(ctx, pancake.UpdateOnetimeStatusParams{
    ID:     "PROD_xxx",
    Status: pancake.ProductVersionStatusInactive,
})
```

**Parameters `UpdateOnetimeStatusParams`**:

| Field    | Type                           | Required | Description                |
| -------- | ------------------------------ | -------- | -------------------------- |
| `ID`     | `string`                       | Yes      | Product ID                 |
| `Status` | `pancake.ProductVersionStatus` | Yes      | `Active` or `Inactive`     |

**Returns `*OnetimeProductResult`**: `{ Product, Warnings }`

---

## Subscription Products

### `client.SubscriptionProducts.Create(ctx, params)`

Create a subscription product with a billing period and multi-currency pricing.

```go
res, err := client.SubscriptionProducts.Create(ctx, pancake.CreateSubscriptionProductParams{
    StoreID:       "STO_xxx",
    Name:          "Pro Plan",
    BillingPeriod: pancake.BillingPeriodMonthly,
    Prices: pancake.Prices{
        "USD": {Amount: "9.99", TaxCategory: pancake.TaxCategorySaaS},
    },
    Description: pancake.Ptr("Unlimited access to all features"),
})
```

**Parameters `CreateSubscriptionProductParams`**:

| Field           | Type                    | Required | Description                                                    |
| --------------- | ----------------------- | -------- | -------------------------------------------------------------- |
| `StoreID`       | `string`                | Yes      | Store ID                                                       |
| `Name`          | `string`                | Yes      | Product name                                                   |
| `BillingPeriod` | `pancake.BillingPeriod` | Yes      | Billing period (`Weekly` / `Monthly` / `Quarterly` / `Yearly`) |
| `Prices`        | `pancake.Prices`        | Yes      | Multi-currency prices                                          |
| `Description`   | `*string`               | No       | Product description                                            |
| `Media`         | `[]pancake.MediaItem`   | No       | Media assets                                                   |
| `SuccessURL`    | `*string`               | No       | Redirect URL after successful payment                          |
| `Metadata`      | `map[string]any`        | No       | Custom metadata                                                |

**Returns `*SubscriptionProductResult`**: `{ Product, Warnings }`

### `client.SubscriptionProducts.Update(ctx, params)`

Update a subscription product. Creates a new immutable version; skips if unchanged.

> Only `ID` is required. Omitted fields keep their current values.

```go
// Update only the name
res, err := client.SubscriptionProducts.Update(ctx, pancake.UpdateSubscriptionProductParams{
    ID:   "PROD_xxx",
    Name: pancake.Ptr("Pro Plan v2"),
})

// Update billing period and prices
res2, err := client.SubscriptionProducts.Update(ctx, pancake.UpdateSubscriptionProductParams{
    ID:            "PROD_xxx",
    BillingPeriod: pancake.Ptr(pancake.BillingPeriodYearly),
    Prices: pancake.Prices{
        "USD": {Amount: "99.00", TaxCategory: pancake.TaxCategorySaaS},
    },
})
```

**Parameters `UpdateSubscriptionProductParams`**:

| Field           | Type                     | Required | Description                           |
| --------------- | ------------------------ | -------- | ------------------------------------- |
| `ID`            | `string`                 | Yes      | Product ID                            |
| `Name`          | `*string`                | No       | Product name                          |
| `BillingPeriod` | `*pancake.BillingPeriod` | No       | Billing period                        |
| `Prices`        | `pancake.Prices`         | No       | Multi-currency prices                 |
| `Description`  | `*string`                 | No       | Product description                   |
| `Media`         | `[]pancake.MediaItem`    | No       | Media assets                          |
| `SuccessURL`    | `*string`                | No       | Redirect URL after successful payment |
| `Metadata`      | `map[string]any`         | No       | Custom metadata                       |

**Returns `*SubscriptionProductResult`**: `{ Product, Warnings }`

### `client.SubscriptionProducts.Publish(ctx, params)`

Publish the test version to production.

```go
res, err := client.SubscriptionProducts.Publish(ctx, pancake.PublishSubscriptionProductParams{ID: "PROD_xxx"})
```

**Returns `*SubscriptionProductResult`**: `{ Product, Warnings }`

### `client.SubscriptionProducts.UpdateStatus(ctx, params)`

Activate or deactivate a subscription product.

```go
res, err := client.SubscriptionProducts.UpdateStatus(ctx, pancake.UpdateSubscriptionStatusParams{
    ID:     "PROD_xxx",
    Status: pancake.ProductVersionStatusActive,
})
```

**Returns `*SubscriptionProductResult`**: `{ Product, Warnings }`

---

## Subscription Product Groups

Groups enable **shared trial periods** and **plan switching** across related subscription products (e.g. Free / Pro / Enterprise tiers).

> **Note**: Group IDs are UUIDs (not Short IDs). The `ID` field in responses and the `ID` parameter in requests use raw UUID format.

### `client.SubscriptionProductGroups.Create(ctx, params)`

```go
res, err := client.SubscriptionProductGroups.Create(ctx, pancake.CreateSubscriptionProductGroupParams{
    StoreID:     "STO_xxx",
    Name:        "Pro Plans",
    Description: pancake.Ptr("All Pro tier plans"),
    Rules:       &pancake.GroupRulesInput{SharedTrial: pancake.Ptr(true), SelfServicePlanChange: pancake.Ptr(true)},
    ProductIDs:  []string{"PROD_aaa", "PROD_bbb"},
})
```

**Parameters `CreateSubscriptionProductGroupParams`**:

| Field         | Type                  | Required | Description                                       |
| ------------- | --------------------- | -------- | ------------------------------------------------- |
| `StoreID`     | `string`              | Yes      | Store ID                                          |
| `Name`        | `string`              | Yes      | Group name                                        |
| `Description` | `*string`             | No       | Group description                                 |
| `Rules`       | `*pancake.GroupRulesInput` | No  | Group switches, each an optional `*bool`: `SharedTrial` shares the trial across the group, `SelfServicePlanChange` lets customers change plan within the group from the customer portal. A switch left nil is created off |
| `ProductIDs`  | `[]string`            | No       | Subscription product IDs to include               |

**Returns `*SubscriptionProductGroupResult`**: `{ Group, Warnings }`

### `client.SubscriptionProductGroups.Update(ctx, params)`

Update a group. `ProductIDs` is a **full replacement** (not a merge); `Rules` is the opposite — it is merged switch by switch, so a switch left nil keeps its stored value.

```go
res, err := client.SubscriptionProductGroups.Update(ctx, pancake.UpdateSubscriptionProductGroupParams{
    ID:         "spg_xxx",
    ProductIDs: []string{"PROD_aaa", "PROD_bbb", "PROD_ccc"},
})

// Open self-service plan change without touching SharedTrial
opened, err := client.SubscriptionProductGroups.Update(ctx, pancake.UpdateSubscriptionProductGroupParams{
    ID:    "spg_xxx",
    Rules: &pancake.GroupRulesInput{SelfServicePlanChange: pancake.Ptr(true)},
})
// opened.Group.Rules => { SharedTrial: <stored value>, SelfServicePlanChange: true }
```

**Returns `*SubscriptionProductGroupResult`**: `{ Group, Warnings }` — `Group.Rules` always carries both switches, so reading one never needs a fallback.

### `client.SubscriptionProductGroups.Delete(ctx, params)`

Hard-delete a group.

```go
res, err := client.SubscriptionProductGroups.Delete(ctx, pancake.DeleteSubscriptionProductGroupParams{ID: "spg_xxx"})
```

**Returns `*SubscriptionProductGroupResult`**: `{ Group, Warnings }`

### `client.SubscriptionProductGroups.Publish(ctx, params)`

Publish a test-environment group to production (upsert).

```go
res, err := client.SubscriptionProductGroups.Publish(ctx, pancake.PublishSubscriptionProductGroupParams{ID: "spg_xxx"})
```

**Returns `*SubscriptionProductGroupResult`**: `{ Group, Warnings }`

---

## Coupons

A coupon defines a discount; the `Code` you set becomes its default promotion code, the code buyers type. Codes are 3–32 letters, digits or `-`, case-insensitive (stored uppercase), and unique among the store's **active** codes in the same environment; a deactivated code is released and can be reused. Deactivation is permanent, and only the name of a coupon can change. List and read coupons with `GraphQL.Query` (`coupons`, `couponsCount`, `coupon`).

### `client.Coupons.Create(ctx, params)`

```go
res, err := client.Coupons.Create(ctx, pancake.CreateCouponParams{
	StoreID:           "STO_xxx",
	Name:              "Black Friday 20% off",
	Code:              "BF2026",
	Discount:          pancake.CouponDiscount{Type: pancake.CouponDiscountTypePercentage, PercentOff: pancake.Ptr(20.0)},
	Duration:          pancake.Ptr(pancake.CouponDurationRepeating),
	DurationInPeriods: pancake.Ptr(3),
}, pancake.WithIdempotencyKey("coupon-bf2026"))
// res.Coupon.Status is CouponStatusActive, or CouponStatusPending while the payment provider is confirming.
```

Fixed-amount coupons set `Discount: pancake.CouponDiscount{Type: pancake.CouponDiscountTypeFixedAmount, AmountOff: map[string]string{"USD": "5.00"}}` (each amount greater than 0). `Name` is at most 64 characters, counted in UTF-16 code units after trimming. Other fields: `ProductIDs`, `MaxRedemptions`, `DailyRedemptionLimit` (per buyer per UTC day), `RedeemBy` (ISO 8601; also ends discounts already attached to subscriptions), `Restrictions` (`FirstTimeOnly`, `CustomerEmails`, `PerCustomerLimit`).

**Returns `*CouponResult`**: `{ Coupon, Warnings }`. Errors: 409 when the code is taken (also when retrying **without** the idempotency key), 502 when the payment provider rejects the coupon.

### `client.Coupons.Update(ctx, params)`

Rename a coupon — the only field that can change.

```go
res, err := client.Coupons.Update(ctx, pancake.UpdateCouponParams{ID: "CPN_xxx", Name: "Black Friday (extended)"})
```

**Returns `*CouponResult`**: `{ Coupon, Warnings }`

### `client.Coupons.Deactivate(ctx, params)`

Deactivate permanently. All its promotion codes are deactivated and their codes released. Discounts already attached to subscriptions continue until the subscription's next plan change.

```go
res, err := client.Coupons.Deactivate(ctx, pancake.DeactivateCouponParams{ID: "CPN_xxx"})
// res.Coupon.Status == pancake.CouponStatusInactive
```

**Returns `*CouponResult`**: `{ Coupon, Warnings }`

---

## Promotion Codes

### `client.PromotionCodes.Create(ctx, params)`

```go
res, err := client.PromotionCodes.Create(ctx, pancake.CreatePromotionCodeParams{
	CouponID: "CPN_xxx",
	Code:     pancake.Ptr("KOL-ALICE"), // nil generates a random code
	Restrictions: &pancake.PromotionCodeRestrictions{
		MaxRedemptions: pancake.Ptr(100),
		MinimumAmount:  map[string]string{"USD": "50.00"},
	},
})
```

`PromotionCodeRestrictions` adds `MaxRedemptions` (≤ the coupon's), `ExpiresAt` (≤ the coupon's `RedeemBy`; defaults to it) and `MinimumAmount` (pre-discount, pre-tax amount per currency) to the buyer restrictions.

**Returns `*PromotionCodeResult`**: `{ PromotionCode, Warnings }`

### `client.PromotionCodes.Generate(ctx, params)`

Generate 1–500 random codes in one call. **Always retry with the same idempotency key** — a retry without it generates another batch.

```go
res, err := client.PromotionCodes.Generate(ctx, pancake.GeneratePromotionCodesParams{
	CouponID: "CPN_xxx", Count: 100, Prefix: pancake.Ptr("KOL-"),
}, pancake.WithIdempotencyKey("kol-batch-2026-11"))
// res.BatchID, res.PromotionCodes
```

**Returns `*GeneratePromotionCodesResult`**: `{ BatchID, PromotionCodes, Warnings }`

### `client.PromotionCodes.Deactivate(ctx, params)`

Deactivate permanently and release the code. The coupon's own code can only be deactivated together with the coupon (409).

```go
res, err := client.PromotionCodes.Deactivate(ctx, pancake.DeactivatePromotionCodeParams{ID: "PMC_xxx"})
```

**Returns `*PromotionCodeResult`**: `{ PromotionCode, Warnings }`

---

## Orders

### `client.Orders.CancelSubscription(ctx, params)`

Cancel a subscription order. The resulting status depends on the current order state:

| Current Status        | Result      | Behavior                                                 |
| --------------------- | ----------- | -------------------------------------------------------- |
| `pending`             | `canceled`  | Immediate cancellation                                   |
| `active` / `trialing` | `canceling` | PSP cancellation initiated; webhook updates status later |

```go
res, err := client.Orders.CancelSubscription(ctx, pancake.CancelSubscriptionParams{
    OrderID: "ORD_xxx",
})
// res.Status: "canceled" or "canceling"
```

**Parameters `CancelSubscriptionParams`**:

| Field     | Type     | Required | Description |
| --------- | -------- | -------- | ----------- |
| `OrderID` | `string` | Yes      | Order ID    |

**Returns `*CancelSubscriptionResult`**:

| Field      | Type                              | Description                                            |
| ---------- | --------------------------------- | ------------------------------------------------------ |
| `OrderID`  | `string`                          | Order ID                                               |
| `Status`   | `pancake.SubscriptionOrderStatus` | Resulting status (`"canceled"` or `"canceling"`)       |
| `Warnings` | `[]pancake.Notice`                | Migration notices                                      |

---

## Customer Self-Service

Issue a session token and create a customer session to let customers manage their own orders.

### `client.Customer(token)`

Create a customer session from a session token issued by `client.Auth.IssueSessionToken`,
running in `Config.Environment`.

```go
tok, err := client.Auth.IssueSessionToken(ctx, pancake.IssueSessionTokenParams{
    StoreID:       pancake.Ptr("STO_xxx"),
    BuyerIdentity: "customer@example.com",
})
customer := client.Customer(tok.Token)
```

No I/O is performed; the call only wires up Bearer-token HTTP plumbing.

`Config.Environment` is sent as `X-Environment` on every session request — the
session token carries none of its own, and the gateway rejects a Bearer credential
without the header. There is no default: guessing would route the call to the
other environment. When it is unset, the first session method returns `*Error`
(400, `ErrorLayerSDK`) before sending anything.

Session tokens expire 5 minutes after issuance — issue one right before use
rather than caching it.

### `client.CustomerWithEnvironment(token, environment)`

Same as `client.Customer`, with the environment given per session instead of
taken from `Config.Environment`.

```go
customer := client.CustomerWithEnvironment(tok.Token, pancake.EnvironmentTest)
```

### `customer.CancelSubscription(ctx, params)`

| Field     | Type     | Required | Description           |
| --------- | -------- | -------- | --------------------- |
| `OrderID` | `string` | Yes      | Subscription order ID |

**Returns `*CancelSubscriptionResult`**: `{ OrderID, Status, Warnings }` — `Status` is `"canceling"` (active) or `"canceled"` (pending)

### `customer.CancelOnetimeOrder(ctx, params)`

| Field     | Type     | Required | Description       |
| --------- | -------- | -------- | ----------------- |
| `OrderID` | `string` | Yes      | One-time order ID |

**Returns `*CancelOnetimeOrderResult`**: `{ OrderID, Status, Warnings }` — `Status` is `"canceled"`

### `customer.ReactivateSubscription(ctx, params)`

| Field     | Type     | Required | Description                                           |
| --------- | -------- | -------- | ----------------------------------------------------- |
| `OrderID` | `string` | Yes      | Subscription order ID (must be in `canceling` status) |

**Returns `*ReactivateSubscriptionResult`**: `{ OrderID, Status, Warnings }` — `Status` is `"active"`

### `customer.CreateRefundTicket(ctx, params)`

```go
refund, err := customer.CreateRefundTicket(ctx, pancake.CreateRefundTicketParams{
    PaymentID: "PAY_xxx",
    Reason:    "Product not as described",
    RequestedAmount: pancake.RequestedAmount{
        Amount:   "29.00",
        Currency: "USD",
    },
    RefundTicketMerchantExternalID: pancake.Ptr("REF-2026-00012"),
})
```

| Field                            | Type                      | Required | Description                                                                                                                                                                                            |
| -------------------------------- | ------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `PaymentID`                      | `string`                  | Yes      | Payment ID to refund                                                                                                                                                                                   |
| `Reason`                         | `string`                  | Yes      | Reason for the refund request                                                                                                                                                                          |
| `RequestedAmount`                | `pancake.RequestedAmount` | Yes      | Refund amount (`{Amount, Currency}`)                                                                                                                                                                   |
| `Metadata`                       | `map[string]any`          | No       | Custom metadata                                                                                                                                                                                        |
| `RefundTicketMerchantExternalID` | `*string`                 | No       | Your business-side refund-ticket identifier (max 128 chars). Surfaces under the same name in webhook payload (`data.refundTicketMerchantExternalId`) and GraphQL `RefundTicket` / `Refund` types.       |

**`RequestedAmount`**:

| Field      | Type     | Description                                |
| ---------- | -------- | ------------------------------------------ |
| `Amount`   | `string` | Amount in display format (e.g., `"29.00"`) |
| `Currency` | `string` | Currency code (ISO 4217)                   |

**Returns `*RefundTicketResult`**: `{ Ticket, Warnings }`

### `customer.ResubmitRefundTicket(ctx, params)`

| Field             | Type                      | Required | Description           |
| ----------------- | ------------------------- | -------- | --------------------- |
| `TicketID`        | `string`                  | Yes      | Existing ticket ID    |
| `PaymentID`       | `string`                  | Yes      | Payment ID            |
| `Reason`          | `string`                  | Yes      | Updated reason        |
| `RequestedAmount` | `pancake.RequestedAmount` | Yes      | Updated refund amount |

**Returns `*RefundTicketResult`**: `{ Ticket, Warnings }`

### `customer.CreatePlanChangeSession(ctx, params)`

The self-service half of a plan change: the customer switches one of their own subscriptions to another plan in the same group. Same `create-session` endpoint as the merchant methods, reached with the customer session token.

```go
session, err := customer.CreatePlanChangeSession(ctx, pancake.CustomerPlanChangeParams{
    OriginOrderID: "ORD_xxx",
    ProductID:     "PROD_target_plan",
    Currency:      "USD",
    ChangeTiming:  pancake.Ptr(pancake.ChangeTimingNextPeriod),
})
```

**Parameters `CustomerPlanChangeParams`**:

| Field | Type | Required | Description |
| --------------- | ---------------------- | -------- | ----------- |
| `OriginOrderID` | `string`               | Yes      | The customer's own subscription being changed (`ORD_xxx`) |
| `ProductID`     | `string`               | Yes      | Target plan — must be in the same product group as the current plan |
| `Currency`      | `string`               | Yes      | Currency code (ISO 4217); must match the origin subscription |
| `ChangeTiming`  | `*pancake.ChangeTiming`| No       | `ChangeTimingImmediate` or `ChangeTimingNextPeriod`; nil lets the platform derive it |
| `SuccessURL`    | `*string`              | No       | Redirect URL after the change is confirmed and paid |
| `DarkMode`      | `*bool`                | No       | Dark mode override |
| `Language`      | `*pancake.CashierLanguage` | No   | Confirmation page language (IETF BCP 47) |

The merchant-only fields (`ChangeAmount`, `ChangeCreditAmount`, `WithTrial`, `PriceSnapshot`, `ExpiresInSeconds`, `Metadata`, `OrderMerchantExternalID`, `IncludePaymentMethods`, `ExcludePaymentMethods`) are **not** on this struct. A customer-session request carries no merchant id, so the platform drops all of them without reporting it — accepting them here would only look like they worked.

Three platform checks apply to this path and not to merchant-issued links, each a 403:

| Check | Message when it fails |
|-------|----------------------|
| The subscription belongs to this customer | `Subscription order does not belong to this credential` |
| Target plan is in the same product group | `Target plan is not in the same product group as the current plan` |
| That group's `selfServicePlanChange` is on | `Self-service plan change is not enabled for this product group` |

**Returns `*CheckoutSessionResult`**: `{ SessionID, CheckoutURL, ExpiresAt, Warnings }`

### `customer.GraphQL.Query(ctx, params)`

Same parameters as `client.GraphQL.Query` but scoped to the customer's own data via session token.

| Field       | Type             | Required | Description          |
| ----------- | ---------------- | -------- | -------------------- |
| `Query`     | `string`         | Yes      | GraphQL query string |
| `Variables` | `map[string]any` | No       | Query variables      |

**Returns `*GraphQLResponse`**: `{ Data, Errors, Warnings }`

For typed access, use the package-level `pancake.CustomerGraphQLQuery[T]`:

```go
type OrdersQuery struct {
    Orders []struct {
        ID     string `json:"id"`
        Status string `json:"status"`
    } `json:"orders"`
}
resp, err := pancake.CustomerGraphQLQuery[OrdersQuery](ctx, customer, pancake.GraphQLParams{
    Query: `query { orders { id status } }`,
})
```

---

## Checkout

Waffo supports two checkout modes based on whether the merchant knows the customer's identity at checkout time:

- **Authenticated** — the merchant has a user system or collects customer info before checkout. The customer's identity is provided upfront, the checkout form is pre-filled, and a session token is automatically issued.
- **Anonymous** — the customer arrives via a template store or shared link with no prior context. They fill in billing details manually on the checkout page.

> **Authenticated checkout is recommended.** The key advantage: the order is bound to the `BuyerIdentity` you provide — a **merchant-controlled stable identifier**. Even if the customer changes the email on the checkout form, the order stays tied to your identifier. In anonymous mode, the customer self-reports their email, and a different address means a different user — **previous orders become unlinked** and **subscription trial periods can be exploited** (new email = new user = fresh trial). Additionally, anonymous checkout only supports creating orders — customers cannot cancel orders, manage subscriptions, or submit refund tickets afterward.

Changing the plan of an existing subscription is a third flow with its own pair of methods — `Checkout.CreatePlanChangeSession` and `Checkout.Authenticated.CreatePlanChange`.

For advanced use cases, the low-level `Checkout.CreateSession` is also available.

### `client.Checkout.Authenticated.Create(ctx, params)`

Authenticated checkout — the merchant provides customer identity. The SDK issues a session token, creates a checkout session, and returns a checkout URL with the token appended as a URL fragment (`#token=...`). The checkout page pre-fills customer information from the token.

Internally calls `POST /v1/actions/auth/issue-session-token` and `POST /v1/actions/checkout/create-session` in parallel.

`BuyerIdentity` is for order attribution and trial tracking only — it is not rendered on the checkout page. To pre-fill the email field on the checkout form, set `BuyerEmail` explicitly.

```go
// One-time product with customer identity (checkout page email field stays empty)
res, err := client.Checkout.Authenticated.Create(ctx, pancake.AuthenticatedCheckoutParams{
    CreateCheckoutSessionParams: pancake.CreateCheckoutSessionParams{
        ProductID:  "PROD_xxx",
        Currency:   "USD",
        SuccessURL: pancake.Ptr("https://example.com/thank-you"),
    },
    BuyerIdentity: "userIdInYourSystem",
})
// => redirect customer to res.CheckoutURL (includes #token=...)

// Subscription with trial and billing detail
subRes, err := client.Checkout.Authenticated.Create(ctx, pancake.AuthenticatedCheckoutParams{
    CreateCheckoutSessionParams: pancake.CreateCheckoutSessionParams{
        ProductID:  "PROD_yyy",
        Currency:   "USD",
        BuyerEmail: pancake.Ptr("customer@example.com"),
        WithTrial:  pancake.Ptr(true),
        BillingDetail: &pancake.BillingDetail{
            Country:    "US",
            IsBusiness: false,
            State:      pancake.Ptr("CA"),
            Postcode:   pancake.Ptr("94105"),
        },
    },
    BuyerIdentity: "userIdInYourSystem",
})
```

**Parameters `AuthenticatedCheckoutParams`** (embeds `CreateCheckoutSessionParams` plus `BuyerIdentity`):

| Field                     | Type                      | Required | Description                                                                                                                                                                |
| ------------------------- | ------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ProductID`               | `string`                  | Yes      | Product ID (product type is auto-detected server-side)                                                                                                                     |
| `Currency`                | `string`                  | Yes      | Currency code (ISO 4217)                                                                                                                                                   |
| `BuyerIdentity`           | `string`                  | Yes      | Customer identity (email or merchant-defined identifier); not serialized to the create-session body                                                                            |
| `BuyerEmail`              | `*string`                 | No       | Pre-fill checkout page email field (independent from `BuyerIdentity`)                                                                                                      |
| `BillingDetail`           | `*pancake.BillingDetail`  | No       | Pre-filled billing details (country, tax ID, etc.)                                                                                                                         |
| `PriceSnapshot`           | `*pancake.PriceInfo`      | No       | Price snapshot override (reads from DB if omitted)                                                                                                                         |
| `WithTrial`               | `*bool`                   | No       | Enable trial period (subscription only)                                                                                                                                    |
| `SuccessURL`              | `*string`                 | No       | Redirect URL after successful payment                                                                                                                                      |
| `ExpiresInSeconds`        | `*int`                    | No       | Session expiry in seconds (default: 24 hours, recommended: 60 to 604800 / 1 min to 7 days)                                                                                 |
| `DarkMode`                | `*bool`                   | No       | Dark mode override (true=dark, false=light, nil=store default)                                                                                                             |
| `Metadata`                | `map[string]string`       | No       | Custom metadata                                                                                                                                                            |
| `OrderMerchantExternalID` | `*string`                 | No       | Your business-side order identifier (max 128 chars). Surfaces under the same name on `Order` / `Payment` / `Refund` GraphQL types and in webhook payload (`data.orderMerchantExternalId`). |
| `Language`                | `*pancake.CashierLanguage` | No      | Default language of the hosted checkout page (IETF BCP 47). The customer can switch it on the page; omit to let the provider infer. |
| `IncludePaymentMethods`   | `[]pancake.PaymentMethod` | No       | Whitelist — offer only these (`card` / `applepay` / `googlepay` / `wechat`). Every value must be supported by the product type × currency pair. Mutually exclusive with `ExcludePaymentMethods`. |
| `ExcludePaymentMethods`   | `[]pancake.PaymentMethod` | No       | Blacklist — offer everything the currency supports except these. Values the currency does not offer are ignored. Mutually exclusive with `IncludePaymentMethods`. |
| `CouponID`                | `*string`                 | No       | Coupon to apply (`CPN_xxx`, from `Coupons.Create`). Locked: the customer sees the discount but cannot change or remove it, and no code entry field is shown. One-time products, new subscriptions (the coupon's `Duration` sets how many billing periods are discounted) and plan changes (see `CreatePlanChangeSession`). Mutually exclusive with `PromotionCode` and `AllowPromotionCodes` |
| `PromotionCode`           | `*string`                 | No       | Promotion code to prefill (the code string, case-insensitive). Merchant credentials only. Changeable — the customer can replace or remove it — only with `AllowPromotionCodes` true; otherwise locked. Mutually exclusive with `CouponID` |
| `AllowPromotionCodes`     | `*bool`                   | No       | Show a code entry field where the customer can enter a coupon or promotion code. The platform defaults it to `false` for API Key sessions. With `PromotionCode`, true lets the customer change or remove the prefilled code. Mutually exclusive with `CouponID` |

**Returns `*AuthenticatedCheckoutResult`**:

| Field            | Type               | Description                             |
| ---------------- | ------------------ | --------------------------------------- |
| `SessionID`      | `string`           | Session ID                              |
| `CheckoutURL`    | `string`           | Checkout URL with `#token=...` appended |
| `ExpiresAt`      | `string`           | Session expiration time                 |
| `Token`          | `string`           | Issued JWT token                        |
| `TokenExpiresAt` | `string`           | Token expiration time                   |
| `Warnings`       | `[]pancake.Notice` | Migration notices                       |

### `client.Checkout.Anonymous.Create(ctx, params)`

Anonymous checkout — visitor enters without a session token. The customer fills in billing details manually on the checkout page.

Internally calls `POST /v1/actions/checkout/create-session`.

```go
res, err := client.Checkout.Anonymous.Create(ctx, pancake.AnonymousCheckoutParams{
    ProductID: "PROD_xxx",
    Currency:  "USD",
})
// => redirect customer to res.CheckoutURL (customer fills form manually)

// With price snapshot override
snapshotRes, err := client.Checkout.Anonymous.Create(ctx, pancake.AnonymousCheckoutParams{
    ProductID: "PROD_xxx",
    Currency:  "USD",
    PriceSnapshot: &pancake.PriceInfo{
        Amount:      "19.99",
        TaxCategory: pancake.TaxCategoryDigitalGoods,
    },
})
```

`AnonymousCheckoutParams` is a type alias of `CreateCheckoutSessionParams` (same fields).

**Parameters `AnonymousCheckoutParams`**:

| Field                     | Type                      | Required | Description                                                                                                                                                                                                |
| ------------------------- | ------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ProductID`               | `string`                  | Yes      | Product ID (product type is auto-detected server-side)                                                                                                                                                     |
| `Currency`                | `string`                  | Yes      | Currency code (ISO 4217)                                                                                                                                                                                   |
| `PriceSnapshot`           | `*pancake.PriceInfo`      | No       | Price snapshot override (reads from DB if omitted)                                                                                                                                                         |
| `WithTrial`               | `*bool`                   | No       | Enable trial period (subscription only)                                                                                                                                                                    |
| `BuyerEmail`              | `*string`                 | No       | Pre-fill checkout page email field                                                                                                                                                                         |
| `BillingDetail`           | `*pancake.BillingDetail`  | No       | Pre-filled billing details                                                                                                                                                                                 |
| `SuccessURL`              | `*string`                 | No       | Redirect URL after successful payment                                                                                                                                                                      |
| `ExpiresInSeconds`        | `*int`                    | No       | Session expiry in seconds (default: 24 hours, recommended: 60 to 604800 / 1 min to 7 days)                                                                                                                 |
| `DarkMode`                | `*bool`                   | No       | Dark mode override                                                                                                                                                                                         |
| `Metadata`                | `map[string]string`       | No       | Custom metadata                                                                                                                                                                                            |
| `OrderMerchantExternalID` | `*string`                 | No       | Your business-side order identifier (max 128 chars). Honored on the API Key path; visitor / store-slug flows silently drop it. Same field name in webhook payload and GraphQL `Order` / `Payment` / `Refund`. |
| `CouponID`                | `*string`                 | No       | Coupon to apply (`CPN_xxx`, from `Coupons.Create`). Locked: the customer sees the discount but cannot change or remove it, and no code entry field is shown. One-time products, new subscriptions (the coupon's `Duration` sets how many billing periods are discounted) and plan changes (see `CreatePlanChangeSession`). Mutually exclusive with `PromotionCode` and `AllowPromotionCodes` |
| `PromotionCode`           | `*string`                 | No       | Promotion code to prefill (the code string, case-insensitive). Merchant credentials only. Changeable — the customer can replace or remove it — only with `AllowPromotionCodes` true; otherwise locked. Mutually exclusive with `CouponID` |
| `AllowPromotionCodes`     | `*bool`                   | No       | Show a code entry field where the customer can enter a coupon or promotion code. The platform defaults it to `false` for API Key sessions. With `PromotionCode`, true lets the customer change or remove the prefilled code. Mutually exclusive with `CouponID` |

**Returns `*CheckoutSessionResult`**:

| Field         | Type               | Description              |
| ------------- | ------------------ | ------------------------ |
| `SessionID`   | `string`           | Session ID               |
| `CheckoutURL` | `string`           | Hosted checkout page URL |
| `ExpiresAt`   | `string`           | Session expiration time  |
| `Warnings`    | `[]pancake.Notice` | Migration notices        |

### `client.Checkout.CreatePlanChangeSession(ctx, params)`

Issue a link that changes an existing subscription to another plan. Same `create-session` endpoint as a new purchase — `OriginOrderID` is what puts it into plan change mode, which is why it is a required field here instead of an optional one on the new-purchase params.

The returned `CheckoutURL` points at the change confirmation page (`…/store/{slug}/change/{sessionId}`), where the customer confirms the change.

```go
session, err := client.Checkout.CreatePlanChangeSession(ctx, pancake.CreatePlanChangeSessionParams{
    OriginOrderID:      "ORD_xxx",
    ProductID:          "PROD_target_plan",
    Currency:           "USD",
    ChangeTiming:       pancake.Ptr(pancake.ChangeTimingImmediate),
    ChangeCreditAmount: pancake.Ptr("8.00"),
})
```

**Parameters `CreatePlanChangeSessionParams`**:

| Field | Type | Required | Description |
| ------------------------- | ---------------------- | -------- | ----------- |
| `OriginOrderID`           | `string`               | Yes      | Subscription being changed (`ORD_xxx`). Its presence is what selects plan change mode |
| `ProductID`               | `string`               | Yes      | Target plan — a subscription product other than the current one |
| `Currency`                | `string`               | Yes      | Currency code (ISO 4217) |
| `ChangeTiming`            | `*pancake.ChangeTiming`| No       | `ChangeTimingImmediate` or `ChangeTimingNextPeriod`. Leave nil to let the platform derive it; the derived tier is not echoed back |
| `ChangeAmount`            | `*string`              | No       | What to charge for this change (display string, tax inclusive). Mutually exclusive with `ChangeCreditAmount`; merchant credentials only |
| `ChangeCreditAmount`      | `*string`              | No       | How much to credit against this change (display string, tax inclusive, same basis as `ChangeAmount`). Mutually exclusive with `ChangeAmount`; merchant credentials only |
| `WithTrial`               | `*bool`                | No       | Trial toggle for the target plan, three-state as on a new purchase. Merchant credentials only |
| `PriceSnapshot`           | `*pancake.PriceSnapshot` | No     | Price override for the target plan |
| `SuccessURL`              | `*string`              | No       | Redirect URL after the change is confirmed and paid |
| `ExpiresInSeconds`        | `*int`                 | No       | Session expiry in seconds (default: 24 hours, recommended: 60 to 604800 / 1 min to 7 days) |
| `DarkMode`                | `*bool`                | No       | Dark mode override |
| `Metadata`                | `map[string]string`    | No       | Custom metadata |
| `OrderMerchantExternalID` | `*string`              | No       | Business-side order identifier (max 128 chars) |
| `Language`                | `*pancake.CashierLanguage` | No   | Confirmation page language (IETF BCP 47) |
| `IncludePaymentMethods` / `ExcludePaymentMethods` | `[]pancake.PaymentMethod` | No | Whitelist / blacklist, mutually exclusive |
| `CouponID`                | `*string`              | No       | Coupon to apply to the new plan (`CPN_xxx`). Locked. Replaces the subscription's current discount. Mutually exclusive with `PromotionCode` and `AllowPromotionCodes` |
| `PromotionCode`           | `*string`              | No       | Promotion code to prefill on the confirmation page (code string, case-insensitive). Replaces the subscription's current discount. Changeable only with `AllowPromotionCodes` true. Mutually exclusive with `CouponID` |
| `AllowPromotionCodes`     | `*bool`                | No       | Show a code entry field on the confirmation page. The platform defaults it to `false` for API Key sessions. Mutually exclusive with `CouponID` |

There is no `BuyerEmail` or `BillingDetail`: in plan change mode the customer email, billing details, and tax all come from the origin subscription.

**Discounts on a plan change.** A `CouponID` or `PromotionCode` replaces the subscription's current discount: the remaining periods of the old one are dropped, and the new one counts its periods from the new subscription's first charge above zero (a first charge of $0, when the credit covers the new price, uses no period). Without one, the current discount carries over to the new plan with the periods already used, unless it has no periods left, its coupon's `RedeemBy` is within 60 seconds, or the coupon is deactivated — then the new plan is charged the full price. A promotion code's minimum amount is checked against the new plan's regular price before the discount, excluding tax. See [Discounts](https://docs.waffo.ai/guides/discounts#subscriptions).

Two amount fields, two ways to price the same change: `ChangeAmount` says what to charge for this period, `ChangeCreditAmount` says how much to credit against it. Same unit and tax basis, opposite meaning — sending both is rejected with a 400, and the SDK forwards what you pass rather than picking one.

**Returns `*CheckoutSessionResult`**: `{ SessionID, CheckoutURL, ExpiresAt, Warnings }`

### `client.Checkout.Authenticated.CreatePlanChange(ctx, params)`

The authenticated form of the above: issues a customer session token in parallel and appends it to the confirmation URL (`#token=...`), so the customer lands on the page already signed in.

```go
res, err := client.Checkout.Authenticated.CreatePlanChange(ctx, pancake.AuthenticatedPlanChangeParams{
    CreatePlanChangeSessionParams: pancake.CreatePlanChangeSessionParams{
        OriginOrderID: "ORD_xxx",
        ProductID:     "PROD_target_plan",
        Currency:      "USD",
        ChangeTiming:  pancake.Ptr(pancake.ChangeTimingNextPeriod),
    },
    BuyerIdentity: "userIdInYourSystem",
})
```

**Parameters `AuthenticatedPlanChangeParams`**: embeds `CreatePlanChangeSessionParams`, plus `BuyerIdentity` (`string`, required) — routed to `issue-session-token` only, never into the session body.

**Returns `*AuthenticatedCheckoutResult`**: `{ SessionID, CheckoutURL, ExpiresAt, Token, TokenExpiresAt, Warnings }`

> **There is no anonymous plan change.** A Store Slug session is an anonymous credential with no subscription to attribute the change to, and the platform answers 403 — so `client.Checkout.Anonymous` carries no plan change method.

### `client.Checkout.CreateSession(ctx, params)` (low-level)

Create a checkout session directly. For most use cases, prefer `Checkout.Authenticated.Create` or `Checkout.Anonymous.Create`.

```go
session, err := client.Checkout.CreateSession(ctx, pancake.CreateCheckoutSessionParams{
    ProductID:  "PROD_xxx",
    Currency:   "USD",
    BuyerEmail: pancake.Ptr("customer@example.com"),
})
```

**Parameters `CreateCheckoutSessionParams`**: same fields as `AnonymousCheckoutParams` above.

`CouponID`, `PromotionCode` and `AllowPromotionCodes` are checked by every checkout and plan change method: a `CouponID` without the `CPN_` prefix, an empty `PromotionCode`, or `CouponID` set together with `PromotionCode` or `AllowPromotionCodes` (either value), returns `*pancake.Error` (400, `ErrorLayerSDK`) before any request is sent. The platform answers 409 for a coupon or prefilled promotion code from another store or environment, one that is not `active`, expires within 60 seconds, does not cover the product, or has no fixed amount in the session currency; the message names the reason. Rules tied to the customer are checked when the order is placed.

```go
// Locked coupon
_, err = client.Checkout.CreateSession(ctx, pancake.CreateCheckoutSessionParams{
    ProductID: "PROD_xxx",
    Currency:  "USD",
    CouponID:  pancake.Ptr("CPN_xxx"),
})

// Code entry field on the checkout page
_, err = client.Checkout.CreateSession(ctx, pancake.CreateCheckoutSessionParams{
    ProductID:           "PROD_xxx",
    Currency:            "USD",
    AllowPromotionCodes: pancake.Ptr(true),
})

// Prefilled promotion code the customer can change or remove (leave AllowPromotionCodes nil to lock it)
_, err = client.Checkout.CreateSession(ctx, pancake.CreateCheckoutSessionParams{
    ProductID:           "PROD_xxx",
    Currency:            "USD",
    PromotionCode:       pancake.Ptr("SPRING-20"),
    AllowPromotionCodes: pancake.Ptr(true),
})
```

**`BillingDetail` fields**:

| Field          | Type      | Required    | Description                                                                                             |
| -------------- | --------- | ----------- | ------------------------------------------------------------------------------------------------------- |
| `Country`      | `string`  | Yes         | Country code (ISO 3166-1 alpha-2)                                                                       |
| `IsBusiness`   | `bool`    | Yes         | Whether this is a business purchase                                                                     |
| `Postcode`     | `*string` | No          | Postal / ZIP code                                                                                       |
| `State`        | `*string` | Conditional | State / province code (required when `Country` is `US` or `CA`)                                         |
| `BusinessName` | `*string` | Conditional | Business name (required when `IsBusiness` is `true`)                                                    |
| `TaxID`        | `*string` | Conditional | Tax ID / VAT number (required for EU countries when `IsBusiness` is `true`; triggers reverse charge 0%) |

**Returns `*CheckoutSessionResult`**: `{ SessionID, CheckoutURL, ExpiresAt, Warnings }`

---

## GraphQL

### `client.GraphQL.Query(ctx, params)`

Execute a GraphQL query. Only Query operations are supported — Mutations return a 403 error. `Data` comes back as `json.RawMessage`; unmarshal into a struct you control.

> **Note**: GraphQL field names may differ from SDK Go types. For example, `Prices` is `map[string]PriceInfo` in REST but `[CurrencyPrice!]!` in GraphQL. Use introspection (`__schema` / `__type` queries) to discover the exact schema. See [GraphQL Guide](./graphql-guide.md) for details.

```go
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query { stores { id name status } }`,
})
var data struct {
    Stores []pancake.Store `json:"stores"`
}
_ = json.Unmarshal(resp.Data, &data)

productResp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query:     `query ($id: ID!) { onetimeProduct(id: $id) { id name prices } }`,
    Variables: map[string]any{"id": "PROD_xxx"},
})
```

For type-safe access, use the package-level generic `pancake.GraphQLQuery[T]`:

```go
type StoresQuery struct {
    Stores []pancake.Store `json:"stores"`
}
resp, err := pancake.GraphQLQuery[StoresQuery](ctx, client, pancake.GraphQLParams{
    Query: `query { stores { id name status } }`,
})
fmt.Println(resp.Data.Stores[0].Name)
```

**Parameters `GraphQLParams`**:

| Field       | Type             | Required | Description          |
| ----------- | ---------------- | -------- | -------------------- |
| `Query`     | `string`         | Yes      | GraphQL query string |
| `Variables` | `map[string]any` | No       | Query variables      |

**Returns `*GraphQLResponse`** (raw):

| Field      | Type               | Description             |
| ---------- | ------------------ | ----------------------- |
| `Data`     | `json.RawMessage`  | Query result            |
| `Errors`   | `[]pancake.Notice` | GraphQL errors (if any) |
| `Warnings` | `[]pancake.Notice` | Migration notices       |

**Returns `*TypedGraphQLResponse[T]`** (generic):

| Field      | Type               | Description             |
| ---------- | ------------------ | ----------------------- |
| `Data`     | `T`                | Unmarshalled query data |
| `Errors`   | `[]pancake.Notice` | GraphQL errors          |
| `Warnings` | `[]pancake.Notice` | Migration notices       |

See [GraphQL Guide](graphql-guide.md) for introspection, filters, pagination, and practical examples.

---

## Idempotency

Every write method takes trailing `RequestOption` values:

```go
res, err := client.Stores.Create(ctx, pancake.CreateStoreParams{Name: "My Store"},
    pancake.WithIdempotencyKey("MER_store-create-9f2c"))
```

| Option | Effect |
| ------ | ------ |
| `pancake.WithIdempotencyKey(key)` | Sends `key` as `X-Idempotency-Key` on that call |
| *(none)* | No header is sent and nothing is deduplicated |

Platform behavior when a key is sent: the first request executes and its 2xx response is cached for **24 hours**; the same key returns that cached response; the same key while the original is in flight returns **409**; a non-2xx original leaves the key free to retry. Keys are at most 256 characters of letters, numbers, hyphens and underscores, and a malformed one is rejected by the gateway with a 400.

The SDK never derives a key — **uniqueness is the caller's to guarantee**, and reusing one key across two different calls makes the second replay the first one's response. For that reason a key passed to `Checkout.Authenticated.Create` / `.CreatePlanChange` is applied to the `create-session` call only, not to the token call. GraphQL queries accept no options.

---

## Error Handling

All SDK methods return `error`. When the API returns a non-success response (or client-side validation fails), the error is `*pancake.Error`. Use `errors.As` to extract it:

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

Client-side validation failures carry `Layer: ErrorLayerSDK` so they can be distinguished from server-returned errors.

---

## Types

All exported types:

| Export                                  | Description                                                |
| --------------------------------------- | ---------------------------------------------------------- |
| **Config**                              |                                                            |
| `Config`                                | Client configuration                                       |
| **Response Envelope**                   |                                                            |
| `Notice`                                | Error / warning object (`{Message, Layer, AIHint, ...}`)   |
| `APIError`                              | Alias of `Notice` (deprecated; kept for back-compat)       |
| `Error`                                 | Returned error type (`{Status, Errors}`); use `errors.As`  |
| **Auth**                                |                                                            |
| `IssueSessionTokenParams`               | Issue token request                                        |
| `SessionToken`                          | Token response                                             |
| **Store**                               |                                                            |
| `Store`                                 | Store entity                                               |
| `CreateStoreParams`                     | Create store request                                       |
| `UpdateStoreParams`                     | Update store request                                       |
| `DeleteStoreParams`                     | Delete store request                                       |
| `CreateStoreResult`                     | Create store response (also aliased as Update/DeleteStoreResult) |
| `NotificationSettings`                  | 21 notification toggles (10 Notify* + 11 Email*)           |
| `CheckoutSettings`                      | Checkout page theme (light/dark)                           |
| `CheckoutThemeSettings`                 | Single-theme checkout styling                              |
| **Store Webhook** (managed via `client.Webhooks`)             |                                              |
| `StoreWebhook`                          | Configured webhook endpoint                                |
| `AddWebhookParams`                      | Add webhook request                                        |
| `UpdateWebhookParams`                   | Update webhook request                                     |
| `RemoveWebhookParams`                   | Remove webhook request                                     |
| **Store Merchant**                      |                                                            |
| `AddMerchantParams`                     | Add merchant request                                       |
| `AddMerchantResult`                     | Add merchant response                                      |
| `RemoveMerchantParams`                  | Remove merchant request                                    |
| `RemoveMerchantResult`                  | Remove merchant response                                   |
| `UpdateRoleParams`                      | Update role request                                        |
| `UpdateRoleResult`                      | Update role response                                       |
| **Product (shared)**                    |                                                            |
| `PriceInfo`                             | Single-currency price (amount in display units)            |
| `Prices`                                | Multi-currency prices (`map[string]PriceInfo`)             |
| `MediaItem`                             | Media asset (image or video)                               |
| **Onetime Product**                     |                                                            |
| `OnetimeProductDetail`                  | One-time product entity                                    |
| `OnetimeProductResult`                  | Envelope `{Product, Warnings}`                             |
| `CreateOnetimeProductParams`            | Create request                                             |
| `UpdateOnetimeProductParams`            | Update request (creates new version)                       |
| `PublishOnetimeProductParams`           | Publish test → prod                                        |
| `UpdateOnetimeStatusParams`             | Activate / deactivate                                      |
| **Subscription Product**                |                                                            |
| `SubscriptionProductDetail`             | Subscription product entity                                |
| `SubscriptionProductResult`             | Envelope `{Product, Warnings}`                             |
| `CreateSubscriptionProductParams`       | Create request                                             |
| `UpdateSubscriptionProductParams`       | Update request (creates new version)                       |
| `PublishSubscriptionProductParams`      | Publish test → prod                                        |
| `UpdateSubscriptionStatusParams`        | Activate / deactivate                                      |
| **Subscription Product Group**          |                                                            |
| `SubscriptionProductGroup`              | Product group entity                                       |
| `SubscriptionProductGroupResult`        | Envelope `{Group, Warnings}`                               |
| `GroupRules`                            | Group switches as returned on an entity (both always set)  |
| `GroupRulesInput`                       | Group switches as accepted on create / update (optional, merged) |
| `CreateSubscriptionProductGroupParams`  | Create request                                             |
| `UpdateSubscriptionProductGroupParams`  | Update request (`ProductIDs` = full replacement)           |
| `DeleteSubscriptionProductGroupParams`  | Delete request                                             |
| `PublishSubscriptionProductGroupParams` | Publish test → prod                                        |
| **Coupon & Promotion Code**             |                                                            |
| `CouponDiscount`                        | Discount of a coupon (`Type` + `PercentOff` or `AmountOff`) |
| `DiscountRestrictions`                  | Buyer restrictions accepted on create                      |
| `CouponRestrictionsDetail`              | Buyer restrictions as returned on a coupon                 |
| `CouponDetail`                          | Coupon entity                                              |
| `CreateCouponParams`                    | Create request (`Code` becomes the default promotion code) |
| `UpdateCouponParams`                    | Rename request                                             |
| `DeactivateCouponParams`                | Deactivate request (permanent)                             |
| `CouponResult`                          | Envelope `{Coupon, Warnings}`                              |
| `PromotionCodeRestrictions`             | Promotion code restrictions accepted on create / generate  |
| `PromotionCodeRestrictionsDetail`       | Promotion code restrictions as returned on a promotion code |
| `PromotionCodeDetail`                   | Promotion code entity                                      |
| `CreatePromotionCodeParams`             | Create request (`Code` optional)                           |
| `GeneratePromotionCodesParams`          | Bulk generate request (1–500 codes)                        |
| `DeactivatePromotionCodeParams`         | Deactivate request (permanent)                             |
| `PromotionCodeResult`                   | Envelope `{PromotionCode, Warnings}`                       |
| `GeneratePromotionCodesResult`          | Envelope `{BatchID, PromotionCodes, Warnings}`             |
| **Order**                               |                                                            |
| `CancelSubscriptionParams`              | Cancel subscription request                                |
| `CancelSubscriptionResult`              | Cancel subscription response                               |
| `BillingDetail`                         | Customer billing details (country, tax ID, etc.)              |
| **Customer Self-Service**                  |                                                            |
| `CancelOnetimeOrderParams`              | Cancel one-time order request                              |
| `CancelOnetimeOrderResult`              | Cancel one-time order response                             |
| `ReactivateSubscriptionParams`          | Reactivate subscription request                            |
| `ReactivateSubscriptionResult`          | Reactivate subscription response                           |
| `CreateRefundTicketParams`              | Create refund ticket request                               |
| `ResubmitRefundTicketParams`            | Resubmit refund ticket request                             |
| `RefundTicket`                          | Refund ticket entity                                       |
| `RefundTicketResult`                    | Envelope `{Ticket, Warnings}`                              |
| `RefundTicketVersionData`               | Per-submission refund ticket data                          |
| `RequestedAmount`                       | Refund amount (`{Amount, Currency}`)                       |
| **Checkout**                            |                                                            |
| `AuthenticatedCheckoutParams`           | Authenticated checkout request (with customer identity)       |
| `AuthenticatedCheckoutResult`           | Authenticated checkout response (URL with token + expiry)  |
| `AnonymousCheckoutParams`               | Alias of `CreateCheckoutSessionParams`                     |
| `CreateCheckoutSessionParams`           | Low-level checkout session request                         |
| `CreatePlanChangeSessionParams`         | Plan change session request (`OriginOrderID` required)     |
| `CustomerPlanChangeParams`              | Customer-initiated plan change request (no merchant-only fields) |
| `RequestOption` / `WithIdempotencyKey`  | Per-call options; idempotency key only                     |
| `AuthenticatedPlanChangeParams`         | Plan change request with customer identity                 |
| `ChangeTiming`                          | When a plan change takes effect (`immediate` / `next_period`) |
| `CheckoutSessionResult`                 | Checkout session response (URL + expiry)                   |
| **GraphQL**                             |                                                            |
| `GraphQLParams`                         | GraphQL query parameters                                   |
| `GraphQLResponse`                       | Raw GraphQL response envelope (`Data` is `json.RawMessage`) |
| `TypedGraphQLResponse[T]`               | Typed GraphQL response envelope                            |
| `GraphQLErrorLocation`                  | Position in query string (graphql-js errors)               |
| `GraphQLError`                          | Alias of `Notice` (deprecated)                             |
| `GraphQLWarning`                        | Alias of `Notice` (deprecated)                             |
| **Webhook**                             |                                                            |
| `WebhookEvent`                          | Verified event envelope (`Data` is `json.RawMessage`)      |
| `TypedWebhookEvent[T]`                  | Typed event envelope                                       |
| `WebhookEventData`                      | Common event data fields                                   |
| `WebhookAmountBreakdown`                | Amount breakdown of the object an event refers to          |
| `WebhookPublicKeys`                     | Per-environment webhook public keys                        |
| `VerifyWebhookOptions`                  | Verification options (environment, tolerance, key)         |
| **Nullable helpers**                    |                                                            |
| `Nullable[T]`                           | Tri-state JSON field (absent / explicit null / value)      |
| `Ptr[T](v)`                             | Pointer-to-value helper                                    |
| `NullValue[T](v)` / `NullValuePtr[T](v)` | Wrap a value as a non-null Nullable                       |
| `ExplicitNull[T]()` / `ExplicitNullPtr[T]()` | Wrap as an explicit-null Nullable                     |
| **Enums** (see `enums.go`)              |                                                            |
| `Environment`                           | `EnvironmentTest` / `EnvironmentProd`                      |
| `TaxCategory`                           | `TaxCategoryDigitalGoods` / `SaaS` / `Software` / `Ebook` / `OnlineCourse` / `Consulting` / `ProfessionalService` |
| `BillingPeriod`                         | `Weekly` / `Monthly` / `Quarterly` / `Yearly`              |
| `PaymentMethod`                         | `Card` / `ApplePay` / `GooglePay` / `WeChat`               |
| `CashierLanguage`                       | `En` / `PtBR` / `EsMX` / `IDID` / `ViVN` / `RuRU` / `EnKE` / `EsPE` / `EsCO` / `EsCL` / `ZhHantTW` / `ZhHantHK` / `ThTH` / `JaJP` / `EnNG` / `KoKR` / `EnHK` / `ZhHansHK` / `PlPL` / `TrTR` / `ZhHans` / `MsMY` |
| `ProductVersionStatus`                  | `Active` / `Inactive`                                      |
| `EntityStatus`                          | `Active` / `Inactive` / `Suspended`                        |
| `StoreRole`                             | `Owner` / `Admin` / `Member`                               |
| `OnetimeOrderStatus`                    | `Pending` / `Completed` / `Canceled`                       |
| `SubscriptionOrderStatus`               | `Pending` / `Active` / `Canceling` / `PastDue` / `Closed` / `Canceled` / `Expired` |
| `PaymentStatus`                         | `Pending` / `Succeeded` / `Failed` / `Canceled`            |
| `RefundTicketStatus`                    | `Pending` / `UnderReview` / `Approved` / `Rejected` / `Returned` / `Processing` / `Succeeded` / `Failed` / `Cancelled` |
| `RefundStatus`                          | `Succeeded` / `Failed`                                     |
| `MediaType`                             | `Image` / `Video`                                          |
| `ErrorLayer`                            | `Gateway` / `User` / `Store` / `Product` / `Order` / `Ticket` / `GraphQL` / `Resource` / `Email` / `SDK` |
| `WebhookEventType`                      | `OrderCompleted` / `SubscriptionActivated` / `SubscriptionPaymentSucceeded` / `SubscriptionRenewed` / `SubscriptionRecovered` / `SubscriptionPlanChanged` / `SubscriptionPlanChangeScheduled` / `SubscriptionPlanChangeFailed` / `SubscriptionCanceling` / `SubscriptionUncanceled` / `SubscriptionCanceled` / `SubscriptionPastDue` / `RefundSucceeded` / `RefundFailed` |
| `WebhookChannel`                        | `HTTP` / `Feishu` / `Discord` / `Telegram` / `Slack`       |
| `CouponStatus`                          | `Pending` / `Active` / `Inactive` / `Failed`               |
| `CouponDiscountType`                    | `Percentage` / `FixedAmount`                               |
| `CouponDuration`                        | `Once` / `Repeating` / `Forever`                           |
| `PromotionCodeStatus`                   | `Active` / `Inactive`                                      |
