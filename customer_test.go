package pancake

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

// customerTestServer accepts Bearer-authenticated requests and records each one.
type customerTestServer struct {
	srv     *httptest.Server
	mu      sync.Mutex
	reqs    []recordedRequest
	respond func(req recordedRequest) (status int, body any)
}

func newCustomerTestServer() *customerTestServer {
	s := &customerTestServer{
		respond: func(_ recordedRequest) (int, any) {
			return 200, map[string]any{"data": map[string]any{}}
		},
	}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		rec := recordedRequest{
			Method:  req.Method,
			Path:    req.URL.Path,
			Headers: req.Header.Clone(),
			Body:    body,
		}
		s.mu.Lock()
		s.reqs = append(s.reqs, rec)
		s.mu.Unlock()

		status, payload := s.respond(rec)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
	}))
	return s
}

func (s *customerTestServer) requests() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]recordedRequest, len(s.reqs))
	copy(out, s.reqs)
	return out
}

func newCustomerTestClient(t *testing.T) (*Client, *CustomerSession, *customerTestServer) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))

	srv := newCustomerTestServer()
	t.Cleanup(func() { srv.srv.Close() })

	c, err := New(Config{
		MerchantID: "MER_AbCdEfGhIjKlMnOpQrStUv",
		PrivateKey: privPEM,
		BaseURL:    srv.srv.URL,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	customer := c.Customer("JWT_CUSTOMER_TOKEN")
	return c, customer, srv
}

func TestCustomer_AttachesBearer(t *testing.T) {
	_, customer, srv := newCustomerTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"orderId": "ORD_AbCdEfGhIjKlMnOpQrStUv", "status": "canceled"}}
	}

	if _, err := customer.CancelOnetimeOrder(context.Background(), CancelOnetimeOrderParams{
		OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv",
	}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	reqs := srv.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 req, got %d", len(reqs))
	}
	if got, want := reqs[0].Headers.Get("Authorization"), "Bearer JWT_CUSTOMER_TOKEN"; got != want {
		t.Errorf("Authorization header = %q, want %q", got, want)
	}
}

func TestCustomer_Methods_HappyPaths(t *testing.T) {
	ctx := context.Background()
	_, customer, srv := newCustomerTestClient(t)

	cases := []struct {
		name     string
		path     string
		response map[string]any
		call     func() error
	}{
		{
			"CancelSubscription",
			"/v1/actions/subscription-order/cancel-order",
			map[string]any{"orderId": "ORD_AbCdEfGhIjKlMnOpQrStUv", "status": "canceling"},
			func() error {
				_, err := customer.CancelSubscription(ctx, CancelSubscriptionParams{
					OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv",
				})
				return err
			},
		},
		{
			"CancelOnetimeOrder",
			"/v1/actions/onetime-order/cancel-order",
			map[string]any{"orderId": "ORD_AbCdEfGhIjKlMnOpQrStUv", "status": "canceled"},
			func() error {
				_, err := customer.CancelOnetimeOrder(ctx, CancelOnetimeOrderParams{
					OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv",
				})
				return err
			},
		},
		{
			"ReactivateSubscription",
			"/v1/actions/subscription-order/reactivate-order",
			map[string]any{"orderId": "ORD_AbCdEfGhIjKlMnOpQrStUv", "status": "active"},
			func() error {
				_, err := customer.ReactivateSubscription(ctx, ReactivateSubscriptionParams{
					OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv",
				})
				return err
			},
		},
		{
			"CreateRefundTicket",
			"/v1/actions/refund-ticket/create-ticket",
			map[string]any{"ticket": map[string]any{"id": "TKT_AbCdEfGhIjKlMnOpQrStUv", "type": "refund", "status": "pending", "subjectId": "PAY_x", "submitterId": "x", "submitterType": "customer", "metadata": map[string]any{}, "createdAt": "z", "updatedAt": "z"}},
			func() error {
				_, err := customer.CreateRefundTicket(ctx, CreateRefundTicketParams{
					PaymentID:       "PAY_AbCdEfGhIjKlMnOpQrStUv",
					Reason:          "did not work",
					RequestedAmount: RequestedAmount{Amount: "29.00", Currency: "USD"},
				})
				return err
			},
		},
		{
			"ResubmitRefundTicket",
			"/v1/actions/refund-ticket/resubmit-ticket",
			map[string]any{"ticket": map[string]any{"id": "TKT_AbCdEfGhIjKlMnOpQrStUv", "type": "refund", "status": "under_review", "subjectId": "PAY_x", "submitterId": "x", "submitterType": "customer", "metadata": map[string]any{}, "createdAt": "z", "updatedAt": "z"}},
			func() error {
				_, err := customer.ResubmitRefundTicket(ctx, ResubmitRefundTicketParams{
					TicketID:        "TKT_AbCdEfGhIjKlMnOpQrStUv",
					PaymentID:       "PAY_AbCdEfGhIjKlMnOpQrStUv",
					Reason:          "more detail",
					RequestedAmount: RequestedAmount{Amount: "29.00", Currency: "USD"},
				})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv.respond = func(_ recordedRequest) (int, any) {
				return 200, map[string]any{"data": tc.response}
			}
			if err := tc.call(); err != nil {
				t.Fatalf("call: %v", err)
			}
			reqs := srv.requests()
			last := reqs[len(reqs)-1]
			if last.Path != tc.path {
				t.Errorf("path = %q, want %q", last.Path, tc.path)
			}
			if last.Headers.Get("Authorization") == "" {
				t.Error("missing Authorization header")
			}
		})
	}
}

