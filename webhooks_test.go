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
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
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

func TestVerifyWebhook_AcceptsTimestampFromRetrySchedule(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)
	// Last retry of a 31-minute schedule still carries the original timestamp.
	retryTS := time.Now().Add(-31 * time.Minute).UnixMilli()
	payload, sig := makeSignedPayload(t, priv, map[string]any{"id": "evt"}, retryTS)
	if _, err := VerifyWebhook(payload, sig, &VerifyWebhookOptions{PublicKey: pubPEM}); err != nil {
		t.Fatalf("verify within retry schedule: %v", err)
	}
}

func TestVerifyWebhook_FutureTolerance(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)

	t.Run("rejects a timestamp too far ahead", func(t *testing.T) {
		future := time.Now().Add(5 * time.Minute).UnixMilli()
		payload, sig := makeSignedPayload(t, priv, map[string]any{"id": "evt"}, future)
		if _, err := VerifyWebhook(payload, sig, &VerifyWebhookOptions{PublicKey: pubPEM}); err == nil {
			t.Fatal("expected future timestamp error")
		}
	})

	t.Run("tolerates small clock skew", func(t *testing.T) {
		skewed := time.Now().Add(30 * time.Second).UnixMilli()
		payload, sig := makeSignedPayload(t, priv, map[string]any{"id": "evt"}, skewed)
		if _, err := VerifyWebhook(payload, sig, &VerifyWebhookOptions{PublicKey: pubPEM}); err != nil {
			t.Fatalf("verify with 30s skew: %v", err)
		}
	})

	t.Run("honors a custom window", func(t *testing.T) {
		future := time.Now().Add(5 * time.Minute).UnixMilli()
		payload, sig := makeSignedPayload(t, priv, map[string]any{"id": "evt"}, future)
		if _, err := VerifyWebhook(payload, sig, &VerifyWebhookOptions{
			PublicKey:         pubPEM,
			FutureToleranceMS: 10 * 60 * 1000,
		}); err != nil {
			t.Fatalf("verify with widened future window: %v", err)
		}
	})
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

func TestVerifyWebhookTyped_PaymentMethodBrand(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)
	for _, brand := range []string{"wechatpay", "alipay", "alipayhk", "visa", "future_wallet", ""} {
		t.Run(brand, func(t *testing.T) {
			data := map[string]any{"orderId": "ORD_brand", "paymentId": "PAY_brand"}
			if brand != "" {
				data["paymentMethodBrand"] = brand
			}
			payload, sig := makeSignedPayload(t, priv, map[string]any{
				"id": "PAY_brand", "eventType": "order.completed", "data": data,
			}, time.Now().UnixMilli())

			evt, err := VerifyWebhookTyped[WebhookEventData](payload, sig, &VerifyWebhookOptions{PublicKey: pubPEM})
			if err != nil {
				t.Fatalf("typed: %v", err)
			}
			if evt.Data.PaymentMethod != nil {
				t.Fatal("paymentMethod should remain absent")
			}
			if brand == "" {
				if evt.Data.PaymentMethodBrand != nil {
					t.Fatal("paymentMethodBrand should be nil when omitted")
				}
				out, err := json.Marshal(evt.Data)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				if strings.Contains(string(out), `"paymentMethodBrand"`) {
					t.Fatalf("paymentMethodBrand should be omitted: %s", out)
				}
			} else if evt.Data.PaymentMethodBrand == nil || *evt.Data.PaymentMethodBrand != brand {
				t.Errorf("paymentMethodBrand = %v, want %s", evt.Data.PaymentMethodBrand, brand)
			}
		})
	}
}

