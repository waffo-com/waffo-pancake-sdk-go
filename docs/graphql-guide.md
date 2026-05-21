# GraphQL Guide

The Waffo Pancake GraphQL API is **query-only** — Mutations are not supported and return a 403 error. All queries go through `client.GraphQL.Query(ctx, ...)` (raw access) or `pancake.GraphQLQuery[T](ctx, client, ...)` (typed access).

## Introspection

Introspection is **enabled by default**. Use it during development to explore the full schema, discover available types, fields, and filter conditions.

> **Recommended**: Always use introspection to stay in sync with the server — this guide covers common queries, but the schema is the source of truth.
>
> **Important**: The SDK's Go types (e.g. `pancake.Prices`, `pancake.MediaItem`) reflect the **REST API** shape. The GraphQL schema may represent the same data differently — for example, `prices` is a `map[string]PriceInfo` in REST but `[CurrencyPrice!]!` (array of `{currency, priceInfo}`) in GraphQL. Always use introspection or the examples below for GraphQL field names, not the SDK type definitions.

### Discover All Query Fields

```go
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `{
        __schema {
            queryType {
                fields {
                    name
                    description
                    args { name type { name kind ofType { name } } }
                }
            }
        }
    }`,
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(resp.Data))
```

### Inspect a Specific Type

```go
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `{
        __type(name: "OnetimeOrder") {
            fields {
                name
                type { name kind ofType { name } }
            }
        }
    }`,
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(resp.Data))
```

### Interactive Schema Browsers

You can also connect [GraphiQL](https://github.com/graphql/graphiql) or [Apollo Sandbox](https://studio.apollographql.com/sandbox) to `https://api.waffo.ai/v1/graphql` for interactive schema browsing with auto-complete.

---

## Practical Examples

Both raw and typed access patterns are shown in the first example below. Subsequent examples use the typed form for brevity; either form works.

### 1. Store Queries

```go
// Typed form — define a struct and let GraphQLQuery unmarshal into it.
type StoresQuery struct {
    Stores []struct {
        ID           string  `json:"id"`
        Name         string  `json:"name"`
        Slug         string  `json:"slug"`
        Status       string  `json:"status"`
        SupportEmail *string `json:"supportEmail"`
        CreatedAt    string  `json:"createdAt"`
    } `json:"stores"`
}
typed, err := pancake.GraphQLQuery[StoresQuery](ctx, client, pancake.GraphQLParams{
    Query: `query { stores { id name slug status supportEmail createdAt } }`,
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(typed.Data.Stores[0].Name)

// Raw form — Data is json.RawMessage; unmarshal yourself.
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query { stores { id name slug status supportEmail createdAt } }`,
})
if err != nil {
    log.Fatal(err)
}
var raw struct {
    Stores []pancake.Store `json:"stores"`
}
_ = json.Unmarshal(resp.Data, &raw)