func TestCustomer_ValidationFailures(t *testing.T) {
	ctx := context.Background()
	_, customer, _ := newCustomerTestClient(t)

	cases := []struct {
		name string
		call func() error
	}{
		{"CancelSubscription bad id", func() error {
			_, err := customer.CancelSubscription(ctx, CancelSubscriptionParams{OrderID: "bad"})
			return err
		}},
		{"CancelOnetimeOrder bad id", func() error {
			_, err := customer.CancelOnetimeOrder(ctx, CancelOnetimeOrderParams{OrderID: "bad"})
			return err
		}},
		{"ReactivateSubscription bad id", func() error {
			_, err := customer.ReactivateSubscription(ctx, ReactivateSubscriptionParams{OrderID: "bad"})
			return err
		}},
		{"CreateRefundTicket bad payment id", func() error {
			_, err := customer.CreateRefundTicket(ctx, CreateRefundTicketParams{
				PaymentID:       "bad",
				Reason:          "r",
				RequestedAmount: RequestedAmount{Amount: "1", Currency: "USD"},
			})
			return err
		}},
		{"CreateRefundTicket bad currency", func() error {
			_, err := customer.CreateRefundTicket(ctx, CreateRefundTicketParams{
				PaymentID:       "PAY_AbCdEfGhIjKlMnOpQrStUv",
				Reason:          "r",
				RequestedAmount: RequestedAmount{Amount: "1", Currency: "usd"},
			})
			return err
		}},
		{"ResubmitRefundTicket bad ticket id", func() error {
			_, err := customer.ResubmitRefundTicket(ctx, ResubmitRefundTicketParams{
				TicketID:        "bad",
				PaymentID:       "PAY_AbCdEfGhIjKlMnOpQrStUv",
				Reason:          "r",
				RequestedAmount: RequestedAmount{Amount: "1", Currency: "USD"},
			})
			return err
		}},
		{"GraphQL.Query empty", func() error {
			_, err := customer.GraphQL.Query(ctx, GraphQLParams{})
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
				t.Fatalf("error type %T", err)
			}
			if pe.Errors[0].Layer != ErrorLayerSDK {
				t.Errorf("expected sdk layer, got %q", pe.Errors[0].Layer)
			}
		})
	}
}

