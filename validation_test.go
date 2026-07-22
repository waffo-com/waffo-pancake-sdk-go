package pancake

import (
	"strings"
	"testing"
)

func TestValidateShortID(t *testing.T) {
	cases := []struct {
		field, value, prefix string
		wantErr              bool
	}{
		{"id", "STO_AbCdEfGhIjKlMnOpQrStUv", "STO", false},
		{"id", "STO_short", "STO", true},
		{"id", "PROD_AbCdEfGhIjKlMnOpQrStUv", "STO", true},
		{"id", "", "STO", true},
	}
	for _, c := range cases {
		err := validateShortID(c.field, c.value, c.prefix)
		if (err != nil) != c.wantErr {
			t.Errorf("validateShortID(%q, %q, %q) err=%v wantErr=%v", c.field, c.value, c.prefix, err, c.wantErr)
		}
	}
}

func TestValidateCurrencyCode(t *testing.T) {
	if err := validateCurrencyCode("currency", "USD"); err != nil {
		t.Errorf("USD: %v", err)
	}
	if err := validateCurrencyCode("currency", "usd"); err == nil {
		t.Errorf("usd should fail")
	}
	if err := validateCurrencyCode("currency", "US"); err == nil {
		t.Errorf("two letters should fail")
	}
}

func TestValidateAmountString(t *testing.T) {
	good := []string{"0", "1", "9.99", "1000", "1000.123"}
	bad := []string{"", "-1", "1.", ".5", "1,000", "abc"}
	for _, v := range good {
		if err := validateAmountString("amount", v); err != nil {
			t.Errorf("good amount %q: %v", v, err)
		}
	}
	for _, v := range bad {
		if err := validateAmountString("amount", v); err == nil {
			t.Errorf("bad amount %q should fail", v)
		}
	}
}

func TestValidateCountryCode(t *testing.T) {
	if err := validateCountryCode("country", "US"); err != nil {
		t.Errorf("US: %v", err)
	}
	if err := validateCountryCode("country", "USA"); err == nil {
		t.Errorf("USA should fail")
	}
}

func TestValidateMaxLength(t *testing.T) {
	// nil pointer (optional field omitted) — accept
	if err := validateMaxLength("orderMerchantExternalId", nil, 128); err != nil {
		t.Errorf("nil should pass: %v", err)
	}
	// boundary — accept
	boundary := strings.Repeat("x", 128)
	if err := validateMaxLength("orderMerchantExternalId", &boundary, 128); err != nil {
		t.Errorf("128 chars should pass: %v", err)
	}
	// over boundary — reject
	over := strings.Repeat("x", 129)
	if err := validateMaxLength("orderMerchantExternalId", &over, 128); err == nil {
		t.Errorf("129 chars should fail")
	}
}

func TestValidatePaymentMethods(t *testing.T) {
	if err := validatePaymentMethods("paymentMethods", []PaymentMethod{PaymentMethodEWallet, PaymentMethodCreditCard}); err != nil {
		t.Errorf("valid ordered methods: %v", err)
	}
	if err := validatePaymentMethods("paymentMethods", []PaymentMethod{}); err == nil {
		t.Errorf("empty slice should fail")
	}
	if err := validatePaymentMethods("paymentMethods", []PaymentMethod{"BITCOIN"}); err == nil {
		t.Errorf("unknown method should fail")
	}
	if err := validatePaymentMethods("paymentMethods", []PaymentMethod{PaymentMethodCreditCard, PaymentMethodCreditCard}); err == nil {
		t.Errorf("duplicate methods should fail")
	}
}

func TestValidatePrices(t *testing.T) {
	good := Prices{"USD": {Amount: "9.99", TaxCategory: TaxCategoryDigitalGoods}}
	if err := validatePrices("prices", good); err != nil {
		t.Errorf("good prices: %v", err)
	}
	empty := Prices{}
	if err := validatePrices("prices", empty); err == nil {
		t.Errorf("empty prices should fail")
	}
	badCurrency := Prices{"usd": {Amount: "9.99", TaxCategory: TaxCategoryDigitalGoods}}
	if err := validatePrices("prices", badCurrency); err == nil {
		t.Errorf("lowercase currency should fail")
	}
	badAmount := Prices{"USD": {Amount: "abc", TaxCategory: TaxCategoryDigitalGoods}}
	if err := validatePrices("prices", badAmount); err == nil {
		t.Errorf("bad amount should fail")
	}
}
