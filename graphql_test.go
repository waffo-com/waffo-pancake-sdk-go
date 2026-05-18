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

// TestGraphQL_WireShapes covers every response shape /v1/graphql can produce.
// The wire is always the standard single-wrap GraphQL envelope
// {data, errors?, warnings?}; the SDK must surface it verbatim (no
// envelope-stripping, no throw on errors[]).
func TestGraphQL_WireShapes(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name   string
		status int
		body   map[string]any
		assert func(t *testing.T, resp *GraphQLResponse)
	}{
		{
			// Shape #1: standard success
			name:   "success returns data verbatim",
			status: 200,
			body: map[string]any{
				"data": map[string]any{
					"stores": []map[string]any{{"id": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "Acme"}},
				},
			},
			assert: func(t *testing.T, resp *GraphQLResponse) {
				var data struct {
					Stores []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"stores"`
				}
				if err := json.Unmarshal(resp.Data, &data); err != nil {
					t.Fatalf("unmarshal data: %v", err)
				}
				if len(data.Stores) != 1 || data.Stores[0].ID != "STO_AbCdEfGhIjKlMnOpQrStUv" {
					t.Errorf("unexpected data: %+v", data)
				}
				if len(resp.Errors) != 0 {
					t.Errorf("expected no errors, got %+v", resp.Errors)
				}
			},
		},
		{
			// Shape #2: success + warnings (cost over threshold)
			name:   "warnings surface alongside data",
			status: 200,
			body: map[string]any{
				"data": map[string]any{"stores": []any{}},
				"warnings": []map[string]any{
					{
						"message": "Query estimated cost 25000 exceeds warning threshold 20000",
						"layer":   "graphql",
						"aiHint":  "REDUCE_QUERY_SIZE: halve all list `limit` arguments",
					},
				},
			},
			assert: func(t *testing.T, resp *GraphQLResponse) {
				if len(resp.Warnings) != 1 {
					t.Fatalf("expected 1 warning, got %d", len(resp.Warnings))
				}
				if resp.Warnings[0].AIHint == "" {
					t.Error("expected aiHint to be set")
				}
			},
		},
		{
			// Shape #3: partial success (data + errors co-exist)
			name:   "partial-success envelope preserved",
			status: 200,
			body: map[string]any{
				"data": map[string]any{
					"stores":    []map[string]any{{"id": "STO_AbCdEfGhIjKlMnOpQrStUv"}},
					"analytics": nil,
				},
				"errors": []map[string]any{
					{"message": "analytics resolver failed", "path": []string{"analytics"}},
				},
			},
			assert: func(t *testing.T, resp *GraphQLResponse) {
				if len(resp.Data) == 0 {
					t.Fatal("expected data to be present")
				}
				if len(resp.Errors) != 1 || resp.Errors[0].Message != "analytics resolver failed" {
					t.Errorf("unexpected errors: %+v", resp.Errors)
				}
			},
		},
		{
			// Shape #4: schema validation error (status 200 by GraphQL convention)
			name:   "status-200 schema validation error does not throw",
			status: 200,
			body: map[string]any{
				"data": nil,
				"errors": []map[string]any{
					{"message": "Cannot query field 'nonexistent' on type 'Query'", "layer": "graphql"},
				},
			},
			assert: func(t *testing.T, resp *GraphQLResponse) {
				if len(resp.Errors) != 1 {
					t.Fatalf("expected 1 error, got %d", len(resp.Errors))
				}
			},
		},
		{
			// Shape #5a: gateway error with non-2xx status
			name:   "HTTP 401 returns envelope instead of throwing",
			status: 401,
			body: map[string]any{
				"data": nil,
				"errors": []map[string]any{
					{"message": "Authentication required", "layer": "gateway"},
				},
			},
			assert: func(t *testing.T, resp *GraphQLResponse) {
				if len(resp.Errors) != 1 || resp.Errors[0].Layer != "gateway" {
					t.Errorf("unexpected errors: %+v", resp.Errors)
				}
			},
		},
		{
			// Shape #5b: graphql-service crash with 500
			name:   "HTTP 500 returns envelope instead of throwing",
			status: 500,
			body: map[string]any{
				"data": nil,
				"errors": []map[string]any{
					{"message": "Request processing failed", "layer": "graphql"},
				},
			},
			assert: func(t *testing.T, resp *GraphQLResponse) {
				if len(resp.Errors) != 1 || resp.Errors[0].Layer != "graphql" {
					t.Errorf("unexpected errors: %+v", resp.Errors)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _, server := newSignedTestClient(t)
			server.respond = func(_ recordedRequest) (int, any) {
				return tc.status, tc.body
			}
			resp, err := client.GraphQL.Query(ctx, GraphQLParams{Query: "query { stores { id } }"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.assert(t, resp)
		})
	}
}

// TestGraphQL_NoIdempotencyKey guards against the GraphQL query carrying
// X-Idempotency-Key: the gateway caches idempotent responses for 24h, which
// would serve stale data on subsequent identical queries. Queries must hit
// the live database.
func TestGraphQL_NoIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"stores": []any{}}}
	}
	_, err := client.GraphQL.Query(ctx, GraphQLParams{Query: "{ stores { id } }"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reqs := server.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if got := reqs[0].Headers.Get("X-Idempotency-Key"); got != "" {
		t.Errorf("GraphQL request must not carry X-Idempotency-Key, got %q", got)
	}
	if reqs[0].Headers.Get("X-Signature") == "" {
		t.Error("expected X-Signature header to be present")
	}
}

func TestGraphQL_NonJSONBodyReturnsError(t *testing.T) {
	ctx := context.Background()

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
		w.WriteHeader(502)
		_, _ = w.Write([]byte("<html><body>Bad Gateway</body></html>"))
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
	_, err = c.GraphQL.Query(ctx, GraphQLParams{Query: "{ stores { id } }"})
	if err == nil {
		t.Fatal("expected error for non-JSON body")
	}
	var perr *Error
	if !errors.As(err, &perr) || perr.Status != 502 {
		t.Errorf("unexpected error: %+v", err)
	}
}

// TestBuyerGraphQL_WireShapes mirrors the merchant suite for the Bearer-auth path.
func TestBuyerGraphQL_WireShapes(t *testing.T) {
	ctx := context.Background()

	t.Run("success returns data verbatim", func(t *testing.T) {
		_, buyer, srv := newBuyerTestClient(t)
		srv.respond = func(_ recordedRequest) (int, any) {
			return 200, map[string]any{
				"data": map[string]any{
					"orders": []map[string]any{{"id": "ORD_AbCdEfGhIjKlMnOpQrStUv", "status": "completed"}},
				},
			}
		}
		resp, err := buyer.GraphQL.Query(ctx, GraphQLParams{Query: "{ orders { id status } }"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var data struct {
			Orders []struct{ ID, Status string } `json:"orders"`
		}
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(data.Orders) != 1 || data.Orders[0].ID != "ORD_AbCdEfGhIjKlMnOpQrStUv" {
			t.Errorf("unexpected data: %+v", data)
		}
	})

	t.Run("HTTP 403 returns envelope instead of throwing", func(t *testing.T) {
		_, buyer, srv := newBuyerTestClient(t)
		srv.respond = func(_ recordedRequest) (int, any) {
			return 403, map[string]any{
				"data":   nil,
				"errors": []map[string]any{{"message": "session expired", "layer": "gateway"}},
			}
		}
		resp, err := buyer.GraphQL.Query(ctx, GraphQLParams{Query: "{ orders { id } }"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Errors) != 1 || resp.Errors[0].Message != "session expired" {
			t.Errorf("unexpected errors: %+v", resp.Errors)
		}
	})
}