func TestVerifyWebhookTyped_PaymentTimestamp(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)
	for _, ts := range []string{"2026-03-10T23:29:58.123Z", ""} {
		t.Run(ts, func(t *testing.T) {
			data := map[string]any{"orderId": "ORD_ts"}
			if ts != "" {
				data["paymentId"] = "PAY_ts"
				data["paymentDate"] = ts[:10]
				data["paymentTimestamp"] = ts
			}
			payload, sig := makeSignedPayload(t, priv, map[string]any{
				"id": "PAY_ts", "eventType": "order.completed", "data": data,
			}, time.Now().UnixMilli())

			evt, err := VerifyWebhookTyped[WebhookEventData](payload, sig, &VerifyWebhookOptions{PublicKey: pubPEM})
			if err != nil {
				t.Fatalf("typed: %v", err)
			}
			if ts == "" {
				if evt.Data.PaymentTimestamp != nil || evt.Data.PaymentDate != nil {
					t.Fatal("paymentTimestamp and paymentDate should be nil when omitted")
				}
				out, err := json.Marshal(evt.Data)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				if strings.Contains(string(out), `"paymentTimestamp"`) {
					t.Fatalf("paymentTimestamp should be omitted: %s", out)
				}
				return
			}
			if evt.Data.PaymentTimestamp == nil || *evt.Data.PaymentTimestamp != ts {
				t.Errorf("paymentTimestamp = %v, want %s", evt.Data.PaymentTimestamp, ts)
			}
			if evt.Data.PaymentDate == nil || *evt.Data.PaymentDate != ts[:10] {
				t.Errorf("paymentDate = %v, want %s", evt.Data.PaymentDate, ts[:10])
			}
		})
	}
}

func TestWebhookAmountBreakdown_DiscountFields(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)
	payload, sig := makeSignedPayload(t, priv, map[string]any{
		"id":        "evt_disc",
		"eventType": "order.completed",
		"data": map[string]any{
			"orderId": "ORD_x", "amount": "86.40", "currency": "USD", "buyerEmail": "a@b.c", "productName": "P", "taxAmount": "6.40",
			"listPrice": map[string]any{
				"total": "86.40", "subtotal": "99.99", "taxAmount": "6.40", "taxRate": 8, "taxName": "JCT",
				"listTotal": "107.99", "discountAmount": "21.59",
			},
		},
	}, time.Now().UnixMilli())

	evt, err := VerifyWebhookTyped[WebhookEventData](payload, sig, &VerifyWebhookOptions{PublicKey: pubPEM})
	if err != nil {
		t.Fatalf("typed: %v", err)
	}
	lp := evt.Data.ListPrice
	if lp == nil {
		t.Fatal("ListPrice is nil")
	}
	if lp.ListTotal == nil || *lp.ListTotal != "107.99" {
		t.Errorf("ListTotal = %v, want 107.99", lp.ListTotal)
	}
	if lp.DiscountAmount == nil || *lp.DiscountAmount != "21.59" {
		t.Errorf("DiscountAmount = %v, want 21.59", lp.DiscountAmount)
	}
	if lp.Total != "86.40" {
		t.Errorf("Total = %q, want 86.40", lp.Total)
	}
}

func TestWebhookAmountBreakdown_DiscountFieldsAbsent(t *testing.T) {
	var data WebhookEventData
	raw := `{"orderId":"ORD_x","amount":"3.12","currency":"USD","buyerEmail":"a@b.c","productName":"P","taxAmount":"0",` +
		`"originalPayment":{"total":"4.08","subtotal":"3.20","taxAmount":"0.88"}}`
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	op := data.OriginalPayment
	if op == nil {
		t.Fatal("OriginalPayment is nil")
	}
	if op.ListTotal != nil || op.DiscountAmount != nil {
		t.Errorf("discount fields should be nil on a legacy block: ListTotal=%v DiscountAmount=%v", op.ListTotal, op.DiscountAmount)
	}

	out, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(out); got != `{"total":"4.08","subtotal":"3.20","taxAmount":"0.88"}` {
		t.Errorf("marshal = %s, want discount keys omitted", got)
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

// TestWebhookEventType_MatchesPlatform reads enums.go so that adding or dropping
// a WebhookEventType constant without updating this list fails the test.
func TestWebhookEventType_MatchesPlatform(t *testing.T) {
	want := []string{
		"order.completed",
		"subscription.activated",
		"subscription.payment_succeeded",
		"subscription.renewed",
		"subscription.recovered",
		"subscription.plan_changed",
		"subscription.plan_change_scheduled",
		"subscription.plan_change_failed",
		"subscription.canceling",
		"subscription.uncanceled",
		"subscription.canceled",
		"subscription.past_due",
		"refund.succeeded",
		"refund.failed",
		"chargeback.filed",
		"chargeback.updated",
		"store.domain_changed",
		"checkout.completed",
		"checkout.expired",
		"store.domain_unbound",
		"dispute.created",
		"dispute.updated",
		"dispute.message_received",
	}

	file, err := parser.ParseFile(token.NewFileSet(), "enums.go", nil, 0)
	if err != nil {
		t.Fatalf("parse enums.go: %v", err)
	}
	var got []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			ident, ok := vs.Type.(*ast.Ident)
			if !ok || ident.Name != "WebhookEventType" {
				continue
			}
			for _, v := range vs.Values {
				lit, ok := v.(*ast.BasicLit)
				if !ok {
					t.Fatalf("non-literal WebhookEventType value: %#v", v)
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", lit.Value, err)
				}
				got = append(got, s)
			}
		}
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("WebhookEventType constants = %v\nwant %v", got, want)
	}
}

