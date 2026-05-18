package pancake

import (
	"context"
	"encoding/json"
	"fmt"
)

// Resource-layer helpers that turn a parsed transport envelope
// (data + errors + warnings + status) into the (*Result, error) shape every
// resource method exposes:
//
//   - Throw *Error when the envelope carries errors[]
//   - Otherwise unmarshal data into out, return warnings separately
//
// Resource methods then attach warnings onto their named Result struct's
// Warnings field, preserving the (*Result, error) signature.
//
// These functions live here (not in http_client.go / buyer_http_client.go)
// because they encode resource-layer policy (when to throw, when to extract
// warnings). The transport files do plain sign + fetch + JSON parse.

// postAction is the resource-layer helper for the signed merchant transport.
func postAction[T any](ctx context.Context, c *httpClient, path string, body any, opts *postOptions) (*T, []Notice, error) {
	status, env, err := c.post(ctx, path, body, opts)
	if err != nil {
		return nil, nil, err
	}
	return unwrapEnvelope[T](status, env)
}

// buyerPostAction is the resource-layer helper for the Bearer buyer transport.
func buyerPostAction[T any](ctx context.Context, c *buyerHTTPClient, path string, body any) (*T, []Notice, error) {
	status, env, err := c.post(ctx, path, body)
	if err != nil {
		return nil, nil, err
	}
	return unwrapEnvelope[T](status, env)
}

func unwrapEnvelope[T any](status int, env *envelope) (*T, []Notice, error) {
	if len(env.Errors) > 0 {
		return nil, nil, &Error{Status: status, Errors: env.Errors}
	}
	var out T
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &out); err != nil {
			return nil, nil, fmt.Errorf("decode response data: %w", err)
		}
	}
	return &out, env.Warnings, nil
}
