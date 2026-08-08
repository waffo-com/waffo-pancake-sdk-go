package pancake

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/waffo-com/waffo-pancake-sdk-go/internal/signing"
)

// DefaultWebhookToleranceMS is the default past-facing replay-protection window
// applied when [VerifyWebhookOptions.ToleranceMS] is zero.
//
// The signature timestamp is stamped once, before the first delivery attempt —
// retries carry the original header, so by the last attempt the timestamp is as
// old as the whole retry schedule (observed above 31 minutes). A window shorter
// than that rejects legitimate retries. 45 minutes covers the schedule plus
// clock skew on the receiving server.
//
// A window this wide does not make replay attacks cheap on its own: every event
// carries a stable ID, and handlers are expected to be idempotent on it.
const DefaultWebhookToleranceMS = 45 * 60 * 1000

// DefaultWebhookFutureToleranceMS is the default future-facing window, matching
// the gateway's API Key check. Only clock skew puts a timestamp ahead of now, so
// this stays tight.
const DefaultWebhookFutureToleranceMS = 60 * 1000

// builtinTestPublicKey is the test-environment webhook verification key.
// Mirrored byte-for-byte from sdks/waffo-pancake-sdk-ts/src/webhooks.ts.
const builtinTestPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxnmRY6yMMA3lVqmAU6ZG
b1sjL/+r/z6E+ZjkXaDAKiqOhk9rpazni0bNsGXwmftTPk9jy2wn+j6JHODD/WH/
SCnSfvKkLIjy4Hk7BuCgB174C0ydan7J+KgXLkOwgCAxxB68t2tezldwo74ZpXgn
F49opzMvQ9prEwIAWOE+kV9iK6gx/AckSMtHIHpUesoPDkldpmFHlB2qpf1vsFTZ
5kD6DmGl+2GIVK01aChy2lk8pLv0yUMu18v44sLkO5M44TkGPJD9qG09wrvVG2wp
OTVCn1n5pP8P+HRLcgzbUB3OlZVfdFurn6EZwtyL4ZD9kdkQ4EZE/9inKcp3c1h4
xwIDAQAB
-----END PUBLIC KEY-----`

// builtinProdPublicKey is the production-environment webhook verification key.
// Mirrored byte-for-byte from sdks/waffo-pancake-sdk-ts/src/webhooks.ts.
const builtinProdPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAz+xApdTIb4ua+DgZKQ54
iBsD82ybyhGCLRETONW4Jgbb3A8DUM1LqBk6r/CmTOCHqLalTQHNigvP3R5zkDNX
iRJz6gA4MJ/+8K0+mnEE2RISQzN+Qu65TNd6svb+INm/kMaftY4uIXr6y6kchtTJ
dwnQhcKdAL2v7h7IFnkVelQsKxDdb2PqX8xX/qwd01iXvMcpCCaXovUwZsxH2QN5
ZKBTseJivbhUeyJCco4fdUyxOMHe2ybCVhyvim2uxAl1nkvL5L8RCWMCAV55LLo0
9OhmLahz/DYNu13YLVP6dvIT09ZFBYU6Owj1NxdinTynlJCFS9VYwBgmftosSE1U
dwIDAQAB
-----END PUBLIC KEY-----`

// AddWebhookResult wraps the response of Webhooks.Add / Update / Remove.
type AddWebhookResult struct {
	Webhook  StoreWebhook `json:"webhook"`
	Warnings []Notice     `json:"warnings,omitempty"`
}

// UpdateWebhookResult mirrors AddWebhookResult.
type UpdateWebhookResult = AddWebhookResult

// RemoveWebhookResult mirrors AddWebhookResult.
type RemoveWebhookResult = AddWebhookResult

// WebhooksResource manages webhook configuration (Add / Update / Remove) and
// verifies inbound webhook signatures (Verify). Verification is a local
// cryptographic operation and does not perform any I/O.
type WebhooksResource struct {
	http       *httpClient
	publicKeys WebhookPublicKeys
}

