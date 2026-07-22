package pancake

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestResources_HappyPaths covers the happy path of every merchant-scoped
// resource method by stubbing the HTTP layer with a fixed envelope-shaped
// response and asserting that (a) the right endpoint was hit and (b) the
// envelope was unwrapped into the typed result.
func TestResources_HappyPaths(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name     string
		path     string
		response map[string]any // value of the envelope "data" field
		call     func(c *Client) error
	}{
		// ---- Auth ----
		{
			name:     "Auth.IssueSessionToken (storeId)",
			path:     "/v1/actions/auth/issue-session-token",
			response: map[string]any{"token": "JWT", "expiresAt": "2026-12-01T00:00:00Z"},
			call: func(c *Client) error {
				_, err := c.Auth.IssueSessionToken(ctx, IssueSessionTokenParams{
					StoreID:       Ptr("STO_AbCdEfGhIjKlMnOpQrStUv"),
					BuyerIdentity: "user@x.com",
				})
				return err
			},
		},
		{
			name:     "Auth.IssueSessionToken (productId)",
			path:     "/v1/actions/auth/issue-session-token",
			response: map[string]any{"token": "JWT2", "expiresAt": "z"},
			call: func(c *Client) error {
				_, err := c.Auth.IssueSessionToken(ctx, IssueSessionTokenParams{
					ProductID:     Ptr("PROD_AbCdEfGhIjKlMnOpQrStUv"),
					BuyerIdentity: "user@x.com",
				})
				return err
			},
		},

		// ---- Stores ----
		{
			name:     "Stores.Delete",
			path:     "/v1/actions/store/delete-store",
			response: map[string]any{"store": map[string]string{"id": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "X"}},
			call: func(c *Client) error {
				_, err := c.Stores.Delete(ctx, DeleteStoreParams{ID: "STO_AbCdEfGhIjKlMnOpQrStUv"})
				return err
			},
		},
		{
			name:     "Stores.Update with Nullable Logo",
			path:     "/v1/actions/store/update-store",
			response: map[string]any{"store": map[string]string{"id": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "X"}},
			call: func(c *Client) error {
				_, err := c.Stores.Update(ctx, UpdateStoreParams{
					ID:           "STO_AbCdEfGhIjKlMnOpQrStUv",
					Name:         Ptr("Renamed"),
					Logo:         ExplicitNullPtr[string](),
					SupportEmail: NullValuePtr("a@b.c"),
				})
				return err
			},
		},

		// ---- StoreMerchants ----
		{
			name:     "StoreMerchants.Add",
			path:     "/v1/actions/store-merchant/add-merchant",
			response: map[string]any{"storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "merchantId": "MER_AbCdEfGhIjKlMnOpQrStUv", "email": "a@b.c", "role": "admin", "status": "pending", "addedAt": "z"},
			call: func(c *Client) error {
				_, err := c.StoreMerchants.Add(ctx, AddMerchantParams{
					StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv",
					Email:   "a@b.c",
					Role:    "admin",
				})
				return err
			},
		},
		{
			name:     "StoreMerchants.Remove",
			path:     "/v1/actions/store-merchant/remove-merchant",
			response: map[string]any{"message": "ok", "removedAt": "z"},
			call: func(c *Client) error {
				_, err := c.StoreMerchants.Remove(ctx, RemoveMerchantParams{
					StoreID:    "STO_AbCdEfGhIjKlMnOpQrStUv",
					MerchantID: "MER_AbCdEfGhIjKlMnOpQrStUv",
				})
				return err
			},
		},
		{
			name:     "StoreMerchants.UpdateRole",
			path:     "/v1/actions/store-merchant/update-role",
			response: map[string]any{"storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "merchantId": "MER_AbCdEfGhIjKlMnOpQrStUv", "role": "member", "updatedAt": "z"},
			call: func(c *Client) error {
				_, err := c.StoreMerchants.UpdateRole(ctx, UpdateRoleParams{
					StoreID:    "STO_AbCdEfGhIjKlMnOpQrStUv",
					MerchantID: "MER_AbCdEfGhIjKlMnOpQrStUv",
					Role:       "member",
				})
				return err
			},
		},

		// ---- OnetimeProducts ----
		{
			name:     "OnetimeProducts.Create",
			path:     "/v1/actions/onetime-product/create-product",
			response: map[string]any{"product": map[string]any{"id": "PROD_AbCdEfGhIjKlMnOpQrStUv", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "X", "prices": map[string]any{}, "media": []any{}, "metadata": map[string]any{}, "status": "active", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.OnetimeProducts.Create(ctx, CreateOnetimeProductParams{
					StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv",
					Name:    "X",
					Prices:  Prices{"USD": {Amount: "9.99", TaxCategory: TaxCategoryDigitalGoods}},
				})
				return err
			},
		},
		{
			name:     "OnetimeProducts.Update",
			path:     "/v1/actions/onetime-product/update-product",
			response: map[string]any{"product": map[string]any{"id": "PROD_AbCdEfGhIjKlMnOpQrStUv", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "Y", "prices": map[string]any{}, "media": []any{}, "metadata": map[string]any{}, "status": "active", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.OnetimeProducts.Update(ctx, UpdateOnetimeProductParams{
					ID:     "PROD_AbCdEfGhIjKlMnOpQrStUv",
					Name:   Ptr("Y"),
					Prices: Prices{"USD": {Amount: "19.99", TaxCategory: TaxCategoryDigitalGoods}},
				})
				return err
			},
		},
		{
			name:     "OnetimeProducts.Publish",
			path:     "/v1/actions/onetime-product/publish-product",
			response: map[string]any{"product": map[string]any{"id": "PROD_AbCdEfGhIjKlMnOpQrStUv", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "X", "prices": map[string]any{}, "media": []any{}, "metadata": map[string]any{}, "status": "active", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.OnetimeProducts.Publish(ctx, PublishOnetimeProductParams{ID: "PROD_AbCdEfGhIjKlMnOpQrStUv"})
				return err
			},
		},
		{
			name:     "OnetimeProducts.UpdateStatus",
			path:     "/v1/actions/onetime-product/update-status",
			response: map[string]any{"product": map[string]any{"id": "PROD_AbCdEfGhIjKlMnOpQrStUv", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "X", "prices": map[string]any{}, "media": []any{}, "metadata": map[string]any{}, "status": "inactive", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.OnetimeProducts.UpdateStatus(ctx, UpdateOnetimeStatusParams{
					ID:     "PROD_AbCdEfGhIjKlMnOpQrStUv",
					Status: ProductVersionStatusInactive,
				})
				return err
			},
		},

		// ---- SubscriptionProducts ----
		{
			name:     "SubscriptionProducts.Create",
			path:     "/v1/actions/subscription-product/create-product",
			response: map[string]any{"product": map[string]any{"id": "PROD_AbCdEfGhIjKlMnOpQrStUv", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "Pro", "billingPeriod": "monthly", "prices": map[string]any{}, "media": []any{}, "metadata": map[string]any{}, "status": "active", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.SubscriptionProducts.Create(ctx, CreateSubscriptionProductParams{
					StoreID:       "STO_AbCdEfGhIjKlMnOpQrStUv",
					Name:          "Pro",
					BillingPeriod: BillingPeriodMonthly,
					Prices:        Prices{"USD": {Amount: "9.99", TaxCategory: TaxCategorySaaS}},
				})
				return err
			},
		},
		{
			name:     "SubscriptionProducts.Update",
			path:     "/v1/actions/subscription-product/update-product",
			response: map[string]any{"product": map[string]any{"id": "PROD_AbCdEfGhIjKlMnOpQrStUv", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "ProV2", "billingPeriod": "yearly", "prices": map[string]any{}, "media": []any{}, "metadata": map[string]any{}, "status": "active", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				bp := BillingPeriodYearly
				_, err := c.SubscriptionProducts.Update(ctx, UpdateSubscriptionProductParams{
					ID:            "PROD_AbCdEfGhIjKlMnOpQrStUv",
					Name:          Ptr("ProV2"),
					BillingPeriod: &bp,
					Prices:        Prices{"USD": {Amount: "99.99", TaxCategory: TaxCategorySaaS}},
				})
				return err
			},
		},
		{
			name:     "SubscriptionProducts.Publish",
			path:     "/v1/actions/subscription-product/publish-product",
			response: map[string]any{"product": map[string]any{"id": "PROD_AbCdEfGhIjKlMnOpQrStUv", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "Pro", "billingPeriod": "monthly", "prices": map[string]any{}, "media": []any{}, "metadata": map[string]any{}, "status": "active", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.SubscriptionProducts.Publish(ctx, PublishSubscriptionProductParams{ID: "PROD_AbCdEfGhIjKlMnOpQrStUv"})
				return err
			},
		},
		{
			name:     "SubscriptionProducts.UpdateStatus",
			path:     "/v1/actions/subscription-product/update-status",
			response: map[string]any{"product": map[string]any{"id": "PROD_AbCdEfGhIjKlMnOpQrStUv", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "Pro", "billingPeriod": "monthly", "prices": map[string]any{}, "media": []any{}, "metadata": map[string]any{}, "status": "inactive", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.SubscriptionProducts.UpdateStatus(ctx, UpdateSubscriptionStatusParams{
					ID:     "PROD_AbCdEfGhIjKlMnOpQrStUv",
					Status: ProductVersionStatusInactive,
				})
				return err
			},
		},

		// ---- SubscriptionProductGroups ----
		{
			name:     "SubscriptionProductGroups.Create",
			path:     "/v1/actions/subscription-product-group/create-group",
			response: map[string]any{"group": map[string]any{"id": "GRP_x", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "G", "rules": map[string]any{"sharedTrial": true}, "productIds": []any{}, "environment": "test", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.SubscriptionProductGroups.Create(ctx, CreateSubscriptionProductGroupParams{
					StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv",
					Name:    "G",
				})
				return err
			},
		},
		{
			name:     "SubscriptionProductGroups.Update",
			path:     "/v1/actions/subscription-product-group/update-group",
			response: map[string]any{"group": map[string]any{"id": "GRP_x", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "G", "rules": map[string]any{"sharedTrial": false}, "productIds": []any{"PROD_AbCdEfGhIjKlMnOpQrStUv"}, "environment": "test", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.SubscriptionProductGroups.Update(ctx, UpdateSubscriptionProductGroupParams{
					ID:         "GRP_x",
					ProductIDs: []string{"PROD_AbCdEfGhIjKlMnOpQrStUv"},
				})
				return err
			},
		},
		{
			name:     "SubscriptionProductGroups.Delete",
			path:     "/v1/actions/subscription-product-group/delete-group",
			response: map[string]any{"group": map[string]any{"id": "GRP_x", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "G", "rules": map[string]any{"sharedTrial": false}, "productIds": []any{}, "environment": "test", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.SubscriptionProductGroups.Delete(ctx, DeleteSubscriptionProductGroupParams{ID: "GRP_x"})
				return err
			},
		},
		{
			name:     "SubscriptionProductGroups.Publish",
			path:     "/v1/actions/subscription-product-group/publish-group",
			response: map[string]any{"group": map[string]any{"id": "GRP_x", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "G", "rules": map[string]any{"sharedTrial": false}, "productIds": []any{}, "environment": "prod", "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.SubscriptionProductGroups.Publish(ctx, PublishSubscriptionProductGroupParams{ID: "GRP_x"})
				return err
			},
		},

		// ---- Orders ----
		{
			name:     "Orders.CancelSubscription",
			path:     "/v1/actions/subscription-order/cancel-order",
			response: map[string]any{"orderId": "ORD_AbCdEfGhIjKlMnOpQrStUv", "status": "canceling"},
			call: func(c *Client) error {
				_, err := c.Orders.CancelSubscription(ctx, CancelSubscriptionParams{OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv"})
				return err
			},
		},

		// ---- Checkout (low-level + anonymous; authenticated is in http_client_test) ----
		{
			name:     "Checkout.CreateSession",
			path:     "/v1/actions/checkout/create-session",
			response: map[string]any{"sessionId": "ses", "checkoutUrl": "https://x/y", "expiresAt": "z"},
			call: func(c *Client) error {
				_, err := c.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
					ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
					Currency:  "USD",
				})
				return err
			},
		},
		{
			name:     "Checkout.Anonymous.Create",
			path:     "/v1/actions/checkout/create-session",
			response: map[string]any{"sessionId": "ses", "checkoutUrl": "https://x/y", "expiresAt": "z"},
			call: func(c *Client) error {
				_, err := c.Checkout.Anonymous.Create(ctx, AnonymousCheckoutParams{
					ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
					Currency:  "USD",
				})
				return err
			},
		},

		// ---- Webhooks management ----
		{
			name:     "Webhooks.Add",
			path:     "/v1/actions/store/add-webhook",
			response: map[string]any{"webhook": map[string]any{"id": "wh_1", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "channel": "http", "url": "https://x", "events": []any{"order.completed"}, "testMode": false, "secret": nil, "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.Webhooks.Add(ctx, AddWebhookParams{
					StoreID:  "STO_AbCdEfGhIjKlMnOpQrStUv",
					Channel:  WebhookChannelHTTP,
					URL:      "https://x",
					Events:   []WebhookEventType{WebhookEventTypeOrderCompleted},
					TestMode: false,
				})
				return err
			},
		},
		{
			name:     "Webhooks.Update",
			path:     "/v1/actions/store/update-webhook",
			response: map[string]any{"webhook": map[string]any{"id": "wh_1", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "channel": "http", "url": "https://y", "events": []any{}, "testMode": false, "secret": nil, "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.Webhooks.Update(ctx, UpdateWebhookParams{ID: "wh_1", URL: Ptr("https://y")})
				return err
			},
		},
		{
			name:     "Webhooks.Remove",
			path:     "/v1/actions/store/remove-webhook",
			response: map[string]any{"webhook": map[string]any{"id": "wh_1", "storeId": "STO_AbCdEfGhIjKlMnOpQrStUv", "channel": "http", "url": "https://x", "events": []any{}, "testMode": false, "secret": nil, "createdAt": "z", "updatedAt": "z"}},
			call: func(c *Client) error {
				_, err := c.Webhooks.Remove(ctx, RemoveWebhookParams{ID: "wh_1"})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _, server := newSignedTestClient(t)
			server.respond = func(_ recordedRequest) (int, any) {
				return 200, map[string]any{"data": tc.response}
			}
			if err := tc.call(client); err != nil {
				t.Fatalf("call returned err: %v", err)
			}
			reqs := server.requests()
			if len(reqs) == 0 {
				t.Fatal("no request recorded")
			}
			// Confirm the expected endpoint was hit (anywhere in the recorded list).
			hit := false
			for _, r := range reqs {
				if r.Path == tc.path {
					hit = true
					break
				}
			}
			if !hit {
				paths := make([]string, len(reqs))
				for i, r := range reqs {
					paths[i] = r.Path
				}
				t.Errorf("expected path %q hit, got: %v", tc.path, paths)
			}
		})
	}
}

// TestCheckout_PaymentMethods covers the ordered payment-method allow-list:
// it must serialize in the given order when provided, and be entirely absent
// from the request body (not just null) when omitted.
func TestCheckout_PaymentMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("serializes in the given order when provided", func(t *testing.T) {
		client, _, server := newSignedTestClient(t)
		server.respond = func(_ recordedRequest) (int, any) {
			return 200, map[string]any{"data": map[string]any{"sessionId": "ses", "checkoutUrl": "https://x/y", "expiresAt": "z"}}
		}

		_, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
			ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
			Currency:       "USD",
			PaymentMethods: []PaymentMethod{PaymentMethodApplePay, PaymentMethodCard},
		})
		if err != nil {
			t.Fatalf("call returned err: %v", err)
		}

		reqs := server.requests()
		if len(reqs) == 0 {
			t.Fatal("no request recorded")
		}
		var body map[string]any
		if err := json.Unmarshal(reqs[0].Body, &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		methods, ok := body["paymentMethods"].([]any)
		if !ok {
			t.Fatalf("expected paymentMethods array in body, got: %v", body["paymentMethods"])
		}
		if len(methods) != 2 || methods[0] != "applepay" || methods[1] != "card" {
			t.Errorf("expected [applepay, card] in order, got: %v", methods)
		}
	})

	t.Run("omitted entirely from the request body when not provided", func(t *testing.T) {
		client, _, server := newSignedTestClient(t)
		server.respond = func(_ recordedRequest) (int, any) {
			return 200, map[string]any{"data": map[string]any{"sessionId": "ses", "checkoutUrl": "https://x/y", "expiresAt": "z"}}
		}

		_, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
			ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
			Currency:  "USD",
		})
		if err != nil {
			t.Fatalf("call returned err: %v", err)
		}

		reqs := server.requests()
		if len(reqs) == 0 {
			t.Fatal("no request recorded")
		}
		if strings.Contains(string(reqs[0].Body), "paymentMethods") {
			t.Errorf("expected no paymentMethods key in body, got: %s", reqs[0].Body)
		}
	})
}

// TestResources_ValidationFailures covers the SDK-layer rejection branches
// (validateShortID, validateRequired, validatePrices, etc).
func TestResources_ValidationFailures(t *testing.T) {
	ctx := context.Background()
	client, _, _ := newSignedTestClient(t)

	cases := []struct {
		name string
		call func() error
	}{
		{"Auth.IssueSessionToken with no store/product", func() error {
			_, err := client.Auth.IssueSessionToken(ctx, IssueSessionTokenParams{BuyerIdentity: "x"})
			return err
		}},
		{"Auth.IssueSessionToken bad storeId", func() error {
			_, err := client.Auth.IssueSessionToken(ctx, IssueSessionTokenParams{
				StoreID:       Ptr("bad"),
				BuyerIdentity: "x",
			})
			return err
		}},
		{"Auth.IssueSessionToken bad productId", func() error {
			_, err := client.Auth.IssueSessionToken(ctx, IssueSessionTokenParams{
				ProductID:     Ptr("bad"),
				BuyerIdentity: "x",
			})
			return err
		}},
		{"Auth.IssueSessionToken empty buyerIdentity", func() error {
			_, err := client.Auth.IssueSessionToken(ctx, IssueSessionTokenParams{
				StoreID: Ptr("STO_AbCdEfGhIjKlMnOpQrStUv"),
			})
			return err
		}},
		{"Stores.Create empty name", func() error {
			_, err := client.Stores.Create(ctx, CreateStoreParams{})
			return err
		}},
		{"Stores.Delete bad id", func() error {
			_, err := client.Stores.Delete(ctx, DeleteStoreParams{ID: "bad"})
			return err
		}},
		{"StoreMerchants.Add bad role", func() error {
			_, err := client.StoreMerchants.Add(ctx, AddMerchantParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv",
				Email:   "a@b.c",
				Role:    "owner", // not allowed
			})
			return err
		}},
		{"OnetimeProducts.Create empty prices", func() error {
			_, err := client.OnetimeProducts.Create(ctx, CreateOnetimeProductParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv",
				Name:    "X",
			})
			return err
		}},
		{"OnetimeProducts.UpdateStatus bad status", func() error {
			_, err := client.OnetimeProducts.UpdateStatus(ctx, UpdateOnetimeStatusParams{
				ID:     "PROD_AbCdEfGhIjKlMnOpQrStUv",
				Status: "garbage",
			})
			return err
		}},
		{"SubscriptionProducts.Create bad billingPeriod", func() error {
			_, err := client.SubscriptionProducts.Create(ctx, CreateSubscriptionProductParams{
				StoreID:       "STO_AbCdEfGhIjKlMnOpQrStUv",
				Name:          "X",
				BillingPeriod: "decade",
				Prices:        Prices{"USD": {Amount: "1", TaxCategory: TaxCategorySaaS}},
			})
			return err
		}},
		{"SubscriptionProductGroups.Update missing id", func() error {
			_, err := client.SubscriptionProductGroups.Update(ctx, UpdateSubscriptionProductGroupParams{})
			return err
		}},
		{"Orders.CancelSubscription bad order id", func() error {
			_, err := client.Orders.CancelSubscription(ctx, CancelSubscriptionParams{OrderID: "bad"})
			return err
		}},
		{"Checkout.CreateSession missing currency", func() error {
			_, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
				ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
			})
			return err
		}},
		{"Checkout.Authenticated.Create missing buyerIdentity", func() error {
			_, err := client.Checkout.Authenticated.Create(ctx, AuthenticatedCheckoutParams{
				CreateCheckoutSessionParams: CreateCheckoutSessionParams{
					ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
					Currency:  "USD",
				},
			})
			return err
		}},
		{"GraphQL.Query empty", func() error {
			_, err := client.GraphQL.Query(ctx, GraphQLParams{})
			return err
		}},
		{"Webhooks.Add bad store id", func() error {
			_, err := client.Webhooks.Add(ctx, AddWebhookParams{
				StoreID:  "bad",
				Channel:  WebhookChannelHTTP,
				URL:      "https://x",
				Events:   []WebhookEventType{WebhookEventTypeOrderCompleted},
				TestMode: false,
			})
			return err
		}},
		{"Webhooks.Update missing id", func() error {
			_, err := client.Webhooks.Update(ctx, UpdateWebhookParams{})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatal("expected validation error")
			}
			pe, ok := err.(*Error)
			if !ok {
				t.Fatalf("error type %T, want *Error", err)
			}
			if pe.Status != 400 || pe.Errors[0].Layer != ErrorLayerSDK {
				t.Errorf("expected sdk-layer 400 error, got status=%d layer=%q", pe.Status, pe.Errors[0].Layer)
			}
		})
	}
}

// TestGraphQLQuery_Typed checks the top-level generic helper.
//
// Wire is the standard single-wrap GraphQL envelope. The typed helper
// `GraphQLQuery[T]` unmarshals the inner data block into T.
func TestGraphQLQuery_Typed(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{
			"data": map[string]any{
				"stores": []map[string]string{{"id": "STO_a", "name": "X"}},
			},
		}
	}
	type Result struct {
		Stores []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"stores"`
	}
	resp, err := GraphQLQuery[Result](context.Background(), client, GraphQLParams{Query: "{ stores { id } }"})
	if err != nil {
		t.Fatalf("GraphQLQuery: %v", err)
	}
	if len(resp.Data.Stores) != 1 || resp.Data.Stores[0].Name != "X" {
		t.Fatalf("typed data did not deserialize: %+v", resp.Data)
	}
}

// TestNullable_RoundTripJSON ensures Nullable serializes correctly inside a
// parent struct with omitempty.
func TestNullable_OmitemptyInsidePointer(t *testing.T) {
	type Wrap struct {
		A *Nullable[string] `json:"a,omitempty"`
		B *Nullable[string] `json:"b,omitempty"`
	}
	val := NullValuePtr("yes")
	w := Wrap{A: val, B: ExplicitNullPtr[string]()}
	out, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"a":"yes"`) || !strings.Contains(string(out), `"b":null`) {
		t.Errorf("unexpected JSON: %s", string(out))
	}
	// Nil pointer → omitted.
	w2 := Wrap{}
	out2, _ := json.Marshal(w2)
	if string(out2) != `{}` {
		t.Errorf("nil-pointer omit failed: %s", string(out2))
	}
}

// TestNullable_IsZero
func TestNullable_IsZero(t *testing.T) {
	var n Nullable[string]
	if !n.IsZero() {
		t.Error("zero-value Nullable should report IsZero=true")
	}
	n2 := NullValue("x")
	if n2.IsZero() {
		t.Error("populated Nullable should report IsZero=false")
	}
}
