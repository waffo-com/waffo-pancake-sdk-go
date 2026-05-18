package pancake

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// buyerHTTPClient performs Bearer-authenticated POST requests for buyer
// self-service operations. Not exported — consumers go through BuyerSession.
type buyerHTTPClient struct {
	token   string
	baseURL string
	client  *http.Client
}

func newBuyerHTTPClient(token, baseURL string, h *http.Client) *buyerHTTPClient {
	return &buyerHTTPClient{
		token:   token,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  h,
	}
}

// post sends a Bearer-authenticated POST and returns the full envelope plus
// HTTP status. Does not throw on errors[] — caller inspects the envelope.
func (c *buyerHTTPClient) post(ctx context.Context, path string, body any) (int, *envelope, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

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

