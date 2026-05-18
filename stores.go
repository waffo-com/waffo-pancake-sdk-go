package pancake

import "context"

// StoresResource manages stores: create, update, and soft-delete.
type StoresResource struct {
	http *httpClient
}

// Create creates a new store. Slug is generated server-side from Name.
//
// Example:
//
//	res, err := client.Stores.Create(ctx, pancake.CreateStoreParams{Name: "My Store"})
//	fmt.Println(res.Store.ID) // "STO_..."
func (r *StoresResource) Create(ctx context.Context, p CreateStoreParams) (*CreateStoreResult, error) {
	if err := validateRequired("name", p.Name); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[CreateStoreResult](ctx, r.http, "/v1/actions/store/create-store", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Update updates an existing store's settings. Only provided fields are
// changed; NotificationSettings and CheckoutSettings accept partial updates
// with [Nullable] used to distinguish "leave unchanged" from "clear to null".
//
// Example:
//
//	res, err := client.Stores.Update(ctx, pancake.UpdateStoreParams{
//	    ID:           "STO_...",
//	    Logo:         pancake.ExplicitNullPtr[string](),
//	    SupportEmail: pancake.NullValuePtr("help@example.com"),
//	})
func (r *StoresResource) Update(ctx context.Context, p UpdateStoreParams) (*UpdateStoreResult, error) {
	if err := validateShortID("id", p.ID, "STO"); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[UpdateStoreResult](ctx, r.http, "/v1/actions/store/update-store", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}

// Delete soft-deletes a store. Only the store owner can delete.
//
// Example:
//
//	res, err := client.Stores.Delete(ctx, pancake.DeleteStoreParams{ID: "STO_..."})
func (r *StoresResource) Delete(ctx context.Context, p DeleteStoreParams) (*DeleteStoreResult, error) {
	if err := validateShortID("id", p.ID, "STO"); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[DeleteStoreResult](ctx, r.http, "/v1/actions/store/delete-store", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}
