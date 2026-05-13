package pancake

import "context"

// SubscriptionProductGroupsResource manages groups of related subscription
// products (shared trial, plan switching).
type SubscriptionProductGroupsResource struct {
	http *httpClient
}

// Create creates a subscription product group.
func (r *SubscriptionProductGroupsResource) Create(ctx context.Context, p CreateSubscriptionProductGroupParams) (*SubscriptionProductGroupResult, error) {
	if err := validateShortID("storeId", p.StoreID, "STO"); err != nil {
		return nil, err
	}
	if err := validateRequired("name", p.Name); err != nil {
		return nil, err
	}
	var out SubscriptionProductGroupResult
	if err := r.http.post(ctx, "/v1/actions/subscription-product-group/create-group", p, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update updates a subscription product group. ProductIDs is a full
// replacement, not a merge.
func (r *SubscriptionProductGroupsResource) Update(ctx context.Context, p UpdateSubscriptionProductGroupParams) (*SubscriptionProductGroupResult, error) {
	if err := validateRequired("id", p.ID); err != nil {
		return nil, err
	}
	var out SubscriptionProductGroupResult
	if err := r.http.post(ctx, "/v1/actions/subscription-product-group/update-group", p, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete hard-deletes a subscription product group.
func (r *SubscriptionProductGroupsResource) Delete(ctx context.Context, p DeleteSubscriptionProductGroupParams) (*SubscriptionProductGroupResult, error) {
	if err := validateRequired("id", p.ID); err != nil {
		return nil, err
	}
	var out SubscriptionProductGroupResult
	if err := r.http.post(ctx, "/v1/actions/subscription-product-group/delete-group", p, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Publish promotes a test-environment group to production (upsert).
func (r *SubscriptionProductGroupsResource) Publish(ctx context.Context, p PublishSubscriptionProductGroupParams) (*SubscriptionProductGroupResult, error) {
	if err := validateRequired("id", p.ID); err != nil {
		return nil, err
	}
	var out SubscriptionProductGroupResult
	if err := r.http.post(ctx, "/v1/actions/subscription-product-group/publish-group", p, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