// TestCustomerGraphQLQuery_Typed covers the top-level generic helper.
//
// Wire is the standard single-wrap GraphQL envelope.
func TestCustomerGraphQLQuery_Typed(t *testing.T) {
	_, customer, srv := newCustomerTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{
			"data": map[string]any{"orders": []map[string]string{{"id": "ORD_a"}}},
		}
	}
	type R struct {
		Orders []struct {
			ID string `json:"id"`
		} `json:"orders"`
	}
	resp, err := CustomerGraphQLQuery[R](context.Background(), customer, GraphQLParams{Query: "{ orders { id } }"})
	if err != nil {
		t.Fatalf("typed: %v", err)
	}
	if len(resp.Data.Orders) != 1 || resp.Data.Orders[0].ID != "ORD_a" {
		t.Fatalf("typed data not deserialized: %+v", resp.Data)
	}
}

// TestDeprecatedBuyerAliases ensures the deprecated buyer-named API surface
// still compiles and works: Client.Buyer, the BuyerSession /
// BuyerGraphQLResource type aliases, and the BuyerGraphQLQuery wrapper.
func TestDeprecatedBuyerAliases(t *testing.T) {
	c, _, srv := newCustomerTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"orderId": "ORD_AbCdEfGhIjKlMnOpQrStUv", "status": "canceled"}}
	}

	session := c.Buyer("JWT_CUSTOMER_TOKEN")
	var _ *BuyerSession = session
	var _ *BuyerGraphQLResource = session.GraphQL

	res, err := session.CancelOnetimeOrder(context.Background(), CancelOnetimeOrderParams{
		OrderID: "ORD_AbCdEfGhIjKlMnOpQrStUv",
	})
	if err != nil {
		t.Fatalf("cancel via deprecated Buyer session: %v", err)
	}
	if res.Status != "canceled" {
		t.Errorf("status = %q, want %q", res.Status, "canceled")
	}
	reqs := srv.requests()
	if got, want := reqs[len(reqs)-1].Headers.Get("Authorization"), "Bearer JWT_CUSTOMER_TOKEN"; got != want {
		t.Errorf("Authorization header = %q, want %q", got, want)
	}

	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{
			"data": map[string]any{"orders": []map[string]string{{"id": "ORD_a"}}},
		}
	}
	type R struct {
		Orders []struct {
			ID string `json:"id"`
		} `json:"orders"`
	}
	resp, err := BuyerGraphQLQuery[R](context.Background(), session, GraphQLParams{Query: "{ orders { id } }"})
	if err != nil {
		t.Fatalf("deprecated BuyerGraphQLQuery: %v", err)
	}
	if len(resp.Data.Orders) != 1 || resp.Data.Orders[0].ID != "ORD_a" {
		t.Fatalf("typed data not deserialized: %+v", resp.Data)
	}
}

// TestWebhooks_VerifyInstanceMethod ensures client.Webhooks.Verify forwards
// config-level public keys into the standalone verifier.
func TestWebhooks_VerifyInstanceMethod(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	pubDer, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDer}))

	// Sign a payload with the same key, and inject the public key via Config.
	body, _ := json.Marshal(map[string]any{
		"id": "evt", "eventType": "x", "eventId": "x", "storeId": "x", "storeName": "x", "mode": "prod", "timestamp": "z", "data": map[string]any{},
	})
	ts := time.Now().UnixMilli()
	tsStr := strconv.FormatInt(ts, 10)
	hash := sha256.Sum256([]byte(tsStr + "." + string(body)))
	sigBytes, _ := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hash[:])
	sigHeader := fmt.Sprintf("t=%s,v1=%s", tsStr, base64.StdEncoding.EncodeToString(sigBytes))

	client, err := New(Config{
		MerchantID:       "MER_AbCdEfGhIjKlMnOpQrStUv",
		PrivateKey:       privPEM,
		WebhookPublicKey: WebhookPublicKeys{Shared: pubPEM},
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	event, err := client.Webhooks.Verify(string(body), sigHeader, nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if event.ID != "evt" {
		t.Errorf("event.ID = %q", event.ID)
	}

	// Per-call override beats config-level key.
	if _, err := client.Webhooks.Verify(string(body), sigHeader, &VerifyWebhookOptions{PublicKey: pubPEM}); err != nil {
		t.Errorf("per-call override failed: %v", err)
	}
}
