package pancake

import (
	"encoding/json"
	"testing"
)

func TestNullable_MarshalAbsent(t *testing.T) {
	type wrap struct {
		Logo *Nullable[string] `json:"logo,omitempty"`
	}
	b, _ := json.Marshal(wrap{})
	if got, want := string(b), `{}`; got != want {
		t.Fatalf("absent: got %s want %s", got, want)
	}
}

func TestNullable_MarshalValue(t *testing.T) {
	type wrap struct {
		Logo *Nullable[string] `json:"logo,omitempty"`
	}
	v := NullValue("https://example.com/logo.png")
	b, _ := json.Marshal(wrap{Logo: &v})
	if got, want := string(b), `{"logo":"https://example.com/logo.png"}`; got != want {
		t.Fatalf("value: got %s want %s", got, want)
	}
}

func TestNullable_MarshalExplicitNull(t *testing.T) {
	type wrap struct {
		Logo *Nullable[string] `json:"logo,omitempty"`
	}
	v := ExplicitNull[string]()
	b, _ := json.Marshal(wrap{Logo: &v})
	if got, want := string(b), `{"logo":null}`; got != want {
		t.Fatalf("null: got %s want %s", got, want)
	}
}

func TestNullable_UnmarshalNull(t *testing.T) {
	var n Nullable[string]
	if err := json.Unmarshal([]byte(`null`), &n); err != nil {
		t.Fatal(err)
	}
	if !n.Valid || !n.Null {
		t.Fatalf("expected Valid+Null after null, got %+v", n)
	}
}

func TestNullable_UnmarshalValue(t *testing.T) {
	var n Nullable[string]
	if err := json.Unmarshal([]byte(`"hello"`), &n); err != nil {
		t.Fatal(err)
	}
	if !n.Valid || n.Null || n.Value != "hello" {
		t.Fatalf("unexpected nullable state: %+v", n)
	}
}

func TestPtr(t *testing.T) {
	p := Ptr("foo")
	if p == nil || *p != "foo" {
		t.Fatalf("Ptr(foo) returned %v", p)
	}
}