// Single store by ID
type StoreQuery struct {
    Store *struct {
        ID     string `json:"id"`
        Name   string `json:"name"`
        Slug   string `json:"slug"`
        Status string `json:"status"`
    } `json:"store"`
}
single, err := pancake.GraphQLQuery[StoreQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($id: ID!) {
        store(id: $id) { id name slug status }
    }`,
    Variables: map[string]any{"id": "STO_xxx"},
})
```

### 2. Product Queries

```go
// One-time products with prices
type ProductsQuery struct {
    OnetimeProducts []struct {
        ID     string `json:"id"`
        Name   string `json:"name"`
        Status string `json:"status"`
        Prices []struct {
            Currency  string `json:"currency"`
            PriceInfo struct {
                Amount      string `json:"amount"`
                TaxCategory string `json:"taxCategory"`
            } `json:"priceInfo"`
        } `json:"prices"`
        HasProdVersion bool `json:"hasProdVersion"`
    } `json:"onetimeProducts"`
}
products, err := pancake.GraphQLQuery[ProductsQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        onetimeProducts(storeId: $storeId, filter: { status: { eq: "active" } }) {
            id name status
            prices { currency priceInfo { amount taxCategory } }
            hasProdVersion
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})

// Subscription products
type SubProductsQuery struct {
    SubscriptionProducts []struct {
        ID            string `json:"id"`
        Name          string `json:"name"`
        BillingPeriod string `json:"billingPeriod"`
        Status        string `json:"status"`
        Prices        []struct {
            Currency  string `json:"currency"`
            PriceInfo struct {
                Amount      string `json:"amount"`
                TaxCategory string `json:"taxCategory"`
            } `json:"priceInfo"`
        } `json:"prices"`
    } `json:"subscriptionProducts"`
}
subProducts, err := pancake.GraphQLQuery[SubProductsQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        subscriptionProducts(storeId: $storeId) {
            id name billingPeriod status
            prices { currency priceInfo { amount taxCategory } }
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### 3. Order Queries

```go
// One-time orders with price snapshot
type OnetimeOrdersQuery struct {
    OnetimeOrders []struct {
        ID            string `json:"id"`
        BuyerEmail    string `json:"buyerEmail"`
        Currency      string `json:"currency"`
        Status        string `json:"status"`
        PriceSnapshot struct {
            Currency    string `json:"currency"`
            Subtotal    string `json:"subtotal"`
            TaxAmount   string `json:"taxAmount"`
            Total       string `json:"total"`
            TaxCategory string `json:"taxCategory"`
        } `json:"priceSnapshot"`
        OnetimeProduct struct {
            Name string `json:"name"`
        } `json:"onetimeProduct"`
        CreatedAt string `json:"createdAt"`
    } `json:"onetimeOrders"`
}
orders, err := pancake.GraphQLQuery[OnetimeOrdersQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        onetimeOrders(storeId: $storeId) {
            id buyerEmail currency status
            priceSnapshot { currency subtotal taxAmount total taxCategory }
            onetimeProduct { name }
            createdAt
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})

// Subscription orders
type SubOrdersQuery struct {
    SubscriptionOrders []struct {
        ID            string `json:"id"`
        BuyerEmail    string `json:"buyerEmail"`
        Status        string `json:"status"`
        BillingPeriod string `json:"billingPeriod"`
        PriceSnapshot struct {
            Currency     string `json:"currency"`
            RegularPhase struct {
                Subtotal    string `json:"subtotal"`
                TaxAmount   string `json:"taxAmount"`
                Total       string `json:"total"`
                TaxCategory string `json:"taxCategory"`
            } `json:"regularPhase"`
            SpecialPhase *struct {
                Subtotal    string `json:"subtotal"`
                TaxAmount   string `json:"taxAmount"`
                Total       string `json:"total"`
                TaxCategory string `json:"taxCategory"`
            } `json:"specialPhase"`
            SpecialPhaseDays *int `json:"specialPhaseDays"`
        } `json:"priceSnapshot"`
        CurrentPeriodEnd string  `json:"currentPeriodEnd"`
        CanceledAt       *string `json:"canceledAt"`
    } `json:"subscriptionOrders"`
}
subOrders, err := pancake.GraphQLQuery[SubOrdersQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        subscriptionOrders(storeId: $storeId) {
            id buyerEmail status billingPeriod
            priceSnapshot {
                currency
                regularPhase { subtotal taxAmount total taxCategory }
                specialPhase { subtotal taxAmount total taxCategory }
                specialPhaseDays
            }
            currentPeriodEnd canceledAt
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### 4. Order Details (with Payment History)

```go
// One-time order with payments and refunds
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query ($id: ID!) {
        onetimeOrder(id: $id) {
            id buyerEmail currency status testMode
            priceSnapshot { currency subtotal taxAmount total taxCategory }
            billingDetail { country isBusiness postcode state businessName taxId }
            onetimeProduct { id name }
            productVersion { id versionNumber name }
            payments {
                id status refundStatus
                snapshotAmountDetails { currency subtotal taxAmount total taxCategory phase }
                cardInfo { brand last4 expMonth expYear }
                failureReason createdAt
                refunds { id status requestedAmountDetails { currency amount } createdAt }
            }
            createdAt updatedAt
        }
    }`,
    Variables: map[string]any{"id": "ORD_xxx"},
})

// Subscription order with renewal status
resp, err = client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query ($id: ID!) {
        subscriptionOrder(id: $id) {
            id buyerEmail status billingPeriod
            priceSnapshot {
                currency specialPhaseDays
                specialPhase { subtotal taxAmount total taxCategory }
                regularPhase { subtotal taxAmount total taxCategory }
            }
            billingDetail { country isBusiness }
            currentPeriodStart currentPeriodEnd canceledAt
            subscriptionProduct { id name }
            productVersion { id versionNumber name }
            payments {
                id status refundStatus
                snapshotAmountDetails { currency subtotal taxAmount total taxCategory phase }
                createdAt
            }
            createdAt
        }
    }`,
    Variables: map[string]any{"id": "ORD_xxx"},
})
```

### 5. Payment and Refund Queries

```go
// Payments with filters
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query {
        payments(filter: { status: { eq: "succeeded" } }) {
            id
            onetimeOrder { id }
            subscriptionOrder { id }
            snapshotAmountDetails { currency subtotal taxAmount total taxCategory phase }
            cardInfo { brand last4 }
            status createdAt
        }
    }`,
})

// Refund tickets
resp, err = client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query {
        refundTickets(limit: 20, filter: { status: { eq: "pending" } }) {
            id status reason
            requestedAmountDetails { currency amount }
            payment {
                id status
                snapshotAmountDetails { currency subtotal taxAmount total taxCategory phase }
                onetimeOrder { id buyerEmail store { name } }
                subscriptionOrder { id buyerEmail store { name } }
            }
            createdAt updatedAt
        }
        refundTicketsCount(filter: { status: { eq: "pending" } })
    }`,
})

// Look up by merchant business numbers — flat dual-key naming, same field name
// appears on Order / Payment / Refund types and in webhook payload.
type PaymentsByOrderRef struct {
    Payments []struct {
        ID                      string  `json:"id"`
        OrderID                 string  `json:"orderId"`
        Status                  string  `json:"status"`
        OrderMerchantExternalID *string `json:"orderMerchantExternalId"`
    } `json:"payments"`
}
byOrderRef, err := pancake.GraphQLQuery[PaymentsByOrderRef](ctx, client, pancake.GraphQLParams{
    Query: `query ($ref: String!) {
        payments(filter: { orderMerchantExternalId: { eq: $ref } }) {
            id orderId status orderMerchantExternalId
        }
    }`,
    Variables: map[string]any{"ref": "ORDER-2026-00891"},
})

type ByRefundTicketRef struct {
    RefundTickets []struct {
        ID                             string  `json:"id"`
        Status                         string  `json:"status"`
        RefundTicketMerchantExternalID *string `json:"refundTicketMerchantExternalId"`
    } `json:"refundTickets"`
    Refunds []struct {
        ID                             string  `json:"id"`
        Status                         string  `json:"status"`
        OrderMerchantExternalID        *string `json:"orderMerchantExternalId"`
        RefundTicketMerchantExternalID *string `json:"refundTicketMerchantExternalId"`
        PSPAmountDetails               struct {
            Amount   string `json:"amount"`
            Currency string `json:"currency"`
        } `json:"pspAmountDetails"`
    } `json:"refunds"`
}
byRefundTicketRef, err := pancake.GraphQLQuery[ByRefundTicketRef](ctx, client, pancake.GraphQLParams{
    Query: `query ($ref: String!) {
        refundTickets(filter: { refundTicketMerchantExternalId: { eq: $ref } }) {
            id status refundTicketMerchantExternalId
        }
        refunds(filter: { refundTicketMerchantExternalId: { eq: $ref } }) {
            id status orderMerchantExternalId refundTicketMerchantExternalId
            pspAmountDetails { amount currency }
        }
    }`,
    Variables: map[string]any{"ref": "REF-2026-00012"},
})
```

