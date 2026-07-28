// Cashier language marshal contract tests.
//
// CreateCheckoutSessionParams.Language sets the hosted checkout page's default
// language. JSON uses the IETF BCP 47 tag, aligned with sdk-ts CashierLanguage.
package pancake

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCreateCheckoutSessionParams_Language_Marshal(t *testing.T) {
	p := CreateCheckoutSessionParams{
		ProductID: "PROD_xxx",
		Currency:  "BRL",
		Language:  Ptr(CashierLanguagePtBR),
	}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"language":"pt-BR"`) {
		t.Fatalf("expected language in payload, got %s", out)
	}
}

func TestCreateCheckoutSessionParams_Language_Omitted(t *testing.T) {
	p := CreateCheckoutSessionParams{ProductID: "PROD_xxx", Currency: "USD"}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), "language") {
		t.Fatalf("expected omitempty to drop field, got %s", out)
	}
}

func TestCashierLanguageConstants(t *testing.T) {
	// 与 @waffo-com/pancake-core SUPPORTED_CASHIER_LANGUAGES 保持一致的 22 个标签
	all := []CashierLanguage{
		CashierLanguageEn, CashierLanguagePtBR, CashierLanguageEsMX, CashierLanguageIDID,
		CashierLanguageViVN, CashierLanguageRuRU, CashierLanguageEnKE, CashierLanguageEsPE,
		CashierLanguageEsCO, CashierLanguageEsCL, CashierLanguageZhHantTW, CashierLanguageZhHantHK,
		CashierLanguageThTH, CashierLanguageJaJP, CashierLanguageEnNG, CashierLanguageKoKR,
		CashierLanguageEnHK, CashierLanguageZhHansHK, CashierLanguagePlPL, CashierLanguageTrTR,
		CashierLanguageZhHans, CashierLanguageMsMY,
	}
	if len(all) != 22 {
		t.Fatalf("expected 22 cashier languages, got %d", len(all))
	}
	seen := map[CashierLanguage]bool{}
	for _, l := range all {
		if l == "" {
			t.Fatal("empty language constant")
		}
		if seen[l] {
			t.Fatalf("duplicate language constant %q", l)
		}
		seen[l] = true
	}
	if CashierLanguageIDID != "id-ID" || CashierLanguageZhHantTW != "zh-Hant-TW" {
		t.Fatal("language tag value mismatch")
	}
}