// Add registers a webhook endpoint on a store.
//
// Example:
//
//	res, err := client.Webhooks.Add(ctx, pancake.AddWebhookParams{
//	    StoreID:  "STO_...",
//	    Channel:  pancake.WebhookChannelHTTP,
//	    URL:      "https://example.com/webhooks",
//	    Events:   []pancake.WebhookEventType{pancake.WebhookEventTypeOrderCompleted},
//	    TestMode: false,
//	})
func (r *WebhooksResource) Add(ctx context.Context, p AddWebhookParams) (*AddWebhookResult, error) {
	if err := validateShortID("storeId", p.StoreID, "STO"); err != nil {
		return nil, err
	}
	if err := validateRequired("channel", string(p.Channel)); err != nil {
		return nil, err
	}
	if err := validateRequired("url", p.URL); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[AddWebhookResult](ctx, r.http, "/v1/actions/store/add-webhook", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Update updates a webhook's mutable fields (URL, events, secret). Channel
// and TestMode are immutable — remove and re-add to change them.
func (r *WebhooksResource) Update(ctx context.Context, p UpdateWebhookParams) (*UpdateWebhookResult, error) {
	if err := validateRequired("id", p.ID); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[UpdateWebhookResult](ctx, r.http, "/v1/actions/store/update-webhook", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Remove hard-deletes a webhook. Historical delivery records are retained
// with a nulled-out webhook reference for audit purposes.
func (r *WebhooksResource) Remove(ctx context.Context, p RemoveWebhookParams) (*RemoveWebhookResult, error) {
	if err := validateRequired("id", p.ID); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[RemoveWebhookResult](ctx, r.http, "/v1/actions/store/remove-webhook", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Verify validates the X-Waffo-Signature header against payload using the
// configured public key chain. Config-level public keys are merged into the
// resolution chain automatically.
//
// Example:
//
//	event, err := client.Webhooks.Verify(rawBody, r.Header.Get("X-Waffo-Signature"), nil)
func (r *WebhooksResource) Verify(payload, signatureHeader string, opts *VerifyWebhookOptions) (*WebhookEvent, error) {
	merged := mergeVerifyOptions(opts, r.publicKeys)
	return VerifyWebhook(payload, signatureHeader, merged)
}

// VerifyWebhook is the standalone webhook verifier. Use it when the
// surrounding code does not have access to a *Client — for example a small
// webhook-only service.
//
// Example:
//
//	event, err := pancake.VerifyWebhook(string(body), r.Header.Get("X-Waffo-Signature"), nil)
func VerifyWebhook(payload, signatureHeader string, opts *VerifyWebhookOptions) (*WebhookEvent, error) {
	if signatureHeader == "" {
		return nil, errors.New("missing X-Waffo-Signature header")
	}
	t, v1, err := parseSignatureHeader(signatureHeader)
	if err != nil {
		return nil, err
	}

	tolerance := DefaultWebhookToleranceMS
	futureTolerance := DefaultWebhookFutureToleranceMS
	if opts != nil {
		switch {
		case opts.ToleranceMS < 0:
			tolerance = 0
		case opts.ToleranceMS > 0:
			tolerance = int(opts.ToleranceMS)
		}
		if opts.FutureToleranceMS > 0 {
			futureTolerance = int(opts.FutureToleranceMS)
		}
	}
	// Asymmetric, matching the gateway's API Key check.
	if tolerance > 0 {
		ts, perr := strconv.ParseInt(t, 10, 64)
		if perr != nil {
			return nil, errors.New("invalid timestamp in X-Waffo-Signature header")
		}
		ageMS := time.Now().UnixMilli() - ts
		if ageMS > int64(tolerance) || ageMS < -int64(futureTolerance) {
			return nil, errors.New("webhook timestamp outside tolerance window (possible replay attack)")
		}
	}

	signatureInput := t + "." + payload

	if opts != nil && opts.PublicKey != "" {
		key, kerr := normalizeAndParsePublicKey(opts.PublicKey)
		if kerr != nil {
			return nil, kerr
		}
		if verr := signing.VerifySignature(signatureInput, v1, key); verr != nil {
			return nil, errors.New("invalid webhook signature (custom key)")
		}
		return unmarshalEvent(payload)
	}

	var configKeys WebhookPublicKeys
	if opts != nil && opts.PublicKeys != nil {
		configKeys = *opts.PublicKeys
	}

	if opts != nil && opts.Environment != "" {
		key, kerr := resolveKeyForEnv(opts.Environment, configKeys)
		if kerr != nil {
			return nil, kerr
		}
		if verr := signing.VerifySignature(signatureInput, v1, key); verr != nil {
			return nil, fmt.Errorf("invalid webhook signature (%s key)", string(opts.Environment))
		}
		return unmarshalEvent(payload)
	}

	// Auto-detect: try prod first, then test.
	prodKey, kerr := resolveKeyForEnv(EnvironmentProd, configKeys)
	if kerr != nil {
		return nil, kerr
	}
	if verr := signing.VerifySignature(signatureInput, v1, prodKey); verr == nil {
		return unmarshalEvent(payload)
	}
	testKey, kerr := resolveKeyForEnv(EnvironmentTest, configKeys)
	if kerr != nil {
		return nil, kerr
	}
	if verr := signing.VerifySignature(signatureInput, v1, testKey); verr != nil {
		return nil, errors.New("invalid webhook signature (tried both prod and test keys)")
	}
	return unmarshalEvent(payload)
}

// VerifyWebhookTyped is [VerifyWebhook] with the event payload unmarshaled
// into a caller-provided struct type T.
//
// Example:
//
//	event, err := pancake.VerifyWebhookTyped[pancake.WebhookEventData](body, sig, nil)
func VerifyWebhookTyped[T any](payload, signatureHeader string, opts *VerifyWebhookOptions) (*TypedWebhookEvent[T], error) {
	raw, err := VerifyWebhook(payload, signatureHeader, opts)
	if err != nil {
		return nil, err
	}
	var data T
	if len(raw.Data) > 0 && string(raw.Data) != "null" {
		if err := json.Unmarshal(raw.Data, &data); err != nil {
			return nil, fmt.Errorf("decode webhook data: %w", err)
		}
	}
	return &TypedWebhookEvent[T]{
		ID:        raw.ID,
		Timestamp: raw.Timestamp,
		EventType: raw.EventType,
		EventID:   raw.EventID,
		StoreID:   raw.StoreID,
		StoreName: raw.StoreName,
		Mode:      raw.Mode,
		Data:      data,
	}, nil
}

func mergeVerifyOptions(opts *VerifyWebhookOptions, keys WebhookPublicKeys) *VerifyWebhookOptions {
	if opts == nil {
		k := keys
		return &VerifyWebhookOptions{PublicKeys: &k}
	}
	merged := *opts
	if merged.PublicKeys == nil {
		k := keys
		merged.PublicKeys = &k
	}
	return &merged
}

func parseSignatureHeader(h string) (t, v1 string, err error) {
	for _, pair := range strings.Split(h, ",") {
		eq := strings.IndexByte(pair, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(pair[:eq])
		val := strings.TrimSpace(pair[eq+1:])
		switch key {
		case "t":
			t = val
		case "v1":
			v1 = val
		}
	}
	if t == "" || v1 == "" {
		return "", "", errors.New("malformed X-Waffo-Signature header: missing t or v1")
	}
	return t, v1, nil
}

func resolveKeyForEnv(env Environment, configKeys WebhookPublicKeys) (*rsa.PublicKey, error) {
	if configKeys.Shared != "" {
		return normalizeAndParsePublicKey(configKeys.Shared)
	}
	if env == EnvironmentTest && configKeys.Test != "" {
		return normalizeAndParsePublicKey(configKeys.Test)
	}
	if env == EnvironmentProd && configKeys.Prod != "" {
		return normalizeAndParsePublicKey(configKeys.Prod)
	}

	var perEnv string
	if env == EnvironmentTest {
		perEnv = os.Getenv("WAFFO_WEBHOOK_TEST_PUBLIC_KEY")
	} else {
		perEnv = os.Getenv("WAFFO_WEBHOOK_PROD_PUBLIC_KEY")
	}
	if perEnv != "" {
		return normalizeAndParsePublicKey(perEnv)
	}

	if shared := os.Getenv("WAFFO_WEBHOOK_PUBLIC_KEY"); shared != "" {
		return normalizeAndParsePublicKey(shared)
	}

	if env == EnvironmentTest {
		return normalizeAndParsePublicKey(builtinTestPublicKey)
	}
	return normalizeAndParsePublicKey(builtinProdPublicKey)
}

func normalizeAndParsePublicKey(raw string) (*rsa.PublicKey, error) {
	normalized, err := signing.NormalizePublicKey(raw)
	if err != nil {
		return nil, err
	}
	return signing.ParsePublicKey(normalized)
}

func unmarshalEvent(payload string) (*WebhookEvent, error) {
	var ev WebhookEvent
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		return nil, fmt.Errorf("decode webhook event: %w", err)
	}
	return &ev, nil
}
