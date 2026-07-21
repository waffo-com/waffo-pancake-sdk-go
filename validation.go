package pancake

import "regexp"

var (
	shortIDRe     = regexp.MustCompile(`^[A-Z]{2,5}_[0-9A-Za-z]{22}$`)
	currencyRe    = regexp.MustCompile(`^[A-Z]{3}$`)
	countryCodeRe = regexp.MustCompile(`^[A-Z]{2}$`)
	amountRe      = regexp.MustCompile(`^\d+(\.\d+)?$`)
)

var shortIDLabels = map[string]string{
	"STO":  "Store",
	"PROD": "Product",
	"ORD":  "Order",
	"PAY":  "Payment",
	"REF":  "Refund",
	"TKT":  "Ticket",
	"MER":  "Merchant",
}

// validateRequired returns an SDK-layer error when v is the empty string.
func validateRequired(field, v string) error {
	if v == "" {
		return newSDKError("Missing required field: %s", field)
	}
	return nil
}

// validateShortID checks the {PREFIX}_{base62} Short ID shape.
func validateShortID(field, v, prefix string) error {
	if err := validateRequired(field, v); err != nil {
		return err
	}
	label := shortIDLabels[prefix]
	if label == "" {
		label = prefix
	}
	if !shortIDRe.MatchString(v) {
		return newSDKError("Invalid %s: expected %s Short ID format (%s_xxx), got %q", field, label, prefix, v)
	}
	if len(v) < len(prefix)+1 || v[:len(prefix)+1] != prefix+"_" {
		return newSDKError("Invalid %s: expected %s_ prefix (%s)", field, prefix, label)
	}
	return nil
}

// validateCurrencyCode checks ISO 4217 three-letter uppercase codes.
func validateCurrencyCode(field, v string) error {
	if err := validateRequired(field, v); err != nil {
		return err
	}
	if !currencyRe.MatchString(v) {
		return newSDKError("Invalid %s: expected 3-letter ISO 4217 currency code (e.g., \"USD\"), got %q", field, v)
	}
	return nil
}

// validateAmountString checks display-format numeric strings ("9.99", "1000").
func validateAmountString(field, v string) error {
	if err := validateRequired(field, v); err != nil {
		return err
	}
	if !amountRe.MatchString(v) {
		return newSDKError("Invalid %s: expected numeric string in display format (e.g., \"9.99\", \"1000\"), got %q", field, v)
	}
	return nil
}

// validateCountryCode checks ISO 3166-1 alpha-2 country codes.
func validateCountryCode(field, v string) error {
	if err := validateRequired(field, v); err != nil {
		return err
	}
	if !countryCodeRe.MatchString(v) {
		return newSDKError("Invalid %s: expected 2-letter ISO 3166-1 country code (e.g., \"US\"), got %q", field, v)
	}
	return nil
}

// validatePositiveInt checks v > 0.
func validatePositiveInt(field string, v int) error {
	if v <= 0 {
		return newSDKError("Invalid %s: expected positive integer, got %d", field, v)
	}
	return nil
}

// validateMaxLength checks that an optional string does not exceed limit characters.
func validateMaxLength(field string, v *string, limit int) error {
	if v != nil && len(*v) > limit {
		return newSDKError("%s must be at most %d characters, got %d", field, limit, len(*v))
	}
	return nil
}

// validatePrices checks Prices entries — each currency key and price amount.
func validatePrices(field string, prices Prices) error {
	if len(prices) == 0 {
		return newSDKError("%s must contain at least one currency", field)
	}
	for currency, info := range prices {
		if err := validateCurrencyCode(field+"."+currency+" (key)", currency); err != nil {
			return err
		}
		if err := validateAmountString(field+"."+currency+".amount", info.Amount); err != nil {
			return err
		}
		if err := validateRequired(field+"."+currency+".taxCategory", string(info.TaxCategory)); err != nil {
			return err
		}
	}
	return nil
}

// validateBillingDetail checks BillingDetail required fields.
func validateBillingDetail(d *BillingDetail) error {
	if d == nil {
		return nil
	}
	return validateCountryCode("billingDetail.country", d.Country)
}

// supportedPaymentMethods lists the payment method identifiers currently
// supported by the hosted cashier.
var supportedPaymentMethods = map[string]bool{
	"CREDITCARD": true,
	"DEBITCARD":  true,
	"APPLEPAY":   true,
	"GOOGLEPAY":  true,
	"EWALLET":    true,
}

// validatePaymentMethods checks an optional ordered payment-methods allow-list:
// non-empty when present, no duplicates, and every value a known identifier.
// This only catches obviously malformed input client-side — real availability
// (currency/product type/environment) is always re-validated server-side and
// cannot be bypassed by skipping this check.
func validatePaymentMethods(field string, methods []string) error {
	if methods == nil {
		return nil
	}
	if len(methods) == 0 {
		return newSDKError("%s must not be empty when provided", field)
	}
	seen := make(map[string]bool, len(methods))
	for _, m := range methods {
		if seen[m] {
			return newSDKError("%s must not contain duplicate values", field)
		}
		seen[m] = true
		if !supportedPaymentMethods[m] {
			return newSDKError("Invalid %s entry: expected one of [CREDITCARD, DEBITCARD, APPLEPAY, GOOGLEPAY, EWALLET], got %q", field, m)
		}
	}
	return nil
}

// validateCheckoutCommon runs the shared checks for Checkout endpoints.
func validateCheckoutCommon(p *CreateCheckoutSessionParams) error {
	if err := validateShortID("productId", p.ProductID, "PROD"); err != nil {
		return err
	}
	if err := validateCurrencyCode("currency", p.Currency); err != nil {
		return err
	}
	if p.PriceSnapshot != nil {
		if err := validateAmountString("priceSnapshot.amount", p.PriceSnapshot.Amount); err != nil {
			return err
		}
		if err := validateRequired("priceSnapshot.taxCategory", string(p.PriceSnapshot.TaxCategory)); err != nil {
			return err
		}
	}
	if err := validateBillingDetail(p.BillingDetail); err != nil {
		return err
	}
	if p.ExpiresInSeconds != nil {
		if err := validatePositiveInt("expiresInSeconds", *p.ExpiresInSeconds); err != nil {
			return err
		}
	}
	if err := validateMaxLength("orderMerchantExternalId", p.OrderMerchantExternalID, 128); err != nil {
		return err
	}
	if err := validatePaymentMethods("paymentMethods", p.PaymentMethods); err != nil {
		return err
	}
	return nil
}