> The `Refund` type exposes **both** keys as flat fields (`orderMerchantExternalId` from the originating order, `refundTicketMerchantExternalId` from the originating refund ticket). `Order` / `Payment` / `RefundTicket` carry only the key relevant to their entity. The field name on the wire matches the webhook payload (`data.orderMerchantExternalId` / `data.refundTicketMerchantExternalId`).

### 6. Merchant Info and Store Associations

```go
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query ($id: ID!) {
        merchant(id: $id) {
            id email name status
            storeMerchants {
                role
                store { id name slug status }
            }
            apiKeys { id nickname environment recentlyUsed createdAt }
        }
    }`,
    Variables: map[string]any{"id": "MER_xxx"},
})
```

### 7. Product Versions

```go
// One-time product version history
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query ($productId: String!) {
        onetimeProductVersions(productId: $productId) {
            id versionNumber name description
            prices { currency priceInfo { amount taxCategory } }
            media { type url alt thumbnail }
            metadata isTestVersion isProdVersion createdAt
        }
    }`,
    Variables: map[string]any{"productId": "PROD_xxx"},
})

// Subscription product versions
resp, err = client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query ($productId: String!) {
        subscriptionProductVersions(productId: $productId) {
            id versionNumber name description billingPeriod
            prices { currency priceInfo { amount taxCategory } }
            metadata isTestVersion isProdVersion createdAt
        }
    }`,
    Variables: map[string]any{"productId": "PROD_xxx"},
})
```

### 8. Subscription Product Groups

```go
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        subscriptionProductGroups(storeId: $storeId) {
            id name description
            rules { sharedTrial }
            environment
            products {
                id name billingPeriod
                prices { currency priceInfo { amount taxCategory } }
            }
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### 9. Exchange Rate Query

```go
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query {
        rate(fromCurrency: USD, toCurrency: EUR) {
            fromCurrency toCurrency standardRate rateRefId expiryTime
        }
    }`,
})
```

> `CurrencyCode` is an enum type supporting 40+ ISO 4217 currency codes (e.g. `USD`, `EUR`, `GBP`, `JPY`, `CNY`). Use introspection to get the full list.

### 10. Webhook and Email Delivery Logs

```go
// Webhook delivery logs (auto-filtered by environment)
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        webhookDeliveries(storeId: $storeId, limit: 20, filter: { status: { eq: "failed" } }) {
            id storeId eventType eventId
            payload webhookUrl status httpStatus responseBody
            attemptCount lastAttemptedAt createdAt
        }
        webhookDeliveriesCount(storeId: $storeId, filter: { status: { eq: "failed" } })
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})

// Email delivery logs
resp, err = client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        emailDeliveries(storeId: $storeId, limit: 20, filter: { status: { eq: "failed" } }) {
            id storeId eventType eventId
            recipientType toAddress subject testMode
            status attemptCount lastAttemptedAt errorMessage createdAt
        }
        emailDeliveriesCount(storeId: $storeId, filter: { status: { eq: "failed" } })
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### 11. Dashboard Overview (Combined Query)

Combine multiple queries in a single request to reduce network round trips.

