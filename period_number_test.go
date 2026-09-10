// Billing-period-number contract tests for the webhook payload.
//
// periodNumber is reported by the payment channel and pointer-typed because three
// subscription events (canceling / uncanceled / plan_change_failed) and the one-time
// order and refund events carry no period number at all. Period 0 is a real channel
// value (authorized, not yet charged), so "absent" and "zero" must stay distinguishable.
package pancake

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWebhookEventData_PeriodNumber_Unmarshal(t *testing.T) {
	raw := `{"orderId":"ORD_x","buyerEmail":"b@example.com","currency":"USD","amount":"29.00","taxAmount":"0","productName":"Pro Plan","periodNumber":3}`

	var data WebhookEventData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if data.PeriodNumber == nil {
		t.Fatalf("expected PeriodNumber to be set")
	}
	if *data.PeriodNumber != 3 {
		t.Fatalf("expected PeriodNumber 3, got %d", *data.PeriodNumber)
	}
}

func TestWebhookEventData_PeriodNumber_ZeroIsNotAbsent(t *testing.T) {
	raw := `{"orderId":"ORD_x","buyerEmail":"b@example.com","currency":"USD","amount":"29.00","taxAmount":"0","productName":"Pro Plan","periodNumber":0}`

	var data WebhookEventData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if data.PeriodNumber == nil {
		t.Fatalf("expected period 0 to survive as a value, got nil")
	}
	if *data.PeriodNumber != 0 {
		t.Fatalf("expected PeriodNumber 0, got %d", *data.PeriodNumber)
	}
}

func TestWebhookEventData_PeriodNumber_AbsentStaysNil(t *testing.T) {
	// subscription.canceling — the channel has no notification for it, so no period number
	raw := `{"orderId":"ORD_x","buyerEmail":"b@example.com","currency":"USD","amount":"29.00","taxAmount":"0","productName":"Pro Plan","billingPeriod":"monthly"}`

	var data WebhookEventData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if data.PeriodNumber != nil {
		t.Fatalf("expected PeriodNumber nil, got %d", *data.PeriodNumber)
	}
}

func TestWebhookEventData_PeriodNumber_OmittedWhenNil(t *testing.T) {
	out, err := json.Marshal(WebhookEventData{OrderID: "ORD_x"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), "periodNumber") {
		t.Fatalf("expected omitempty to drop field, got %s", out)
	}
}