func TestVerifyWebhookTyped_DisputeEvents(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)
	base := map[string]any{
		"disputeId":    "DSP_7n2KqLs",
		"source":       "wechat",
		"status":       "in_progress",
		"escalated":    false,
		"category":     nil,
		"reason":       "Item not received",
		"description":  "I paid but never got access.",
		"respondBy":    "2026-11-28T08:00:00.000Z",
		"paymentId":    "PAY_5xK9mRt",
		"orderId":      "ORD_3hBfDe",
		"buyerEmail":   "buyer@example.com",
		"amount":       "19.99",
		"currency":     "CNY",
		"pspDisputeId": "200000020251125000001",
	}
	withExtra := func(extra map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range base {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}

	cases := []struct {
		eventType WebhookEventType
		dispute   map[string]any
	}{
		{WebhookEventTypeDisputeCreated, base},
		{WebhookEventTypeDisputeUpdated, withExtra(map[string]any{"previousStatus": "open", "escalated": true})},
		{WebhookEventTypeDisputeMessageReceived, withExtra(map[string]any{"message": map[string]any{
			"messageId": "DMSG_9pQr", "authorType": "buyer", "body": "Any update?", "attachmentCount": 2, "createdAt": "2026-11-25T09:00:00.000Z",
		}})},
	}
	for _, tc := range cases {
		t.Run(string(tc.eventType), func(t *testing.T) {
			payload, sig := makeSignedPayload(t, priv, map[string]any{
				"id": "DSP_7n2KqLs", "timestamp": "2026-11-25T09:00:01.000Z", "eventType": string(tc.eventType),
				"eventId": "DSP_7n2KqLs-1", "storeId": "STO_x", "storeName": "X", "mode": "test",
				"data": map[string]any{"dispute": tc.dispute},
			}, time.Now().UnixMilli())

			evt, err := VerifyWebhookTyped[WebhookDisputeEventData](payload, sig, &VerifyWebhookOptions{PublicKey: pubPEM})
			if err != nil {
				t.Fatalf("verify: %v", err)
			}
			d := evt.Data.Dispute
			if evt.EventType != string(tc.eventType) || evt.ID != "DSP_7n2KqLs" || evt.Mode != EnvironmentTest {
				t.Fatalf("unexpected envelope: %+v", evt)
			}
			if d.DisputeID != "DSP_7n2KqLs" || d.Source != "wechat" || d.PaymentID != "PAY_5xK9mRt" || d.Description == "" {
				t.Fatalf("unexpected dispute: %+v", d)
			}
			if d.Category != nil || d.Reason == nil || *d.Reason != "Item not received" || d.Amount == nil || *d.Amount != "19.99" {
				t.Fatalf("nullable fields not decoded: %+v", d)
			}
			if d.PSPDisputeID == nil || *d.PSPDisputeID != "200000020251125000001" {
				t.Fatalf("pspDisputeId = %v", d.PSPDisputeID)
			}

			switch tc.eventType {
			case WebhookEventTypeDisputeUpdated:
				if d.PreviousStatus == nil || *d.PreviousStatus != "open" || !d.Escalated {
					t.Fatalf("updated fields: previousStatus=%v escalated=%v", d.PreviousStatus, d.Escalated)
				}
			case WebhookEventTypeDisputeMessageReceived:
				if d.Message == nil || d.Message.MessageID != "DMSG_9pQr" || d.Message.AuthorType != "buyer" || d.Message.AttachmentCount != 2 {
					t.Fatalf("message = %+v", d.Message)
				}
			default:
				if d.PreviousStatus != nil || d.Message != nil || d.Escalated {
					t.Fatalf("created should carry no previousStatus / message: %+v", d)
				}
			}
		})
	}

	// A tampered body must not verify.
	payload, sig := makeSignedPayload(t, priv, map[string]any{"eventType": "dispute.created", "data": map[string]any{"dispute": base}}, time.Now().UnixMilli())
	if _, err := VerifyWebhookTyped[WebhookDisputeEventData](strings.Replace(payload, "19.99", "0.01", 1), sig, &VerifyWebhookOptions{PublicKey: pubPEM}); err == nil {
		t.Fatal("tampered dispute payload verified")
	}
}