```go
type DashboardQuery struct {
    Store *struct {
        Name   string `json:"name"`
        Slug   string `json:"slug"`
        Status string `json:"status"`
    } `json:"store"`
    OnetimeOrdersCount      int `json:"onetimeOrdersCount"`
    SubscriptionOrdersCount int `json:"subscriptionOrdersCount"`
    OnetimeOrders           []struct {
        ID            string `json:"id"`
        BuyerEmail    string `json:"buyerEmail"`
        PriceSnapshot struct {
            Currency string `json:"currency"`
            Total    string `json:"total"`
        } `json:"priceSnapshot"`
        CreatedAt string `json:"createdAt"`
    } `json:"onetimeOrders"`
    RefundTickets []struct {
        ID                     string `json:"id"`
        RequestedAmountDetails struct {
            Currency string `json:"currency"`
            Amount   string `json:"amount"`
        } `json:"requestedAmountDetails"`
        Reason    string `json:"reason"`
        CreatedAt string `json:"createdAt"`
    } `json:"refundTickets"`
    OnetimeProductsCount      int `json:"onetimeProductsCount"`
    SubscriptionProductsCount int `json:"subscriptionProductsCount"`
}
dashboard, err := pancake.GraphQLQuery[DashboardQuery](ctx, client, pancake.GraphQLParams{
    Query: `query Dashboard($storeId: String!) {
        store(id: $storeId) { name slug status }
        onetimeOrdersCount(storeId: $storeId)
        subscriptionOrdersCount(storeId: $storeId)
        onetimeOrders(storeId: $storeId, limit: 5, filter: { status: { eq: "pending" } }) {
            id buyerEmail priceSnapshot { currency total } createdAt
        }
        refundTickets(limit: 5, filter: { status: { eq: "pending" } }) {
            id requestedAmountDetails { currency amount } reason createdAt
        }
        onetimeProductsCount(storeId: $storeId)
        subscriptionProductsCount(storeId: $storeId)
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

---

## Count Queries

All list queries have corresponding `*Count` queries that return the total matching a filter — useful for pagination.

```go
resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        storesCount
        storeMerchantsCount
        apiKeysCount
        onetimeProductsCount(storeId: $storeId, filter: { status: { eq: "active" } })
        subscriptionProductsCount(storeId: $storeId)
        subscriptionProductGroupsCount(storeId: $storeId)
        onetimeOrdersCount(storeId: $storeId)
        subscriptionOrdersCount(storeId: $storeId)
        paymentsCount
        refundsCount
        refundTicketsCount
        webhookDeliveriesCount(storeId: $storeId)
        emailDeliveriesCount(storeId: $storeId)
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

> Count queries accept the same `filter` parameter as their corresponding list queries.

---

## Filter Types

| Filter Type      | Operations                                             | Example Fields                        |
| ---------------- | ------------------------------------------------------ | ------------------------------------- |
| `StringFilter`   | `eq`, `ne`, `contains`, `startsWith`, `endsWith`, `in` | `status`, `name`, `email`, `currency` |
| `DateTimeFilter` | `eq`, `ne`, `gt`, `gte`, `lt`, `lte`                   | `createdAt`, `updatedAt`, `expiresAt` |
| `IntFilter`      | `eq`, `ne`, `gt`, `gte`, `lt`, `lte`                   | `amount`, `totalAmount`               |
| `BooleanFilter`  | `eq`                                                   | `prodEnabled`, `testMode`             |

> To see which filter fields are available for a specific entity, use introspection:
> `__type(name: "OnetimeOrderFilter") { fields { name type { name } } }`

---

## Analytics Queries

Analytics queries provide aggregated statistics, trends, and insights. All analytics queries accept `storeId` (or `storeSlug`) and an `AnalyticsFilterInput` parameter.

### AnalyticsFilterInput

| Field                        | Type     | Required | Description                |
| ---------------------------- | -------- | -------- | -------------------------- |
| `filter.timeRange.startDate` | `String` | Yes      | Start time (ISO 8601)      |
| `filter.timeRange.endDate`   | `String` | Yes      | End time (ISO 8601)        |
| `filter.currency`            | `String` | No       | Currency filter (ISO 4217) |
| `filter.status`              | `String` | No       | Status filter              |

### TimePeriodGranularity

`DAY`, `WEEK`, `MONTH`, `QUARTER`, `YEAR`, `ALL_TIME`

### orderStatistics — Order Aggregation

```go
type OrderStatsQuery struct {
    OrderStatistics struct {
        TotalCount      int `json:"totalCount"`
        CountsByStatus  []struct {
            Status string `json:"status"`
            Count  int    `json:"count"`
        } `json:"countsByStatus"`
        CountsByPeriod []struct {
            Period string `json:"period"`
            Count  int    `json:"count"`
        } `json:"countsByPeriod"`
        RevenueByCurrency []struct {
            Currency     string `json:"currency"`
            TotalAmount  string `json:"totalAmount"`
            PaymentCount int    `json:"paymentCount"`
        } `json:"revenueByCurrency"`
        RevenueByPeriod []struct {
            Period       string `json:"period"`
            Currency     string `json:"currency"`
            TotalAmount  string `json:"totalAmount"`
            PaymentCount int    `json:"paymentCount"`
        } `json:"revenueByPeriod"`
        BuyerMetrics struct {
            TotalBuyers      int `json:"totalBuyers"`
            NewBuyers        int `json:"newBuyers"`
            ReturningBuyers  int `json:"returningBuyers"`
        } `json:"buyerMetrics"`
        RevenueByCountry []struct {
            Country      string `json:"country"`
            TotalAmount  string `json:"totalAmount"`
            PaymentCount int    `json:"paymentCount"`
        } `json:"revenueByCountry"`
        OrdersByCountry []struct {
            Country string `json:"country"`
            Count   int    `json:"count"`
        } `json:"ordersByCountry"`
        B2BVsB2CBreakdown []struct {
            IsBusiness  bool   `json:"isBusiness"`
            Label       string `json:"label"`
            TotalAmount string `json:"totalAmount"`
            OrderCount  int    `json:"orderCount"`
        } `json:"b2bVsB2cBreakdown"`
        RevenueByState []struct {
            State        string `json:"state"`
            TotalAmount  string `json:"totalAmount"`
            PaymentCount int    `json:"paymentCount"`
        } `json:"revenueByState"`
    } `json:"orderStatistics"`
}
orderStats, err := pancake.GraphQLQuery[OrderStatsQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        orderStatistics(
            storeId: $storeId,
            filter: { timeRange: { startDate: "2025-01-01T00:00:00Z", endDate: "2026-01-01T00:00:00Z" } }
        ) {
            totalCount
            countsByStatus { status count }
            countsByPeriod(granularity: MONTH) { period count }
            revenueByCurrency { currency totalAmount paymentCount }
            revenueByPeriod(granularity: MONTH, currency: "usd") { period currency totalAmount paymentCount }
            buyerMetrics { totalBuyers newBuyers returningBuyers }
            revenueByCountry(currency: "usd") { country totalAmount paymentCount }
            ordersByCountry { country count }
            b2bVsB2cBreakdown(currency: "usd") { isBusiness label totalAmount orderCount }
            revenueByState(country: "US", currency: "usd") { state totalAmount paymentCount }
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### paymentStatistics — Payment Success Rates & Refunds

```go
type PaymentStatsQuery struct {
    PaymentStatistics struct {
        SuccessRate struct {
            TotalAttempts int     `json:"totalAttempts"`
            Succeeded     int     `json:"succeeded"`
            Failed        int     `json:"failed"`
            Pending       int     `json:"pending"`
            SuccessRate   float64 `json:"successRate"`
        } `json:"successRate"`
        FailedReasons []struct {
            Reason     string  `json:"reason"`
            Count      int     `json:"count"`
            Percentage float64 `json:"percentage"`
        } `json:"failedReasons"`
        Refunds struct {
            TotalCount       int `json:"totalCount"`
            SucceededCount   int `json:"succeededCount"`
            PendingCount     int `json:"pendingCount"`
            FailedCount      int `json:"failedCount"`
            AmountByCurrency []struct {
                Currency     string `json:"currency"`
                TotalAmount  string `json:"totalAmount"`
                PaymentCount int    `json:"paymentCount"`
            } `json:"amountByCurrency"`
            RefundRate float64 `json:"refundRate"`
        } `json:"refunds"`
        MethodDistribution []struct {
            MethodType  string  `json:"methodType"`
            Count       int     `json:"count"`
            TotalAmount string  `json:"totalAmount"`
            Percentage  float64 `json:"percentage"`
        } `json:"methodDistribution"`
        CardBrandDistribution []struct {
            Brand       string  `json:"brand"`
            Count       int     `json:"count"`
            TotalAmount string  `json:"totalAmount"`
            Percentage  float64 `json:"percentage"`
        } `json:"cardBrandDistribution"`
        TaxSummary []struct {
            Currency     string `json:"currency"`
            TotalTax     string `json:"totalTax"`
            TotalPreTax  string `json:"totalPreTax"`
            TotalAmount  string `json:"totalAmount"`
            PaymentCount int    `json:"paymentCount"`
        } `json:"taxSummary"`
        PreTaxRevenueByCurrency []struct {
            Currency     string `json:"currency"`
            TotalAmount  string `json:"totalAmount"`
            PaymentCount int    `json:"paymentCount"`
        } `json:"preTaxRevenueByCurrency"`
        SettlementRevenueByCurrency []struct {
            Currency     string `json:"currency"`
            TotalAmount  string `json:"totalAmount"`
            PaymentCount int    `json:"paymentCount"`
        } `json:"settlementRevenueByCurrency"`
    } `json:"paymentStatistics"`
}
paymentStats, err := pancake.GraphQLQuery[PaymentStatsQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        paymentStatistics(
            storeId: $storeId,
            filter: { timeRange: { startDate: "2025-01-01T00:00:00Z", endDate: "2026-01-01T00:00:00Z" } }
        ) {
            successRate { totalAttempts succeeded failed pending successRate }
            failedReasons { reason count percentage }
            refunds { totalCount succeededCount pendingCount failedCount amountByCurrency { currency totalAmount paymentCount } refundRate }
            methodDistribution { methodType count totalAmount percentage }
            cardBrandDistribution { brand count totalAmount percentage }
            taxSummary { currency totalTax totalPreTax totalAmount paymentCount }
            preTaxRevenueByCurrency { currency totalAmount paymentCount }
            settlementRevenueByCurrency { currency totalAmount paymentCount }
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### productStatistics — Product Rankings & Revenue

```go
type ProductStatsQuery struct {
    ProductStatistics struct {
        OnetimeCountsByStatus []struct {
            Status string `json:"status"`
            Count  int    `json:"count"`
        } `json:"onetimeCountsByStatus"`
        SubscriptionCountsByStatus []struct {
            Status string `json:"status"`
            Count  int    `json:"count"`
        } `json:"subscriptionCountsByStatus"`
        OnetimeTotalCount      int `json:"onetimeTotalCount"`
        SubscriptionTotalCount int `json:"subscriptionTotalCount"`
        TopByOrderCount        []struct {
            ProductID    string `json:"productId"`
            ProductType  string `json:"productType"`
            ProductName  string `json:"productName"`
            OrderCount   int    `json:"orderCount"`
            TotalRevenue string `json:"totalRevenue"`
            Currency     string `json:"currency"`
        } `json:"topByOrderCount"`
        TopByRevenue []struct {
            ProductID    string `json:"productId"`
            ProductType  string `json:"productType"`
            ProductName  string `json:"productName"`
            OrderCount   int    `json:"orderCount"`
            TotalRevenue string `json:"totalRevenue"`
            Currency     string `json:"currency"`
        } `json:"topByRevenue"`
        RevenueContribution []struct {
            ProductID              string  `json:"productId"`
            ProductType            string  `json:"productType"`
            ProductName            string  `json:"productName"`
            Revenue                string  `json:"revenue"`
            ContributionPercentage float64 `json:"contributionPercentage"`
            CumulativePercentage   float64 `json:"cumulativePercentage"`
        } `json:"revenueContribution"`
    } `json:"productStatistics"`
}
productStats, err := pancake.GraphQLQuery[ProductStatsQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        productStatistics(
            storeId: $storeId,
            filter: { timeRange: { startDate: "2025-01-01T00:00:00Z", endDate: "2026-01-01T00:00:00Z" } }
        ) {
            onetimeCountsByStatus { status count }
            subscriptionCountsByStatus { status count }
            onetimeTotalCount
            subscriptionTotalCount
            topByOrderCount(limit: 10) { productId productType productName orderCount totalRevenue currency }
            topByRevenue(limit: 10, currency: "usd") { productId productType productName orderCount totalRevenue currency }
            revenueContribution(currency: "usd") { productId productType productName revenue contributionPercentage cumulativePercentage }
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### trendAnalysis — Growth Trends

```go
type TrendsQuery struct {
    TrendAnalysis struct {
        OrderGrowth []struct {
            Period        string  `json:"period"`
            CurrentValue  float64 `json:"currentValue"`
            PreviousValue float64 `json:"previousValue"`
            GrowthRate    float64 `json:"growthRate"`
        } `json:"orderGrowth"`
        RevenueGrowth []struct {
            Period        string  `json:"period"`
            CurrentValue  float64 `json:"currentValue"`
            PreviousValue float64 `json:"previousValue"`
            GrowthRate    float64 `json:"growthRate"`
        } `json:"revenueGrowth"`
        CumulativeRevenue []struct {
            Period          string  `json:"period"`
            PeriodValue     float64 `json:"periodValue"`
            CumulativeValue float64 `json:"cumulativeValue"`
        } `json:"cumulativeRevenue"`
        OrderMovingAverage []struct {
            Date          string  `json:"date"`
            DailyValue    float64 `json:"dailyValue"`
            MovingAverage float64 `json:"movingAverage"`
        } `json:"orderMovingAverage"`
        RevenueMovingAverage []struct {
            Date          string  `json:"date"`
            DailyValue    float64 `json:"dailyValue"`
            MovingAverage float64 `json:"movingAverage"`
        } `json:"revenueMovingAverage"`
    } `json:"trendAnalysis"`
}
trends, err := pancake.GraphQLQuery[TrendsQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        trendAnalysis(
            storeId: $storeId,
            filter: { timeRange: { startDate: "2025-01-01T00:00:00Z", endDate: "2026-01-01T00:00:00Z" } }
        ) {
            orderGrowth(granularity: MONTH) { period currentValue previousValue growthRate }
            revenueGrowth(granularity: MONTH, currency: "usd") { period currentValue previousValue growthRate }
            cumulativeRevenue(granularity: MONTH, currency: "usd") { period periodValue cumulativeValue }
            orderMovingAverage(windowDays: 7) { date dailyValue movingAverage }
            revenueMovingAverage(windowDays: 7, currency: "usd") { date dailyValue movingAverage }
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### distributionAnalysis — Amount Distribution & AOV

```go
type DistributionQuery struct {
    DistributionAnalysis struct {
        OrderAmountPercentiles struct {
            P10    float64 `json:"p10"`
            P25    float64 `json:"p25"`
            P50    float64 `json:"p50"`
            P75    float64 `json:"p75"`
            P90    float64 `json:"p90"`
            P95    float64 `json:"p95"`
            P99    float64 `json:"p99"`
            Min    float64 `json:"min"`
            Max    float64 `json:"max"`
            Avg    float64 `json:"avg"`
            Stddev float64 `json:"stddev"`
            Count  int     `json:"count"`
        } `json:"orderAmountPercentiles"`
        AOVTrend []struct {
            Period            string  `json:"period"`
            AverageOrderValue float64 `json:"averageOrderValue"`
            OrderCount        int     `json:"orderCount"`
            TotalRevenue      string  `json:"totalRevenue"`
        } `json:"aovTrend"`
        OrderAmountBuckets []struct {
            RangeMin   float64 `json:"rangeMin"`
            RangeMax   float64 `json:"rangeMax"`
            Count      int     `json:"count"`
            Percentage float64 `json:"percentage"`
        } `json:"orderAmountBuckets"`
    } `json:"distributionAnalysis"`
}
distribution, err := pancake.GraphQLQuery[DistributionQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        distributionAnalysis(
            storeId: $storeId,
            filter: { timeRange: { startDate: "2025-01-01T00:00:00Z", endDate: "2026-01-01T00:00:00Z" } }
        ) {
            orderAmountPercentiles(currency: "usd") { p10 p25 p50 p75 p90 p95 p99 min max avg stddev count }
            aovTrend(granularity: MONTH, currency: "usd") { period averageOrderValue orderCount totalRevenue }
            orderAmountBuckets(currency: "usd", bucketCount: 10) { rangeMin rangeMax count percentage }
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### customerAnalysis — Retention, LTV & Repeat Purchases

```go
type CustomersQuery struct {
    CustomerAnalysis struct {
        CohortRetention []struct {
            CohortPeriod string `json:"cohortPeriod"`
            CohortSize   int    `json:"cohortSize"`
            Retention    []struct {
                PeriodOffset    int     `json:"periodOffset"`
                ActiveCustomers int     `json:"activeCustomers"`
                RetentionRate   float64 `json:"retentionRate"`
            } `json:"retention"`
        } `json:"cohortRetention"`
        LTVDistribution struct {
            AverageLtv float64 `json:"averageLtv"`
            MedianLtv  float64 `json:"medianLtv"`
            Buckets    []struct {
                RangeMin   float64 `json:"rangeMin"`
                RangeMax   float64 `json:"rangeMax"`
                Count      int     `json:"count"`
                Percentage float64 `json:"percentage"`
            } `json:"buckets"`
        } `json:"ltvDistribution"`
        PurchaseFrequency []struct {
            PurchaseCount int     `json:"purchaseCount"`
            CustomerCount int     `json:"customerCount"`
            Percentage    float64 `json:"percentage"`
        } `json:"purchaseFrequency"`
        RepeatPurchaseRate []struct {
            Period       string  `json:"period"`
            TotalBuyers  int     `json:"totalBuyers"`
            RepeatBuyers int     `json:"repeatBuyers"`
            RepeatRate   float64 `json:"repeatRate"`
        } `json:"repeatPurchaseRate"`
        TopCustomers []struct {
            BuyerEmail        string `json:"buyerEmail"`
            TotalSpent        string `json:"totalSpent"`
            OrderCount        int    `json:"orderCount"`
            FirstPurchaseDate string `json:"firstPurchaseDate"`
            LastPurchaseDate  string `json:"lastPurchaseDate"`
        } `json:"topCustomers"`
    } `json:"customerAnalysis"`
}
customers, err := pancake.GraphQLQuery[CustomersQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        customerAnalysis(
            storeId: $storeId,
            filter: { timeRange: { startDate: "2025-01-01T00:00:00Z", endDate: "2026-01-01T00:00:00Z" } }
        ) {
            cohortRetention(granularity: MONTH) {
                cohortPeriod cohortSize
                retention { periodOffset activeCustomers retentionRate }
            }
            ltvDistribution(currency: "usd") {
                averageLtv medianLtv
                buckets { rangeMin rangeMax count percentage }
            }
            purchaseFrequency { purchaseCount customerCount percentage }
            repeatPurchaseRate(granularity: MONTH) { period totalBuyers repeatBuyers repeatRate }
            topCustomers(limit: 10, currency: "usd") { buyerEmail totalSpent orderCount firstPurchaseDate lastPurchaseDate }
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### taxAnalysis — Tax Breakdown

```go
type TaxQuery struct {
    TaxAnalysis struct {
        ByCategory []struct {
            TaxCategory  string `json:"taxCategory"`
            TotalTax     string `json:"totalTax"`
            TotalAmount  string `json:"totalAmount"`
            PaymentCount int    `json:"paymentCount"`
        } `json:"byCategory"`
        ByRateGroup []struct {
            TaxRate      float64 `json:"taxRate"`
            TotalTax     string  `json:"totalTax"`
            TotalAmount  string  `json:"totalAmount"`
            PaymentCount int     `json:"paymentCount"`
        } `json:"byRateGroup"`
        ByCountry []struct {
            Country      string `json:"country"`
            TotalTax     string `json:"totalTax"`
            TotalAmount  string `json:"totalAmount"`
            PaymentCount int    `json:"paymentCount"`
        } `json:"byCountry"`
        B2BVsB2C []struct {
            IsBusiness  bool   `json:"isBusiness"`
            Label       string `json:"label"`
            TotalTax    string `json:"totalTax"`
            TotalAmount string `json:"totalAmount"`
            OrderCount  int    `json:"orderCount"`
        } `json:"b2bVsB2c"`
        EffectiveTaxRateTrend []struct {
            Period       string  `json:"period"`
            AvgTaxRate   float64 `json:"avgTaxRate"`
            PaymentCount int     `json:"paymentCount"`
        } `json:"effectiveTaxRateTrend"`
        TaxAmountByPeriod []struct {
            Period       string `json:"period"`
            TotalTax     string `json:"totalTax"`
            PaymentCount int    `json:"paymentCount"`
        } `json:"taxAmountByPeriod"`
    } `json:"taxAnalysis"`
}
tax, err := pancake.GraphQLQuery[TaxQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        taxAnalysis(
            storeId: $storeId,
            filter: { timeRange: { startDate: "2025-01-01T00:00:00Z", endDate: "2026-01-01T00:00:00Z" } }
        ) {
            byCategory(currency: "usd") { taxCategory totalTax totalAmount paymentCount }
            byRateGroup(currency: "usd") { taxRate totalTax totalAmount paymentCount }
            byCountry(currency: "usd") { country totalTax totalAmount paymentCount }
            b2bVsB2c(currency: "usd") { isBusiness label totalTax totalAmount orderCount }
            effectiveTaxRateTrend(granularity: MONTH, currency: "usd") { period avgTaxRate paymentCount }
            taxAmountByPeriod(granularity: MONTH, currency: "usd") { period totalTax paymentCount }
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### subscriptionAnalysis — Churn, Trial Conversion & Billing

```go
type SubscriptionsQuery struct {
    SubscriptionAnalysis struct {
        BillingPeriodDistribution []struct {
            BillingPeriod string  `json:"billingPeriod"`
            Count         int     `json:"count"`
            TotalAmount   string  `json:"totalAmount"`
            Percentage    float64 `json:"percentage"`
        } `json:"billingPeriodDistribution"`
        ActiveCount       int `json:"activeCount"`
        CancellationStats struct {
            TotalSubscriptions  int     `json:"totalSubscriptions"`
            CanceledCount       int     `json:"canceledCount"`
            CancellationRate    float64 `json:"cancellationRate"`
            AvgLifetimeDays     float64 `json:"avgLifetimeDays"`
            MedianLifetimeDays  float64 `json:"medianLifetimeDays"`
        } `json:"cancellationStats"`
        CancellationTrend []struct {
            Period        string `json:"period"`
            CanceledCount int    `json:"canceledCount"`
        } `json:"cancellationTrend"`
        TrialConversion struct {
            TotalTrials    int     `json:"totalTrials"`
            ConvertedCount int     `json:"convertedCount"`
            ActiveTrials   int     `json:"activeTrials"`
            ConversionRate float64 `json:"conversionRate"`
        } `json:"trialConversion"`
        TrialConversionByProduct []struct {
            ProductID      string  `json:"productId"`
            ProductName    string  `json:"productName"`
            TotalTrials    int     `json:"totalTrials"`
            ConvertedCount int     `json:"convertedCount"`
            ConversionRate float64 `json:"conversionRate"`
        } `json:"trialConversionByProduct"`
        ChurnRate []struct {
            Period      string  `json:"period"`
            StartActive int     `json:"startActive"`
            Churned     int     `json:"churned"`
            ChurnRate   float64 `json:"churnRate"`
        } `json:"churnRate"`
    } `json:"subscriptionAnalysis"`
}
subscriptions, err := pancake.GraphQLQuery[SubscriptionsQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        subscriptionAnalysis(
            storeId: $storeId,
            filter: { timeRange: { startDate: "2025-01-01T00:00:00Z", endDate: "2026-01-01T00:00:00Z" } }
        ) {
            billingPeriodDistribution(currency: "usd") { billingPeriod count totalAmount percentage }
            activeCount
            cancellationStats { totalSubscriptions canceledCount cancellationRate avgLifetimeDays medianLifetimeDays }
            cancellationTrend(granularity: MONTH) { period canceledCount }
            trialConversion { totalTrials convertedCount activeTrials conversionRate }
            trialConversionByProduct { productId productName totalTrials convertedCount conversionRate }
            churnRate(granularity: MONTH) { period startActive churned churnRate }
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```

### refundTicketAnalysis — Refund Reasons & Review Efficiency

```go
type RefundAnalysisQuery struct {
    RefundTicketAnalysis struct {
        ReasonDistribution []struct {
            Reason      string  `json:"reason"`
            Count       int     `json:"count"`
            TotalAmount string  `json:"totalAmount"`
            Percentage  float64 `json:"percentage"`
        } `json:"reasonDistribution"`
        StatusDistribution []struct {
            Status     string  `json:"status"`
            Count      int     `json:"count"`
            Percentage float64 `json:"percentage"`
        } `json:"statusDistribution"`
        ReviewEfficiency struct {
            AvgHours      float64 `json:"avgHours"`
            MedianHours   float64 `json:"medianHours"`
            P90Hours      float64 `json:"p90Hours"`
            TotalReviewed int     `json:"totalReviewed"`
        } `json:"reviewEfficiency"`
        TicketTrend []struct {
            Period         string `json:"period"`
            TotalCreated   int    `json:"totalCreated"`
            ResolvedCount  int    `json:"resolvedCount"`
            ApprovedCount  int    `json:"approvedCount"`
            RejectedCount  int    `json:"rejectedCount"`
        } `json:"ticketTrend"`
        ApprovalRate struct {
            Approved int     `json:"approved"`
            Rejected int     `json:"rejected"`
            Rate     float64 `json:"rate"`
        } `json:"approvalRate"`
        ProcessingSuccessRate struct {
            Succeeded int     `json:"succeeded"`
            Failed    int     `json:"failed"`
            Rate      float64 `json:"rate"`
        } `json:"processingSuccessRate"`
    } `json:"refundTicketAnalysis"`
}
refundAnalysis, err := pancake.GraphQLQuery[RefundAnalysisQuery](ctx, client, pancake.GraphQLParams{
    Query: `query ($storeId: String!) {
        refundTicketAnalysis(
            storeId: $storeId,
            filter: { timeRange: { startDate: "2025-01-01T00:00:00Z", endDate: "2026-01-01T00:00:00Z" } }
        ) {
            reasonDistribution(currency: "usd") { reason count totalAmount percentage }
            statusDistribution { status count percentage }
            reviewEfficiency { avgHours medianHours p90Hours totalReviewed }
            ticketTrend(granularity: MONTH) { period totalCreated resolvedCount approvedCount rejectedCount }
            approvalRate { approved rejected rate }
            processingSuccessRate { succeeded failed rate }
        }
    }`,
    Variables: map[string]any{"storeId": "STO_xxx"},
})
```
