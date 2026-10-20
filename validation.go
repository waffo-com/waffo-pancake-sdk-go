package pancake

import (
	"regexp"
	"strings"
	"unicode/utf16"
)

var (
	shortIDRe      = regexp.MustCompile(`^[A-Z]{2,5}_[0-9A-Za-z]{22}$`)
	currencyRe     = regexp.MustCompile(`^[A-Z]{3}$`)
	countryCodeRe  = regexp.MustCompile(`^[A-Z]{2}$`)
	amountRe       = regexp.MustCompile(`^\d+(\.\d+)?$`)
	discountCodeRe = regexp.MustCompile(`^[A-Za-z0-9-]{3,32}$`)
	codePrefixRe   = regexp.MustCompile(`^[A-Za-z0-9-]{1,12}$`)
)

var shortIDLabels = map[string]string{
	"STO":  "Store",
	"PROD": "Product",
	"ORD":  "Order",
	"PAY":  "Payment",
	"REF":  "Refund",
	"TKT":  "Ticket",
	"MER":  "Merchant",
	"CPN":  "Coupon",
	"PMC":  "Promotion Code",
}

// validateRequired returns an SDK-layer error when v is empty or contains only
// whitespace.
func validateRequired(field, v string) error {
	if strings.TrimSpace(v) == "" {
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

// validateMaxLength checks that an optional string does not exceed limit
// characters. Length is counted in UTF-16 code units, as the server's
// JavaScript string length is, so multi-byte text is not over-counted.
func validateMaxLength(field string, v *string, limit int) error {
	if v == nil {
		return nil
	}
	if n := utf16Length(*v); n > limit {
		return newSDKError("%s must be at most %d characters, got %d", field, limit, n)
	}
	return nil
}

// utf16Length returns the length of s in UTF-16 code units (JavaScript
// String.prototype.length).
func utf16Length(s string) int {
	return len(utf16.Encode([]rune(s)))
}

// validatePositiveAmountString checks a display-format numeric string that
// must be greater than zero ("0" and "0.00" are rejected).
func validatePositiveAmountString(field, v string) error {
	if err := validateAmountString(field, v); err != nil {
		return err
	}
	// amountRe already allows only digits and one '.', so the amount is zero
	// exactly when no digit other than '0' remains.
	if strings.Trim(strings.Replace(v, ".", "", 1), "0") == "" {
		return newSDKError("Invalid %s: expected an amount greater than 0, got %q", field, v)
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

// validateSessionDiscount checks the discount fields of a checkout or plan change
// session.
//
// CouponID is mutually exclusive with both PromotionCode and AllowPromotionCodes: a
// session either has a coupon applied and locked by the merchant, or carries a
// prefilled promotion code and / or a code entry field. Setting CouponID with either
// (with any AllowPromotionCodes value) is rejected here, as the platform would reject
// it with a 400. PromotionCode must not be empty or whitespace only.
func validateSessionDiscount(couponID, promotionCode *string, allowPromotionCodes *bool) error {
	if promotionCode != nil && strings.TrimSpace(*promotionCode) == "" {
		return newSDKError("promotionCode cannot be empty")
	}
	if couponID != nil {
		if err := validateShortID("couponId", *couponID, "CPN"); err != nil {
			return err
		}
		if promotionCode != nil {
			return newSDKError("couponId and promotionCode are mutually exclusive: pass couponId to apply a coupon, or promotionCode to prefill a promotion code")
		}
		if allowPromotionCodes != nil {
			return newSDKError("couponId and allowPromotionCodes are mutually exclusive: pass couponId to apply a coupon, or allowPromotionCodes to let the customer enter a code")
		}
	}
	return nil
}

// validateCheckoutCommon runs the shared checks for Checkout endpoints, including
// the discount fields (see validateSessionDiscount).
func validateCheckoutCommon(p *CreateCheckoutSessionParams) error {
	if err := validateSessionCommon(p.ProductID, p.Currency, p.PriceSnapshot, p.ExpiresInSeconds, p.OrderMerchantExternalID); err != nil {
		return err
	}
	if err := validateBillingDetail(p.BillingDetail); err != nil {
		return err
	}
	return validateSessionDiscount(p.CouponID, p.PromotionCode, p.AllowPromotionCodes)
}

// validatePlanChangeCommon runs the shared checks for plan change endpoints.
//
// It covers the same ground as validateCheckoutCommon — including the discount
// fields — minus BillingDetail (plan change mode takes it from the origin
// subscription) plus the plan-change-only fields. The other mode rules (which field
// may accompany which) belong to the platform — this only catches malformed input
// before the request goes out.
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
	return validateSessionDiscount(p.CouponID, p.PromotionCode, p.AllowPromotionCodes)
}

// validateDiscountCode checks the code format shared by coupons and promotion
// codes: 3-32 letters, digits or '-'.
func validateDiscountCode(field, v string) error {
	if err := validateRequired(field, v); err != nil {
		return err
	}
	if !discountCodeRe.MatchString(v) {
		return newSDKError("Invalid %s: expected 3-32 letters, digits or '-', got %q", field, v)
	}
	return nil
}

// validateCouponName checks that a coupon name is present and at most 64
// characters after trimming, counted in UTF-16 code units as the server does.
func validateCouponName(name string) error {
	if err := validateRequired("name", name); err != nil {
		return err
	}
	trimmed := strings.TrimSpace(name)
	return validateMaxLength("name", &trimmed, 64)
}
