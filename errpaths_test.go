package pancake

import (
	"context"
	"errors"
	"testing"
)

// TestResources_ErrorPropagation makes the mock server return a 400 envelope
// for every resource method, exercising the `if err != nil { return nil, err }`
// branch after r.http.post(...) in each method.
func TestResources_ErrorPropagation(t *testing.T) {
	ctx := context.Background()

	stub := func(_ recordedRequest) (int, any) {
		return 400, map[string]any{
			"data":   nil,
			"errors": []map[string]string{{"message": "boom", "layer": "store"}},
		}
	}
	stubGraphQL := func(_ recordedRequest) (int, any) {
		return 500, map[string]any{
			"data":   nil,
			"errors": []map[string]string{{"message": "internal", "layer": "graphql"}},
		}
	}

	cases := []struct {
		name string
		call func(c *Client) error
	}{
		{"Auth.IssueSessionToken", func(c *Client) error {
			_, err := c.Auth.IssueSessionToken(ctx, IssueSessionTokenParams{
				StoreID:       Ptr("STO_AbCdEfGhIjKlMnOpQrStUv"),
				BuyerIdentity: "x",
			})
			return err
		}},
		{"Stores.Create", func(c *Client) error {
			_, err := c.Stores.Create(ctx, CreateStoreParams{Name: "X"})
			return err
		}},
		{"Stores.Update", func(c *Client) error {
			_, err := c.Stores.Update(ctx, UpdateStoreParams{ID: "STO_AbCdEfGhIjKlMnOpQrStUv"})
			return err
		}},
		{"Stores.Delete", func(c *Client) error {
			_, err := c.Stores.Delete(ctx, DeleteStoreParams{ID: "STO_AbCdEfGhIjKlMnOpQrStUv"})
			return err
		}},
		{"StoreMerchants.Add", func(c *Client) error {
			_, err := c.StoreMerchants.Add(ctx, AddMerchantParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Email: "a@b.c", Role: "admin",
			})
			return err
		}},
		{"StoreMerchants.Remove", func(c *Client) error {
			_, err := c.StoreMerchants.Remove(ctx, RemoveMerchantParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", MerchantID: "MER_AbCdEfGhIjKlMnOpQrStUv",
			})
			return err
		}},
		{"StoreMerchants.UpdateRole", func(c *Client) error {
			_, err := c.StoreMerchants.UpdateRole(ctx, UpdateRoleParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", MerchantID: "MER_AbCdEfGhIjKlMnOpQrStUv", Role: "admin",
			})
			return err
		}},
		{"OnetimeProducts.Create", func(c *Client) error {
			_, err := c.OnetimeProducts.Create(ctx, CreateOnetimeProductParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Name: "X",
				Prices: Prices{"USD": {Amount: "1", TaxCategory: TaxCategoryDigitalGoods}},
			})
			return err
		}},
		{"OnetimeProducts.Update", func(c *Client) error {
			_, err := c.OnetimeProducts.Update(ctx, UpdateOnetimeProductParams{
				ID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
			})
			return err
		}},
		{"OnetimeProducts.Publish", func(c *Client) error {
			_, err := c.OnetimeProducts.Publish(ctx, PublishOnetimeProductParams{ID: "PROD_AbCdEfGhIjKlMnOpQrStUv"})
			return err
		}},
		{"OnetimeProducts.UpdateStatus", func(c *Client) error {
			_, err := c.OnetimeProducts.UpdateStatus(ctx, UpdateOnetimeStatusParams{
				ID: "PROD_AbCdEfGhIjKlMnOpQrStUv", Status: ProductVersionStatusActive,
			})
			return err
		}},
		{"SubscriptionProducts.Create", func(c *Client) error {
			_, err := c.SubscriptionProducts.Create(ctx, CreateSubscriptionProductParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Name: "X", BillingPeriod: BillingPeriodMonthly,
				Prices: Prices{"USD": {Amount: "1", TaxCategory: TaxCategorySaaS}},
			})
			return err
		}},
		{"SubscriptionProducts.Update", func(c *Client) error {
			_, err := c.SubscriptionProducts.Update(ctx, UpdateSubscriptionProductParams{
				ID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
			})
			return err
		}},
		{"SubscriptionProducts.Publish", func(c *Client) error {
			_, err := c.SubscriptionProducts.Publish(ctx, PublishSubscriptionProductParams{
				ID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
			})
			return err
		}},
		{"SubscriptionProducts.UpdateStatus", func(c *Client) error {
			_, err := c.SubscriptionProducts.UpdateStatus(ctx, UpdateSubscriptionStatusParams{
				ID: "PROD_AbCdEfGhIjKlMnOpQrStUv", Status: ProductVersionStatusActive,
			})
			return err
		}},
		{"SubscriptionProductGroups.Create", func(c *Client) error {
			_, err := c.SubscriptionProductGroups.Create(ctx, CreateSubscriptionProductGroupParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Name: "X",
			})
			return err
		}},
		{"SubscriptionProductGroups.Update", func(c *Client) error {
			_, err := c.SubscriptionProductGroups.Update(ctx, UpdateSubscriptionProductGroupParams{ID: "GRP_x"})
			return err
		}},
		{"SubscriptionProductGroups.Delete", func(c *Client) error {
			_, err := c.SubscriptionProductGroups.Delete(ctx, DeleteSubscriptionProductGroupParams{ID: "GRP_x"})
			return err
		}},
		{"SubscriptionProductGroups.Publish", func(c *Client) error {
			_, err := c.SubscriptionProductGroups.Publish(ctx, PublishSubscriptionProductGroupParams{ID: "GRP_x"})
			return err
		}},
		{"Orders.CancelSubscription", func(c *Client) error {
			_, err := c.Orders.CancelSubscription(ctx, CancelSubscriptionParams{OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv"})
			return err
		}},
		{"Checkout.CreateSession", func(c *Client) error {
			_, err := c.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
				ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv", Currency: "USD",
			})
			return err
		}},
		{"Checkout.Anonymous.Create", func(c *Client) error {
			_, err := c.Checkout.Anonymous.Create(ctx, AnonymousCheckoutParams{
				ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv", Currency: "USD",
			})
			return err
		}},
		{"Webhooks.Add", func(c *Client) error {
			_, err := c.Webhooks.Add(ctx, AddWebhookParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Channel: WebhookChannelHTTP,
				URL: "https://x", Events: []WebhookEventType{WebhookEventTypeOrderCompleted}, TestMode: false,
			})
			return err
		}},
		{"Webhooks.Update", func(c *Client) error {
			_, err := c.Webhooks.Update(ctx, UpdateWebhookParams{ID: "wh_1"})
			return err
		}},
		{"Webhooks.Remove", func(c *Client) error {
			_, err := c.Webhooks.Remove(ctx, RemoveWebhookParams{ID: "wh_1"})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _, server := newSignedTestClient(t)
			server.respond = stub
			err := tc.call(client)
			if err == nil {
				t.Fatal("expected error")
			}
			var perr *Error
			if !errors.As(err, &perr) {
				t.Fatalf("error not *Error: %T", err)
			}
			if perr.Status != 400 || perr.Errors[0].Message != "boom" {
				t.Errorf("unexpected error: %+v", perr)
			}
		})
	}

	// GraphQL gets a 500 to make sure error wrapping still uses APIError-style.
	t.Run("GraphQL.Query (500)", func(t *testing.T) {
		client, _, server := newSignedTestClient(t)
		server.respond = stubGraphQL
		_, err := client.GraphQL.Query(ctx, GraphQLParams{Query: "{ x }"})
		if err == nil {
			t.Fatal("expected error")
		}
		var perr *Error
		if !errors.As(err, &perr) || perr.Status != 500 {
			t.Errorf("unexpected error: %+v", err)
		}
	})

	// And generic helper GraphQLQuery[T] propagates.
	t.Run("GraphQLQuery[T] propagates server error", func(t *testing.T) {
		client, _, server := newSignedTestClient(t)
		server.respond = stubGraphQL
		_, err := GraphQLQuery[map[string]any](ctx, client, GraphQLParams{Query: "{ x }"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestBuyer_ErrorPropagation(t *testing.T) {
	ctx := context.Background()
	_, buyer, srv := newBuyerTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 403, map[string]any{
			"data":   nil,
			"errors": []map[string]string{{"message": "denied", "layer": "user"}},
		}
	}

	cases := []struct {
		name string
		call func() error
	}{
		{"CancelSubscription", func() error {
			_, err := buyer.CancelSubscription(ctx, CancelSubscriptionParams{OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv"})
			return err
		}},
		{"CancelOnetimeOrder", func() error {
			_, err := buyer.CancelOnetimeOrder(ctx, CancelOnetimeOrderParams{OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv"})
			return err
		}},
		{"ReactivateSubscription", func() error {
			_, err := buyer.ReactivateSubscription(ctx, ReactivateSubscriptionParams{OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv"})
			return err
		}},
		{"CreateRefundTicket", func() error {
			_, err := buyer.CreateRefundTicket(ctx, CreateRefundTicketParams{
				PaymentID: "PAY_AbCdEfGhIjKlMnOpQrStUv", Reason: "r",
				RequestedAmount: RequestedAmount{Amount: "1", Currency: "USD"},
			})
			return err
		}},
		{"ResubmitRefundTicket", func() error {
			_, err := buyer.ResubmitRefundTicket(ctx, ResubmitRefundTicketParams{
				TicketID: "TKT_AbCdEfGhIjKlMnOpQrStUv", PaymentID: "PAY_AbCdEfGhIjKlMnOpQrStUv",
				Reason: "r", RequestedAmount: RequestedAmount{Amount: "1", Currency: "USD"},
			})
			return err
		}},
		{"BuyerGraphQL.Query", func() error {
			_, err := buyer.GraphQL.Query(ctx, GraphQLParams{Query: "{ x }"})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

// TestCheckoutCommon_AllValidationBranches covers validatePositiveInt +
// validateBillingDetail + priceSnapshot branches.
func TestCheckoutCommon_AllValidationBranches(t *testing.T) {
	ctx := context.Background()
	client, _, srv := newSignedTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"sessionId": "x", "checkoutUrl": "https://x", "expiresAt": "z"}}
	}
	negative := -1
	zero := 0

	// negative expiresInSeconds
	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID:        "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:         "USD",
		ExpiresInSeconds: &negative,
	}); err == nil {
		t.Error("expected error for negative expiresInSeconds")
	}
	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID:        "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:         "USD",
		ExpiresInSeconds: &zero,
	}); err == nil {
		t.Error("expected error for zero expiresInSeconds")
	}

	// bad priceSnapshot amount
	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID:     "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:      "USD",
		PriceSnapshot: &PriceInfo{Amount: "abc", TaxCategory: TaxCategoryDigitalGoods},
	}); err == nil {
		t.Error("expected error for bad priceSnapshot amount")
	}

	// missing priceSnapshot taxCategory
	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID:     "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:      "USD",
		PriceSnapshot: &PriceInfo{Amount: "9.99", TaxCategory: ""},
	}); err == nil {
		t.Error("expected error for missing taxCategory")
	}

	// bad billingDetail country
	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID:     "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:      "USD",
		BillingDetail: &BillingDetail{Country: "USA"},
	}); err == nil {
		t.Error("expected error for 3-letter country code")
	}

	// Valid full path — should succeed (covers happy branches).
	positive := 600
	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID:        "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:         "USD",
		PriceSnapshot:    &PriceInfo{Amount: "9.99", TaxCategory: TaxCategoryDigitalGoods},
		BillingDetail:    &BillingDetail{Country: "US", IsBusiness: false},
		ExpiresInSeconds: &positive,
	}); err != nil {
		t.Errorf("full-valid path: %v", err)
	}
}

