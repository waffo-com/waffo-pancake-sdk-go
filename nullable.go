package pancake

import "encoding/json"

// Nullable expresses a tri-state JSON field: absent, explicit null, or a
// concrete value. Use it for fields that distinguish "leave unchanged" from
// "clear to null" — for example UpdateStoreParams.Logo.
//
//	// Absent (do not send the field)
//	var x pancake.Nullable[string]
//
//	// Explicit null (send "logo": null to clear)
//	x := pancake.ExplicitNull[string]()
//
//	// Concrete value
//	x := pancake.NullValue("https://example.com/logo.png")
type Nullable[T any] struct {
	Value T
	// Valid reports whether the field is present on the wire. When false the
	// field is omitted entirely (works with `json:",omitempty"` on the
	// containing struct).
	Valid bool
	// Null, when Valid is true, emits the literal JSON null.
	Null bool
}

// MarshalJSON emits null when Null is set, the wrapped value when not, and
// returns the JSON "null" token when the wrapper itself is invalid (omitempty
// at the parent struct will keep the field off the wire).
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	if n.Null {
		return []byte("null"), nil
	}
	return json.Marshal(n.Value)
}

// UnmarshalJSON parses null into Null=true and any other value into Value.
func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	n.Valid = true
	if string(data) == "null" {
		n.Null = true
		var zero T
		n.Value = zero
		return nil
	}
	n.Null = false
	return json.Unmarshal(data, &n.Value)
}

// IsZero reports whether the Nullable is unset; used by encoding/json when the
// field is tagged with `omitempty` (Go 1.24+ via IsZero) and by direct callers.
func (n Nullable[T]) IsZero() bool {
	return !n.Valid
}

// Ptr returns a pointer to v. Useful for optional struct fields modeled as
// *T:
//
//	UpdateStoreParams{Name: pancake.Ptr("New name")}
func Ptr[T any](v T) *T {
	return &v
}

// NullValue wraps v in a Nullable that emits the value on the wire.
func NullValue[T any](v T) Nullable[T] {
	return Nullable[T]{Value: v, Valid: true}
}

// ExplicitNull returns a Nullable that emits literal JSON null on the wire —
// the way to clear a server-side field.
func ExplicitNull[T any]() Nullable[T] {
	return Nullable[T]{Valid: true, Null: true}
}

// NullValuePtr is a convenience for `*Nullable[T]` request fields. Returns
// a pointer to a Nullable carrying v.
//
//	UpdateStoreParams{Logo: pancake.NullValuePtr("https://example.com/logo.png")}
func NullValuePtr[T any](v T) *Nullable[T] {
	n := NullValue(v)
	return &n
}

// ExplicitNullPtr is a convenience for `*Nullable[T]` request fields.
// Returns a pointer to a Nullable that emits literal JSON null.
//
//	UpdateStoreParams{Logo: pancake.ExplicitNullPtr[string]()}
func ExplicitNullPtr[T any]() *Nullable[T] {
	n := ExplicitNull[T]()
	return &n
}
