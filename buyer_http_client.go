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

func (c *buyerHTTPClient) post(ctx context.Context, path string, body any, out any) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

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