func TestVerifyWebhookTyped_ChargebackStoreAndCheckoutData(t *testing.T) {
	priv, pubPEM := generateRSAKeyPair(t)
	opts := &VerifyWebhookOptions{PublicKey: pubPEM}
	sign := func(body map[string]any) (string, string) {
		return makeSignedPayload(t, priv, body, time.Now().UnixMilli())
	}

	p, s := sign(map[string]any{"eventType": "chargeback.updated", "data": map[string]any{
		"orderId": "ORD_1",
		"chargeback": map[string]any{
			"chargebackId": "CHB_1", "pspChargebackId": "psp-1", "lifeCycle": "FIRST_CYCLE", "status": "WON",
			"previousStatus": "UNDER_REVIEW", "disputedAmount": "10.00", "currency": "USD",
			"chargebackFee": nil, "feeCurrency": nil, "deadline": "2026-12-01T00:00:00.000Z",
		},
	}})
	cb, err := VerifyWebhookTyped[WebhookEventData](p, s, opts)
	if err != nil {
		t.Fatalf("chargeback: %v", err)
	}
	if c := cb.Data.Chargeback; c == nil || c.PreviousStatus == nil || *c.PreviousStatus != "UNDER_REVIEW" || c.ChargebackFee != nil || *c.DisputedAmount != "10.00" {
		t.Fatalf("chargeback = %+v", cb.Data.Chargeback)
	}

	p, s = sign(map[string]any{"eventType": "store.domain_changed", "data": map[string]any{
		"storeId": "STO_1", "storeName": "S", "oldDomain": nil, "newDomain": "shop.example.com", "effectiveAt": "2026-11-25T00:00:00.000Z",
	}})
	changed, err := VerifyWebhookTyped[WebhookStoreDomainChangedData](p, s, opts)
	if err != nil {
		t.Fatalf("domain_changed: %v", err)
	}
	if changed.Data.OldDomain != nil || changed.Data.NewDomain == nil || *changed.Data.NewDomain != "shop.example.com" {
		t.Fatalf("domain_changed = %+v", changed.Data)
	}

	p, s = sign(map[string]any{"eventType": "store.domain_unbound", "data": map[string]any{
		"storeId": "STO_1", "domain": "shop.example.com", "unboundAt": "2026-11-25T00:00:00.000Z", "method": "admin",
	}})
	unbound, err := VerifyWebhookTyped[WebhookStoreDomainUnboundData](p, s, opts)
	if err != nil {
		t.Fatalf("domain_unbound: %v", err)
	}
	if unbound.Data.Method == nil || *unbound.Data.Method != "admin" {
		t.Fatalf("domain_unbound = %+v", unbound.Data)
	}

	p, s = sign(map[string]any{"id": "cs_1", "eventType": "checkout.expired", "data": map[string]any{
		"sessionId": "cs_1", "status": "expired", "orderId": nil,
	}})
	expired, err := VerifyWebhookTyped[WebhookCheckoutSessionData](p, s, opts)
	if err != nil {
		t.Fatalf("checkout.expired: %v", err)
	}
	if expired.Data.SessionID != "cs_1" || expired.Data.Status != "expired" || expired.Data.OrderID != nil {
		t.Fatalf("checkout.expired = %+v", expired.Data)
	}
}