// TestWebhookPublicKeys_IsZero covers the zero-value helper.
func TestWebhookPublicKeys_IsZero(t *testing.T) {
	if !(WebhookPublicKeys{}).IsZero() {
		t.Error("empty WebhookPublicKeys should be zero")
	}
	if (WebhookPublicKeys{Shared: "x"}).IsZero() {
		t.Error("non-empty WebhookPublicKeys should not be zero")
	}
	if (WebhookPublicKeys{Test: "x"}).IsZero() {
		t.Error("non-empty WebhookPublicKeys should not be zero")
	}
	if (WebhookPublicKeys{Prod: "x"}).IsZero() {
		t.Error("non-empty WebhookPublicKeys should not be zero")
	}
}

// TestDecodeEnvelope_EmptyAndNullBody covers the early-return branches.
func TestDecodeEnvelope_EmptyAndNullBody(t *testing.T) {
	// Empty body, status 500 → returns *Error{Status: 500} with no Errors.
	err := decodeEnvelope(500, []byte(""), nil)
	if err == nil {
		t.Fatal("expected error for 500 with empty body")
	}
	var perr *Error
	if !errors.As(err, &perr) || perr.Status != 500 {
		t.Errorf("unexpected error: %+v", err)
	}

	// Empty body, status 200 → returns nil.
	if err := decodeEnvelope(200, []byte(""), nil); err != nil {
		t.Errorf("empty 200 body should succeed: %v", err)
	}

	// data: null → no unmarshal into out, no error.
	var out struct{ X string }
	if err := decodeEnvelope(200, []byte(`{"data":null}`), &out); err != nil {
		t.Errorf("null data: %v", err)
	}
}
