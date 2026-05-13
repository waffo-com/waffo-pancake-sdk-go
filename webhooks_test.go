package pancake

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func generateRSAKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal pkix: %v", err)
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	return key, pubPEM
}

// makeSignedPayload signs body+timestamp with priv and returns the payload
// JSON string plus the X-Waffo-Signature header value.
func makeSignedPayload(t *testing.T, priv *rsa.PrivateKey, eventBody map[string]any, tsMillis int64) (payload, sigHeader string) {
	t.Helper()
	body, err := json.Marshal(eventBody)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	ts := strconv.FormatInt(tsMillis, 10)
	signatureInput := ts + "." + string(body)

	hash := sha256.Sum256([]byte(signatureInput))
	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return string(body), fmt.Sprintf("t=%s,v1=%s", ts, base64.StdEncoding.EncodeToString(sigBytes))
}

func TestVerifyWebhook_AcceptsValidSignature(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)
	now := time.Now().UnixMilli()
	payload, sig := makeSignedPayload(t, priv, map[string]any{
		"id":        "evt_1",
		"timestamp": "2026-05-13T00:00:00Z",
		"eventType": "order.completed",
		"eventId":   "PAY_AbCdEfGhIjKlMnOpQrStUv",
		"storeId":   "STO_AbCdEfGhIjKlMnOpQrStUv",
		"storeName": "X",
		"mode":      "prod",
		"data":      map[string]any{"orderId": "ORD_x"},
	}, now)

	evt, err := VerifyWebhook(payload, sig, &VerifyWebhookOptions{PublicKey: pubPEM})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if evt.ID != "evt_1" || evt.EventType != "order.completed" {
		t.Errorf("unexpected event: %+v", evt)
	}
}

func TestVerifyWebhook_RejectsBadSignature(t *testing.T) {
	_, pubPEM := generateRSAKeyPair(t)
	other, _ := generateRSAKeyPair(t)
	payload, sig := makeSignedPayload(t, other, map[string]any{"id": "evt"}, time.Now().UnixMilli())
	if _, err := VerifyWebhook(payload, sig, &VerifyWebhookOptions{PublicKey: pubPEM}); err == nil {
		t.Fatal("expected signature mismatch error")
	}
}

func TestVerifyWebhook_RejectsStaleTimestamp(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)
	stale := time.Now().Add(-1 * time.Hour).UnixMilli()
	payload, sig := makeSignedPayload(t, priv, map[string]any{"id": "evt"}, stale)
	if _, err := VerifyWebhook(payload, sig, &VerifyWebhookOptions{PublicKey: pubPEM}); err == nil {
		t.Fatal("expected stale timestamp error")
	}
}

func TestVerifyWebhook_ToleranceDisabled(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)
	stale := time.Now().Add(-1 * time.Hour).UnixMilli()
	payload, sig := makeSignedPayload(t, priv, map[string]any{"id": "evt"}, stale)
	if _, err := VerifyWebhook(payload, sig, &VerifyWebhookOptions{
		PublicKey:   pubPEM,
		ToleranceMS: -1,
	}); err != nil {
		t.Fatalf("verify with tolerance disabled: %v", err)
	}
}

func TestVerifyWebhook_MissingHeader(t *testing.T) {
	if _, err := VerifyWebhook(`{}`, "", nil); err == nil {
		t.Fatal("expected missing header error")
	}
}

func TestVerifyWebhook_MalformedHeader(t *testing.T) {
	if _, err := VerifyWebhook(`{}`, "garbage", nil); err == nil {
		t.Fatal("expected malformed header error")
	}
}

func TestVerifyWebhookTyped_DeserializesData(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)
	now := time.Now().UnixMilli()
	payload, sig := makeSignedPayload(t, priv, map[string]any{
		"id":        "evt_1",
		"timestamp": "2026-05-13",
		"eventType": "order.completed",
		"eventId":   "PAY_x",
		"storeId":   "STO_x",
		"storeName": "X",
		"mode":      "prod",
		"data":      map[string]any{"orderId": "ORD_x", "amount": "29.00", "currency": "USD", "buyerEmail": "a@b.c", "productName": "P", "taxAmount": "0"},
	}, now)

	evt, err := VerifyWebhookTyped[WebhookEventData](payload, sig, &VerifyWebhookOptions{PublicKey: pubPEM})
	if err != nil {
		t.Fatalf("typed: %v", err)
	}
	if evt.Data.OrderID != "ORD_x" || evt.Data.Amount != "29.00" {
		t.Errorf("typed data not populated: %+v", evt.Data)
	}
}

func TestParseSignatureHeader(t *testing.T) {
	t1, v1, err := parseSignatureHeader("t=1700000,v1=abc")
	if err != nil || t1 != "1700000" || v1 != "abc" {
		t.Fatalf("parseSignatureHeader: t=%q v1=%q err=%v", t1, v1, err)
	}
	if _, _, err := parseSignatureHeader("garbage"); err == nil {
		t.Fatal("expected malformed error")
	}
}

func TestVerifyWebhook_ExplicitTestEnv(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)
	now := time.Now().UnixMilli()
	payload, sig := makeSignedPayload(t, priv, map[string]any{
		"id":        "evt",
		"eventType": "x",
		"mode":      "prod",
	}, now)

	if _, err := VerifyWebhook(payload, sig, &VerifyWebhookOptions{
		PublicKeys:  &WebhookPublicKeys{Test: pubPEM},
		Environment: EnvironmentTest,
	}); err != nil {
		t.Fatalf("verify with explicit test env: %v", err)
	}

	// With no config keys for prod and a non-matching built-in prod key,
	// verification must fail.
	if _, err := VerifyWebhook(payload, sig, &VerifyWebhookOptions{Environment: EnvironmentProd}); err == nil {
		t.Fatal("expected prod env mismatch failure")
	}
}
