package pancake

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func freshTestPrivateKeyPEM(t *testing.T) string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

// TestResources_OptionalFieldValidation exercises the optional-field validation
// branches in Update methods: when a caller provides Name / BillingPeriod /
// Prices, the per-field validator runs and can reject. The standard happy-path
// and error-propagation suites pass only required fields, leaving these
// branches unreached.
func TestResources_OptionalFieldValidation(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		call func(c *Client) error
	}{
		{"OnetimeProducts.Update empty name rejected", func(c *Client) error {
			_, err := c.OnetimeProducts.Update(ctx, UpdateOnetimeProductParams{
				ID:   "PROD_AbCdEfGhIjKlMnOpQrStUv",
				Name: Ptr(""),
			})
			return err
		}},
		{"OnetimeProducts.Update bad prices rejected", func(c *Client) error {
			_, err := c.OnetimeProducts.Update(ctx, UpdateOnetimeProductParams{
				ID:     "PROD_AbCdEfGhIjKlMnOpQrStUv",
				Prices: Prices{"USD": {Amount: "not-a-number", TaxCategory: TaxCategoryDigitalGoods}},
			})
			return err
		}},
		{"SubscriptionProducts.Update empty name rejected", func(c *Client) error {
			_, err := c.SubscriptionProducts.Update(ctx, UpdateSubscriptionProductParams{
				ID:   "PROD_AbCdEfGhIjKlMnOpQrStUv",
				Name: Ptr(""),
			})
			return err
		}},
		{"SubscriptionProducts.Update bad billing period rejected", func(c *Client) error {
			bp := BillingPeriod("yearly-ish")
			_, err := c.SubscriptionProducts.Update(ctx, UpdateSubscriptionProductParams{
				ID:            "PROD_AbCdEfGhIjKlMnOpQrStUv",
				BillingPeriod: &bp,
			})
			return err
		}},
		{"SubscriptionProducts.Update bad prices rejected", func(c *Client) error {
			_, err := c.SubscriptionProducts.Update(ctx, UpdateSubscriptionProductParams{
				ID:     "PROD_AbCdEfGhIjKlMnOpQrStUv",
				Prices: Prices{"USD": {Amount: "not-a-number", TaxCategory: TaxCategorySaaS}},
			})
			return err
		}},
		{"StoreMerchants.Add invalid role rejected", func(c *Client) error {
			_, err := c.StoreMerchants.Add(ctx, AddMerchantParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv",
				Email:   "x@y.z",
				Role:    "owner", // not in {admin, member}
			})
			return err
		}},
		{"StoreMerchants.UpdateRole invalid role rejected", func(c *Client) error {
			_, err := c.StoreMerchants.UpdateRole(ctx, UpdateRoleParams{
				StoreID:    "STO_AbCdEfGhIjKlMnOpQrStUv",
				MerchantID: "MER_AbCdEfGhIjKlMnOpQrStUv",
				Role:       "owner",
			})
			return err
		}},
		{"Stores.Update missing id rejected", func(c *Client) error {
			_, err := c.Stores.Update(ctx, UpdateStoreParams{ID: ""})
			return err
		}},
		{"Webhooks.Add missing url rejected", func(c *Client) error {
			_, err := c.Webhooks.Add(ctx, AddWebhookParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv",
				Channel: WebhookChannelHTTP,
				URL:     "",
				Events:  []WebhookEventType{WebhookEventTypeOrderCompleted},
			})
			return err
		}},
		{"Webhooks.Add missing channel rejected", func(c *Client) error {
			_, err := c.Webhooks.Add(ctx, AddWebhookParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv",
				URL:     "https://x",
				Events:  []WebhookEventType{WebhookEventTypeOrderCompleted},
			})
			return err
		}},
		{"Auth.IssueSessionToken bad productId rejected", func(c *Client) error {
			_, err := c.Auth.IssueSessionToken(ctx, IssueSessionTokenParams{
				ProductID:     Ptr("bad"),
				BuyerIdentity: "x@y.z",
			})
			return err
		}},
		{"Auth.IssueSessionToken missing buyerIdentity rejected", func(c *Client) error {
			_, err := c.Auth.IssueSessionToken(ctx, IssueSessionTokenParams{
				StoreID:       Ptr("STO_AbCdEfGhIjKlMnOpQrStUv"),
				BuyerIdentity: "",
			})
			return err
		}},
		{"OnetimeProducts.UpdateStatus invalid status rejected", func(c *Client) error {
			_, err := c.OnetimeProducts.UpdateStatus(ctx, UpdateOnetimeStatusParams{
				ID:     "PROD_AbCdEfGhIjKlMnOpQrStUv",
				Status: ProductVersionStatus("retired"),
			})
			return err
		}},
		{"SubscriptionProducts.UpdateStatus invalid status rejected", func(c *Client) error {
			_, err := c.SubscriptionProducts.UpdateStatus(ctx, UpdateSubscriptionStatusParams{
				ID:     "PROD_AbCdEfGhIjKlMnOpQrStUv",
				Status: ProductVersionStatus("retired"),
			})
			return err
		}},
		// SubscriptionProducts.Create — exercise each remaining validation branch
		{"SubscriptionProducts.Create empty name rejected", func(c *Client) error {
			_, err := c.SubscriptionProducts.Create(ctx, CreateSubscriptionProductParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Name: "", BillingPeriod: BillingPeriodMonthly,
				Prices: Prices{"USD": {Amount: "1", TaxCategory: TaxCategorySaaS}},
			})
			return err
		}},
		{"SubscriptionProducts.Create empty prices rejected", func(c *Client) error {
			_, err := c.SubscriptionProducts.Create(ctx, CreateSubscriptionProductParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Name: "X", BillingPeriod: BillingPeriodMonthly,
			})
			return err
		}},
		// OnetimeProducts.Create — empty name branch
		{"OnetimeProducts.Create empty name rejected", func(c *Client) error {
			_, err := c.OnetimeProducts.Create(ctx, CreateOnetimeProductParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Name: "",
				Prices: Prices{"USD": {Amount: "1", TaxCategory: TaxCategoryDigitalGoods}},
			})
			return err
		}},
		// StoreMerchants.Add — missing email branch
		{"StoreMerchants.Add empty email rejected", func(c *Client) error {
			_, err := c.StoreMerchants.Add(ctx, AddMerchantParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Email: "", Role: "admin",
			})
			return err
		}},
		// SubscriptionProductGroups.Create — empty name branch
		{"SubscriptionProductGroups.Create empty name rejected", func(c *Client) error {
			_, err := c.SubscriptionProductGroups.Create(ctx, CreateSubscriptionProductGroupParams{
				StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv", Name: "",
			})
			return err
		}},
		// Customer ResubmitRefundTicket — exercise the 4 unreached validation branches
		{"Customer.ResubmitRefundTicket bad ticketId rejected", func(c *Client) error {
			customer := c.Customer("tok")
			_, err := customer.ResubmitRefundTicket(ctx, ResubmitRefundTicketParams{
				TicketID: "bad", PaymentID: "PAY_AbCdEfGhIjKlMnOpQrStUv",
				Reason: "r", RequestedAmount: RequestedAmount{Amount: "1", Currency: "USD"},
			})
			return err
		}},
		{"Customer.ResubmitRefundTicket bad paymentId rejected", func(c *Client) error {
			customer := c.Customer("tok")
			_, err := customer.ResubmitRefundTicket(ctx, ResubmitRefundTicketParams{
				TicketID: "TKT_AbCdEfGhIjKlMnOpQrStUv", PaymentID: "bad",
				Reason: "r", RequestedAmount: RequestedAmount{Amount: "1", Currency: "USD"},
			})
			return err
		}},
		{"Customer.ResubmitRefundTicket empty reason rejected", func(c *Client) error {
			customer := c.Customer("tok")
			_, err := customer.ResubmitRefundTicket(ctx, ResubmitRefundTicketParams{
				TicketID: "TKT_AbCdEfGhIjKlMnOpQrStUv", PaymentID: "PAY_AbCdEfGhIjKlMnOpQrStUv",
				Reason: "", RequestedAmount: RequestedAmount{Amount: "1", Currency: "USD"},
			})
			return err
		}},
		{"Customer.ResubmitRefundTicket bad amount rejected", func(c *Client) error {
			customer := c.Customer("tok")
			_, err := customer.ResubmitRefundTicket(ctx, ResubmitRefundTicketParams{
				TicketID: "TKT_AbCdEfGhIjKlMnOpQrStUv", PaymentID: "PAY_AbCdEfGhIjKlMnOpQrStUv",
				Reason: "r", RequestedAmount: RequestedAmount{Amount: "not-a-number", Currency: "USD"},
			})
			return err
		}},
		{"Customer.CreateRefundTicket empty reason rejected", func(c *Client) error {
			customer := c.Customer("tok")
			_, err := customer.CreateRefundTicket(ctx, CreateRefundTicketParams{
				PaymentID: "PAY_AbCdEfGhIjKlMnOpQrStUv", Reason: "",
				RequestedAmount: RequestedAmount{Amount: "1", Currency: "USD"},
			})
			return err
		}},
		{"Customer.CreateRefundTicket bad amount rejected", func(c *Client) error {
			customer := c.Customer("tok")
			_, err := customer.CreateRefundTicket(ctx, CreateRefundTicketParams{
				PaymentID: "PAY_AbCdEfGhIjKlMnOpQrStUv", Reason: "r",
				RequestedAmount: RequestedAmount{Amount: "not-a-number", Currency: "USD"},
			})
			return err
		}},
		{"Customer.CreateRefundTicket bad currency rejected", func(c *Client) error {
			customer := c.Customer("tok")
			_, err := customer.CreateRefundTicket(ctx, CreateRefundTicketParams{
				PaymentID: "PAY_AbCdEfGhIjKlMnOpQrStUv", Reason: "r",
				RequestedAmount: RequestedAmount{Amount: "1", Currency: "INVALID"},
			})
			return err
		}},
		{"Checkout.CreateSession missing productId rejected", func(c *Client) error {
			_, err := c.Checkout.CreateSession(ctx, CreateCheckoutSessionParams{Currency: "USD"})
			return err
		}},
		{"Checkout.Anonymous.Create missing productId rejected", func(c *Client) error {
			_, err := c.Checkout.Anonymous.Create(ctx, AnonymousCheckoutParams{Currency: "USD"})
			return err
		}},
		{"Checkout.Authenticated.Create missing productId rejected", func(c *Client) error {
			_, err := c.Checkout.Authenticated.Create(ctx, AuthenticatedCheckoutParams{
				CreateCheckoutSessionParams: CreateCheckoutSessionParams{Currency: "USD"},
				BuyerIdentity:               "x",
			})
			return err
		}},
		{"GraphQL.Query bad query rejected", func(c *Client) error {
			_, err := c.GraphQL.Query(ctx, GraphQLParams{Query: ""})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _, _ := newSignedTestClient(t)
			err := tc.call(client)
			if err == nil {
				t.Fatal("expected validation error")
			}
			var perr *Error
			if !errors.As(err, &perr) {
				t.Fatalf("error type %T, want *Error", err)
			}
			if perr.Status != 400 || perr.Errors[0].Layer != ErrorLayerSDK {
				t.Errorf("expected sdk-layer 400 error, got status=%d layer=%q", perr.Status, perr.Errors[0].Layer)
			}
		})
	}
}

