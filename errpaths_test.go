package pancake

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
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

	// GraphQL returns the envelope on non-2xx status — caller inspects resp.Errors.
	// (Single-wrap GraphQL envelope is preserved across all status codes; the SDK
	// does not throw on errors[].)
	t.Run("GraphQL.Query (500) returns envelope", func(t *testing.T) {
		client, _, server := newSignedTestClient(t)
		server.respond = stubGraphQL
		resp, err := client.GraphQL.Query(ctx, GraphQLParams{Query: "{ x }"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Errors) != 1 || resp.Errors[0].Message != "internal" {
			t.Errorf("unexpected envelope errors: %+v", resp.Errors)
		}
	})

	// And generic helper GraphQLQuery[T] surfaces the same envelope.
	t.Run("GraphQLQuery[T] surfaces envelope errors", func(t *testing.T) {
		client, _, server := newSignedTestClient(t)
		server.respond = stubGraphQL
		resp, err := GraphQLQuery[map[string]any](ctx, client, GraphQLParams{Query: "{ x }"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Errors) != 1 || resp.Errors[0].Message != "internal" {
			t.Errorf("unexpected envelope errors: %+v", resp.Errors)
		}
	})
}

func TestCustomer_ErrorPropagation(t *testing.T) {
	ctx := context.Background()
	_, customer, srv := newCustomerTestClient(t)
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
			_, err := customer.CancelSubscription(ctx, CancelSubscriptionParams{OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv"})
			return err
		}},
		{"CancelOnetimeOrder", func() error {
			_, err := customer.CancelOnetimeOrder(ctx, CancelOnetimeOrderParams{OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv"})
			return err
		}},
		{"ReactivateSubscription", func() error {
			_, err := customer.ReactivateSubscription(ctx, ReactivateSubscriptionParams{OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv"})
			return err
		}},
		{"CreateRefundTicket", func() error {
			_, err := customer.CreateRefundTicket(ctx, CreateRefundTicketParams{
				PaymentID: "PAY_AbCdEfGhIjKlMnOpQrStUv", Reason: "r",
				RequestedAmount: RequestedAmount{Amount: "1", Currency: "USD"},
			})
			return err
		}},
		{"ResubmitRefundTicket", func() error {
			_, err := customer.ResubmitRefundTicket(ctx, ResubmitRefundTicketParams{
				TicketID: "TKT_AbCdEfGhIjKlMnOpQrStUv", PaymentID: "PAY_AbCdEfGhIjKlMnOpQrStUv",
				Reason: "r", RequestedAmount: RequestedAmount{Amount: "1", Currency: "USD"},
			})
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

	// empty paymentMethods slice
	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []PaymentMethod{},
	}); err == nil {
		t.Error("expected error for empty paymentMethods")
	}

	// unknown paymentMethods entry
	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []PaymentMethod{"BITCOIN"},
	}); err == nil {
		t.Error("expected error for unknown paymentMethods entry")
	}

	// duplicate paymentMethods entries
	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []PaymentMethod{PaymentMethodCreditCard, PaymentMethodCreditCard},
	}); err == nil {
		t.Error("expected error for duplicate paymentMethods")
	}
}

// TestCheckoutCommon_PaymentMethodsSerialization verifies the ordered
// paymentMethods whitelist is forwarded on the wire, and omitted entirely
// when not set.
func TestCheckoutCommon_PaymentMethodsSerialization(t *testing.T) {
	ctx := context.Background()
	client, _, srv := newSignedTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"sessionId": "x", "checkoutUrl": "https://x", "expiresAt": "z"}}
	}

	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID:      "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:       "USD",
		PaymentMethods: []PaymentMethod{PaymentMethodEWallet, PaymentMethodCreditCard},
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	reqs := srv.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	var body map[string]any
	if err := json.Unmarshal(reqs[0].Body, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	methods, ok := body["paymentMethods"].([]any)
	if !ok || len(methods) != 2 || methods[0] != "EWALLET" || methods[1] != "CREDITCARD" {
		t.Errorf("expected ordered paymentMethods [EWALLET, CREDITCARD], got %v", body["paymentMethods"])
	}

	// Omitted paymentMethods should not appear in the wire body at all.
	if _, err := client.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{
		ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:  "USD",
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	reqs = srv.requests()
	var secondBody map[string]any
	if err := json.Unmarshal(reqs[1].Body, &secondBody); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if _, present := secondBody["paymentMethods"]; present {
		t.Errorf("paymentMethods should be omitted when not set, got %v", secondBody["paymentMethods"])
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

// TestPost_EmptyBodyOn5xxReturnsError covers the early-return branch inside
// httpClient.post: status >= 400 with empty body surfaces *Error{Status}.
func TestPost_EmptyBodyOn5xxReturnsError(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{
		MerchantID: "MER_AbCdEfGhIjKlMnOpQrStUv",
		PrivateKey: privPEM,
		BaseURL:    srv.URL,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = c.Stores.Create(context.Background(), CreateStoreParams{Name: "x"})
	if err == nil {
		t.Fatal("expected error for empty 500 body")
	}
	var perr *Error
	if !errors.As(err, &perr) || perr.Status != 500 {
		t.Errorf("unexpected error: %+v", err)
	}
}
