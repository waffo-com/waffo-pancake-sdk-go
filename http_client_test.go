package pancake

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/waffo-com/waffo-pancake-sdk-go/internal/signing"
)

func newSignedTestClient(t *testing.T) (*Client, *rsa.PrivateKey, *recordingServer) {
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

	server := newRecordingServer()
	t.Cleanup(server.Close)

	client, err := New(Config{
		MerchantID: "MER_AbCdEfGhIjKlMnOpQrStUv",
		PrivateKey: privPEM,
		BaseURL:    server.URL(),
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client, priv, server
}

type recordingServer struct {
	srv          *httptest.Server
	mu           sync.Mutex
	receivedReqs []recordedRequest
	respond      func(req recordedRequest) (status int, body any)
}

func (r *recordingServer) record(rec recordedRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.receivedReqs = append(r.receivedReqs, rec)
}

func (r *recordingServer) requests() []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]recordedRequest, len(r.receivedReqs))
	copy(out, r.receivedReqs)
	return out
}

type recordedRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
}

func newRecordingServer() *recordingServer {
	r := &recordingServer{
		respond: func(_ recordedRequest) (int, any) {
			return 200, map[string]any{"data": map[string]any{}}
		},
	}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		rec := recordedRequest{
			Method:  req.Method,
			Path:    req.URL.Path,
			Headers: req.Header.Clone(),
			Body:    body,
		}
		r.record(rec)

		status, payload := r.respond(rec)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
	}))
	return r
}

func (r *recordingServer) Close() { r.srv.Close() }
func (r *recordingServer) URL() string {
	return r.srv.URL
}

func TestHTTPClient_SignsRequest(t *testing.T) {
	client, priv, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"store": map[string]string{"id": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "x"}}}
	}

	if _, err := client.Stores.Create(context.Background(), CreateStoreParams{Name: "X"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	reqs := server.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]
	if req.Headers.Get("X-Merchant-Id") != "MER_AbCdEfGhIjKlMnOpQrStUv" {
		t.Errorf("missing or wrong X-Merchant-Id: %q", req.Headers.Get("X-Merchant-Id"))
	}
	if req.Headers.Get("X-Timestamp") == "" || req.Headers.Get("X-Signature") == "" {
		t.Error("missing X-Timestamp or X-Signature")
	}

	// Recompute the canonical signature and verify against the server's pub key.
	canonical := "POST\n/v1/actions/store/create-store\n" + req.Headers.Get("X-Timestamp") + "\n" +
		base64.StdEncoding.EncodeToString(sha256Sum(req.Body))
	if err := signing.VerifySignature(canonical, req.Headers.Get("X-Signature"), &priv.PublicKey); err != nil {
		t.Fatalf("server-side signature verification failed: %v", err)
	}
}

func TestHTTPClient_IdempotencyKey(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"store": map[string]string{"id": "STO_AbCdEfGhIjKlMnOpQrStUv", "name": "x"}}}
	}

	for i := 0; i < 2; i++ {
		if _, err := client.Stores.Create(context.Background(), CreateStoreParams{Name: "X"}); err != nil {
			t.Fatalf("create iter %d: %v", i, err)
		}
	}
	reqs := server.requests()
	if len(reqs) != 2 {
		t.Fatalf("expected 2 reqs, got %d", len(reqs))
	}
	k1 := reqs[0].Headers.Get("X-Idempotency-Key")
	k2 := reqs[1].Headers.Get("X-Idempotency-Key")
	if k1 == "" || k1 != k2 {
		t.Fatalf("expected identical idempotency keys for identical params, got %q vs %q", k1, k2)
	}
	if _, err := hex.DecodeString(k1); err != nil {
		t.Fatalf("idempotency key is not hex: %q (err: %v)", k1, err)
	}

	// Different body -> different key.
	if _, err := client.Stores.Create(context.Background(), CreateStoreParams{Name: "Y"}); err != nil {
		t.Fatalf("create Y: %v", err)
	}
	k3 := server.requests()[2].Headers.Get("X-Idempotency-Key")
	if k3 == k1 {
		t.Fatal("expected different idempotency key for different body")
	}
}

func TestHTTPClient_PropagatesAPIError(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 400, map[string]any{
			"data":   nil,
			"errors": []map[string]string{{"message": "store name conflict", "layer": "store"}},
		}
	}

	_, err := client.Stores.Create(context.Background(), CreateStoreParams{Name: "X"})
	if err == nil {
		t.Fatal("expected error")
	}
	pe, ok := err.(*Error)
	if !ok {
		t.Fatalf("error is %T, want *Error", err)
	}
	if pe.Status != 400 || len(pe.Errors) != 1 || pe.Errors[0].Layer != ErrorLayerStore {
		t.Errorf("unexpected error: %+v", pe)
	}
}

func TestHTTPClient_ClientSideValidation(t *testing.T) {
	client, _, _ := newSignedTestClient(t)
	_, err := client.Stores.Update(context.Background(), UpdateStoreParams{ID: "bad"})
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
}

func TestNew_RejectsBadMerchantID(t *testing.T) {
	_, err := New(Config{MerchantID: "not-a-merchant-id", PrivateKey: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckoutAuthenticated_AppendsTokenFragment(t *testing.T) {
	client, _, server := newSignedTestClient(t)

	server.respond = func(req recordedRequest) (int, any) {
		switch {
		case strings.HasSuffix(req.Path, "/issue-session-token"):
			return 200, map[string]any{"data": map[string]any{"token": "JWT", "expiresAt": "2026-05-13T01:00:00Z"}}
		case strings.HasSuffix(req.Path, "/create-session"):
			return 200, map[string]any{"data": map[string]any{
				"sessionId":   "ses_1",
				"checkoutUrl": "https://pancake.example/checkout/abc",
				"expiresAt":   "2026-05-13T00:45:00Z",
			}}
		}
		return 404, map[string]any{"data": nil, "errors": []map[string]string{{"message": "no", "layer": "gateway"}}}
	}

	res, err := client.Checkout.Authenticated.Create(context.Background(), AuthenticatedCheckoutParams{
		CreateCheckoutSessionParams: CreateCheckoutSessionParams{
			ProductID: "PROD_AbCdEfGhIjKlMnOpQrStUv",
			Currency:  "USD",
		},
		BuyerIdentity: "user-1",
	})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if !strings.HasSuffix(res.CheckoutURL, "#token=JWT") {
		t.Fatalf("CheckoutURL does not include token fragment: %s", res.CheckoutURL)
	}
	if res.Token != "JWT" {
		t.Errorf("Token = %q want JWT", res.Token)
	}
}

func sha256Sum(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}