// TestCustomerHTTP_EmptyBodyOn4xxReturnsError covers the empty-body 4xx branch in
// customerHTTPClient.post.
func TestCustomerHTTP_EmptyBodyOn4xxReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{
		MerchantID:  "MER_AbCdEfGhIjKlMnOpQrStUv",
		PrivateKey:  freshTestPrivateKeyPEM(t),
		BaseURL:     srv.URL,
		Environment: EnvironmentTest,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	customer := c.Customer("token")
	_, err = customer.CancelSubscription(context.Background(), CancelSubscriptionParams{OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv"})
	if err == nil {
		t.Fatal("expected error on empty 401 body")
	}
	var perr *Error
	if !errors.As(err, &perr) || perr.Status != 401 {
		t.Errorf("unexpected error: %+v", err)
	}
}

// TestCustomerHTTP_EmptyBodyOn200ReturnsEmptyEnvelope covers the empty-body 200
// branch (rare but valid: server returns no body on success).
func TestCustomerHTTP_EmptyBodyOn200ReturnsEmptyEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{
		MerchantID:  "MER_AbCdEfGhIjKlMnOpQrStUv",
		PrivateKey:  freshTestPrivateKeyPEM(t),
		BaseURL:     srv.URL,
		Environment: EnvironmentTest,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	customer := c.Customer("token")
	// Empty body yields an empty Result struct; no error.
	res, err := customer.CancelSubscription(context.Background(), CancelSubscriptionParams{OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OrderID != "" {
		t.Errorf("expected zero-valued result, got %+v", res)
	}
}

// TestCustomerHTTP_NonJSONBodyReturnsError covers the non-JSON parse branch.
func TestCustomerHTTP_NonJSONBodyReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(502)
		_, _ = w.Write([]byte("<html>bad gateway</html>"))
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{
		MerchantID:  "MER_AbCdEfGhIjKlMnOpQrStUv",
		PrivateKey:  freshTestPrivateKeyPEM(t),
		BaseURL:     srv.URL,
		Environment: EnvironmentTest,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	customer := c.Customer("token")
	_, err = customer.CancelSubscription(context.Background(), CancelSubscriptionParams{OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv"})
	if err == nil {
		t.Fatal("expected error on non-JSON body")
	}
	var perr *Error
	if !errors.As(err, &perr) || perr.Errors[0].Layer != ErrorLayerSDK {
		t.Errorf("unexpected error: %+v", err)
	}
}

// TestGraphQLQuery_TypedUnmarshalsData verifies the typed helper successfully
// decodes envelope data into the caller's struct, covering the previously
// unreached unmarshal path.
func TestGraphQLQuery_TypedUnmarshalsData(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{
			"data": map[string]any{
				"stores": []map[string]any{{"id": "STO_a", "name": "Acme"}},
			},
		}
	}
	type R struct {
		Stores []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"stores"`
	}
	resp, err := GraphQLQuery[R](context.Background(), client, GraphQLParams{Query: "{ stores { id name } }"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data.Stores) != 1 || resp.Data.Stores[0].Name != "Acme" {
		t.Errorf("typed unmarshal failed: %+v", resp.Data)
	}
}

// TestGraphQLQuery_TypedNullDataReturnsZero covers the `data: null` branch of
// typedFromRaw — typed value stays zero, no error.
func TestGraphQLQuery_TypedNullDataReturnsZero(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": nil}
	}
	type R struct{ X string }
	resp, err := GraphQLQuery[R](context.Background(), client, GraphQLParams{Query: "{ x }"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Data.X != "" {
		t.Errorf("expected zero value, got %+v", resp.Data)
	}
}

// TestResources_HappyPathWithWarnings ensures that resource methods surface
// envelope warnings onto the returned Result's Warnings field.
func TestResources_HappyPathWithWarnings(t *testing.T) {
	ctx := context.Background()
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{
			"data":     map[string]any{"store": map[string]any{"id": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "X", "status": "active"}},
			"warnings": []map[string]any{{"message": "deprecated field", "layer": "store", "aiHint": "Switch to client.webhooks.*"}},
		}
	}

	res, err := client.Stores.Update(ctx, UpdateStoreParams{ID: "STO_AbCdEfGhIjKlMnOpQrStUv"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(res.Warnings))
	}
	if res.Warnings[0].AIHint == "" {
		t.Error("expected aiHint to be propagated")
	}
	if res.Warnings[0].Layer != ErrorLayerStore {
		t.Errorf("expected layer=store, got %q", res.Warnings[0].Layer)
	}
}
