package pancake

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/waffo-com/waffo-pancake-sdk-go/internal/signing"
)

// httpClient is the internal merchant-authenticated transport.
//
// One method: post — signs, fetches, parses the {data, errors?, warnings?}
// envelope. Does NOT unwrap data, throw on errors[], or hide warnings — those
// are policy choices that belong to the resource layer (see postAction helper).
//
// Not exported — consumers go through the resource methods on Client.
type httpClient struct {
	merchantID string
	privateKey *rsa.PrivateKey
	baseURL    string
	client     *http.Client
}

func newHTTPClient(merchantID string, key *rsa.PrivateKey, baseURL string, h *http.Client) *httpClient {
	return &httpClient{
		merchantID: merchantID,
		privateKey: key,
		baseURL:    strings.TrimRight(baseURL, "/"),
		client:     h,
	}
}

// post sends a signed POST and returns the full envelope plus HTTP status.
//
// Attaches X-Idempotency-Key only when the caller supplied one via
// WithIdempotencyKey. No key is derived here and none is sent otherwise, so a
// request without one is not covered by the gateway's 24h dedup cache.
//
// Returns *Error only on transport failures (network reject, non-JSON body).
// Never throws on errors[] — the resource layer inspects the envelope.
func (c *httpClient) post(ctx context.Context, path string, body any, opts requestOptions) (int, *envelope, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request body: %w", err)
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature, err := signing.SignRequest("POST", path, timestamp, bodyBytes, c.privateKey)
	if err != nil {
		return 0, nil, fmt.Errorf("sign request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-Id", c.merchantID)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", signature)
	if opts.idempotencyKey != "" {
		req.Header.Set("X-Idempotency-Key", opts.idempotencyKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response body: %w", err)
	}

	if len(bytes.TrimSpace(respBody)) == 0 {
		if resp.StatusCode >= 400 {
			return resp.StatusCode, nil, &Error{Status: resp.StatusCode}
		}
		return resp.StatusCode, &envelope{}, nil
	}

	var env envelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return resp.StatusCode, nil, &Error{
			Status: resp.StatusCode,
			Errors: []Notice{{Message: fmt.Sprintf("Non-JSON response from %s: %v", path, err), Layer: ErrorLayerSDK}},
		}
	}
	return resp.StatusCode, &env, nil
}
