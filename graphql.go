package pancake

import (
	"context"
	"encoding/json"
)

// GraphQLResource runs GraphQL queries at the merchant scope (Query only, no
// Mutations).
type GraphQLResource struct {
	http *httpClient
}

// Query executes a GraphQL query. Data is returned as raw JSON bytes; pass
// it through [json.Unmarshal] or use [GraphQLQuery] for static-typed access.
//
// Example:
//
//	resp, err := client.GraphQL.Query(ctx, pancake.GraphQLParams{
//	    Query: `query { stores { id name status } }`,
//	})
//	var data struct {
//	    Stores []pancake.Store `json:"stores"`
//	}
//	_ = json.Unmarshal(resp.Data, &data)
func (r *GraphQLResource) Query(ctx context.Context, p GraphQLParams) (*GraphQLResponse, error) {
	if err := validateRequired("query", p.Query); err != nil {
		return nil, err
	}
	_, env, err := r.http.post(ctx, "/v1/graphql", p, &postOptions{NoIdempotency: true})
	if err != nil {
		return nil, err
	}
	return &GraphQLResponse{Data: env.Data, Errors: env.Errors, Warnings: env.Warnings}, nil
}

// GraphQLQuery executes a merchant-scoped GraphQL query and unmarshals the
// response data into a caller-provided struct type T.
//
// Example:
//
//	type StoresQuery struct {
//	    Stores []pancake.Store `json:"stores"`
//	}
//	resp, err := pancake.GraphQLQuery[StoresQuery](ctx, client, pancake.GraphQLParams{
//	    Query: `query { stores { id name status } }`,
//	})
//	fmt.Println(resp.Data.Stores[0].Name)
func GraphQLQuery[T any](ctx context.Context, c *Client, p GraphQLParams) (*TypedGraphQLResponse[T], error) {
	raw, err := c.GraphQL.Query(ctx, p)
	if err != nil {
		return nil, err
	}
	return typedFromRaw[T](raw)
}

// BuyerGraphQLQuery executes a buyer-scoped GraphQL query and unmarshals the
// response data into a caller-provided struct type T.
//
// Example:
//
//	type OrdersQuery struct {
//	    Orders []struct {
//	        ID     string `json:"id"`
//	        Status string `json:"status"`
//	    } `json:"orders"`
//	}
//	resp, err := pancake.BuyerGraphQLQuery[OrdersQuery](ctx, buyer, pancake.GraphQLParams{
//	    Query: `query { orders { id status } }`,
//	})
func BuyerGraphQLQuery[T any](ctx context.Context, s *BuyerSession, p GraphQLParams) (*TypedGraphQLResponse[T], error) {
	raw, err := s.GraphQL.Query(ctx, p)
	if err != nil {
		return nil, err
	}
	return typedFromRaw[T](raw)
}

func typedFromRaw[T any](raw *GraphQLResponse) (*TypedGraphQLResponse[T], error) {
	var data T
	if len(raw.Data) > 0 && string(raw.Data) != "null" {
		if err := json.Unmarshal(raw.Data, &data); err != nil {
			return nil, err
		}
	}
	return &TypedGraphQLResponse[T]{
		Data:     data,
		Errors:   raw.Errors,
		Warnings: raw.Warnings,
	}, nil
}
