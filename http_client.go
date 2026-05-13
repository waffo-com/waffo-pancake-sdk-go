package pancake

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/waffo-com/waffo-pancake-sdk-go/internal/signing"
)

// httpClient is the internal merchant-authenticated transport. It signs every
// POST with RSA-SHA256, attaches a deterministic idempotency key, and unwraps
// the response envelope. Not exported — consumers go through the resource
// methods on Client.
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

// post sends a signed POST and decodes the response envelope's data field
// into out (when out is non-nil). Non-2xx responses or envelopes carrying
// errors are returned as *Error.
func (c *httpClient) post(ctx context.Context, path string, body any, opts *postOptions, out any) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	tsSec := time.Now().Unix()
	timestamp := strconv.FormatInt(tsSec, 10)

	signature, err := signing.SignRequest("POST", path, timestamp, bodyBytes, c.privateKey)
	if err != nil {
		return fmt.Errorf("sign request: %w", err)
	}

	idempotencyInput := c.merchantID + ":" + path + ":" + string(bodyBytes)
	if opts != nil && opts.IdempotencyWindow > 0 {
		window := int64(opts.IdempotencyWindow)
		idempotencyInput = fmt.Sprintf("%s:%d", idempotencyInput, tsSec/window)
	}
	keyHash := sha256.Sum256([]byte(idempotencyInput))
	idempotencyKey := hex.EncodeToString(keyHash[:])

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-Id", c.merchantID)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Idempotency-Key", idempotencyKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	return decodeEnvelope(resp.StatusCode, respBody, out)
}

// decodeEnvelope parses the { data, errors } API envelope, returning *Error
// when the server reports errors and unmarshaling data into out otherwise.
func decodeEnvelope(status int, body []byte, out any) error {
	if len(bytes.TrimSpace(body)) == 0 {
		if status >= 400 {
			return &Error{Status: status}
		}
		return nil
	}

	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []APIError      `json:"errors,omitempty"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("decode response envelope: %w", err)
	}

	if len(env.Errors) > 0 {
		return &Error{Status: status, Errors: env.Errors}
	}

	if out == nil || len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("decode response data: %w", err)
	}
	return nil
}
