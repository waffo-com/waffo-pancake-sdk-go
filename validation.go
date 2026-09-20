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

// validateEnvironment checks that env is one of the two known environments.
func validateEnvironment(field string, env Environment) error {
	switch env {
	case EnvironmentTest, EnvironmentProd:
		return nil
	case "":
		return newSDKError(
			"Missing required field: %s — set Config.Environment or use client.CustomerWithEnvironment(token, pancake.EnvironmentTest)",
			field,
		)
	default:
		return newSDKError("Invalid %s: expected one of [test, prod], got %q", field, string(env))
	}
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

// validateSessionCommon runs the checks shared by every create-session shape:
// new purchase (validateCheckoutCommon) and plan change (validatePlanChangeCommon).
func validateSessionCommon(productID, currency string, priceSnapshot *PriceSnapshot, expiresInSeconds *int, orderMerchantExternalID *string) error {
	if err := validateShortID("productId", productID, "PROD"); err != nil {
		return err
	}
	if err := validateCurrencyCode("currency", currency); err != nil {
		return err
	}
	if priceSnapshot != nil {
		if err := validateAmountString("priceSnapshot.amount", priceSnapshot.Amount); err != nil {
			return err
		}
		if err := validateRequired("priceSnapshot.taxCategory", string(priceSnapshot.TaxCategory)); err != nil {
			return err
		}
	}
	if expiresInSeconds != nil {
		if err := validatePositiveInt("expiresInSeconds", *expiresInSeconds); err != nil {
			return err
		}
	}
	return validateMaxLength("orderMerchantExternalId", orderMerchantExternalID, 128)
}

// validateCheckoutCommon runs the shared checks for Checkout endpoints.
func validateCheckoutCommon(p *CreateCheckoutSessionParams) error {
	if err := validateSessionCommon(p.ProductID, p.Currency, p.PriceSnapshot, p.ExpiresInSeconds, p.OrderMerchantExternalID); err != nil {
		return err
	}
	return validateBillingDetail(p.BillingDetail)
}

// validatePlanChangeCommon runs the shared checks for plan change endpoints.
//
// It covers the same ground as validateCheckoutCommon minus BillingDetail (plan
// change mode takes it from the origin subscription) plus the plan-change-only
// fields. The mode rules themselves (which field may accompany which) belong to
// the platform — this only catches malformed input before the request goes out.
func validatePlanChangeCommon(p *CreatePlanChangeSessionParams) error {
	if err := validateSessionCommon(p.ProductID, p.Currency, p.PriceSnapshot, p.ExpiresInSeconds, p.OrderMerchantExternalID); err != nil {
		return err
	}
	if err := validateShortID("originOrderId", p.OriginOrderID, "ORD"); err != nil {
		return err
	}
	if p.ChangeAmount != nil {
		if err := validateAmountString("changeAmount", *p.ChangeAmount); err != nil {
			return err
		}
	}
	if p.ChangeCreditAmount != nil {
		if err := validateAmountString("changeCreditAmount", *p.ChangeCreditAmount); err != nil {
			return err
		}
	}
	return nil
}
