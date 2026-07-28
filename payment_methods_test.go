// Payment method marshal contract tests.
//
// CreateCheckoutSessionParams.PaymentMethods limits the methods offered on the
// hosted checkout page. JSON keys use the lowercase Pancake enum values, aligned
// with sdk-ts and the create-session endpoint.
package pancake

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCreateCheckoutSessionParams_IncludePaymentMethods_Marshal(t *testing.T) {
	p := CreateCheckoutSessionParams{
		ProductID:             "PROD_xxx",
		Currency:              "USD",
		IncludePaymentMethods: []PaymentMethod{PaymentMethodCard, PaymentMethodWeChat},
	}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"includePaymentMethods":["card","wechat"]`) {
		t.Fatalf("expected includePaymentMethods in payload, got %s", out)
	}
}

func TestCreateCheckoutSessionParams_ExcludePaymentMethods_Marshal(t *testing.T) {
	p := CreateCheckoutSessionParams{
		ProductID:             "PROD_xxx",
		Currency:              "USD",
		ExcludePaymentMethods: []PaymentMethod{PaymentMethodWeChat},
	}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"excludePaymentMethods":["wechat"]`) {
		t.Fatalf("expected excludePaymentMethods in payload, got %s", out)
	}
}

func TestCreateCheckoutSessionParams_PaymentMethodFilters_Omitted(t *testing.T) {
	p := CreateCheckoutSessionParams{ProductID: "PROD_xxx", Currency: "USD"}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), "PaymentMethods") || strings.Contains(string(out), "paymentMethods") {
		t.Fatalf("expected omitempty to drop both filter fields, got %s", out)
	}
}

func TestPaymentMethodConstants(t *testing.T) {
	cases := map[PaymentMethod]string{
		PaymentMethodCard:      "card",
		PaymentMethodApplePay:  "applepay",
		PaymentMethodGooglePay: "googlepay",
		PaymentMethodWeChat:    "wechat",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Fatalf("expected %q, got %q", want, string(got))
		}
	}
}
