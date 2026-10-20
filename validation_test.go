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

func TestValidateRequired(t *testing.T) {
	for _, v := range []string{"", "  ", "\t\n"} {
		perr, ok := validateRequired("name", v).(*Error)
		if !ok || perr.Status != 400 || perr.Errors[0].Layer != ErrorLayerSDK {
			t.Errorf("validateRequired(%q) = %v, want SDK 400", v, perr)
		}
	}
	for _, v := range []string{"x", " x "} {
		if err := validateRequired("name", v); err != nil {
			t.Errorf("validateRequired(%q) = %v, want nil", v, err)
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

func TestValidateMaxLength_CountsUTF16CodeUnits(t *testing.T) {
	// 22 CJK characters are 66 UTF-8 bytes but 22 UTF-16 code units — accept
	cjk := strings.Repeat("优", 22)
	if err := validateMaxLength("name", &cjk, 64); err != nil {
		t.Errorf("22 CJK characters should pass: %v", err)
	}
	// 64 CJK characters at the boundary — accept
	cjk64 := strings.Repeat("优", 64)
	if err := validateMaxLength("name", &cjk64, 64); err != nil {
		t.Errorf("64 CJK characters should pass: %v", err)
	}
	// 65 code units — reject
	cjk65 := strings.Repeat("优", 65)
	if err := validateMaxLength("name", &cjk65, 64); err == nil {
		t.Errorf("65 CJK characters should fail")
	}
	// an emoji outside the BMP is 2 code units: 32 emoji = 64 pass, 32 emoji + 1 = 65 fail
	emoji := strings.Repeat("😀", 32)
	if err := validateMaxLength("name", &emoji, 64); err != nil {
		t.Errorf("32 emoji (64 code units) should pass: %v", err)
	}
	emojiOver := emoji + "x"
	if err := validateMaxLength("name", &emojiOver, 64); err == nil {
		t.Errorf("65 code units should fail")
	}
}

func TestValidateCouponName(t *testing.T) {
	good := []string{"Black Friday", strings.Repeat("优", 64), "  " + strings.Repeat("x", 64) + "  "}
	bad := []string{"", "   ", strings.Repeat("x", 65), strings.Repeat("优", 65)}
	for _, v := range good {
		if err := validateCouponName(v); err != nil {
			t.Errorf("good name %q: %v", v, err)
		}
	}
	for _, v := range bad {
		if err := validateCouponName(v); err == nil {
			t.Errorf("bad name %q should fail", v)
		}
	}
}

func TestValidatePositiveAmountString(t *testing.T) {
	good := []string{"0.01", "5.00", "500", "10"}
	bad := []string{"0", "0.00", "00.000", "", "abc", "-1"}
	for _, v := range good {
		if err := validatePositiveAmountString("amount", v); err != nil {
			t.Errorf("good amount %q: %v", v, err)
		}
	}
	for _, v := range bad {
		if err := validatePositiveAmountString("amount", v); err == nil {
			t.Errorf("bad amount %q should fail", v)
		}
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
