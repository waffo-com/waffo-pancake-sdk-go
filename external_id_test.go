// External-id field marshal/unmarshal contract tests for the flat dual-key
// design (orderMerchantExternalId / refundTicketMerchantExternalId).
//
// The fields appear in four shapes consistently:
//   - CreateCheckoutSessionParams.OrderMerchantExternalID (write-side input)
//   - CreateRefundTicketParams.RefundTicketMerchantExternalID (write-side input)
//   - RefundTicket.RefundTicketMerchantExternalID (read-side response)
//   - WebhookEventData.{OrderMerchantExternalID, RefundTicketMerchantExternalID} (webhook payload)
//
// JSON tags use camelCase keys aligned with sdk-ts / GraphQL / webhook envelope.
package pancake

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCreateCheckoutSessionParams_OrderMerchantExternalID_Marshal(t *testing.T) {
	p := CreateCheckoutSessionParams{
		ProductID:               "PROD_xxx",
		Currency:                "USD",
		OrderMerchantExternalID: Ptr("ORDER-REF-2026-00891"),
	}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, `"orderMerchantExternalId":"ORDER-REF-2026-00891"`) {
		t.Fatalf("expected orderMerchantExternalId in payload, got %s", got)
	}

	// nil pointer should omit the field
	p2 := CreateCheckoutSessionParams{ProductID: "PROD_xxx", Currency: "USD"}
	out2, _ := json.Marshal(p2)
	if strings.Contains(string(out2), "orderMerchantExternalId") {
		t.Fatalf("expected omitempty to drop field, got %s", out2)
	}
}

func TestCreateRefundTicketParams_RefundTicketMerchantExternalID_Marshal(t *testing.T) {
	p := CreateRefundTicketParams{
		PaymentID:                      "PAY_xxx",
		Reason:                         "defective",
		RequestedAmount:                RequestedAmount{Amount: "29.00", Currency: "USD"},
		RefundTicketMerchantExternalID: Ptr("REF-2026-00012"),
	}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"refundTicketMerchantExternalId":"REF-2026-00012"`) {
		t.Fatalf("expected refundTicketMerchantExternalId in payload, got %s", out)
	}
}

func TestRefundTicket_RefundTicketMerchantExternalID_Unmarshal(t *testing.T) {
	raw := []byte(`{
		"id":"TKT_xxx",
		"type":"refund",
		"status":"processing",
		"subjectId":"PAY_xxx",
		"submitterId":"MER_xxx",
		"submitterType":"merchant",
		"currentVersionId":null,"reviewerId":null,"reviewedAt":null,"reviewNote":null,
		"rejectReason":null,"executedAt":null,
		"metadata":{},
		"versionNumber":1,"versionData":null,
		"refundTicketMerchantExternalId":"REF-2026-00012",
		"createdAt":"2026-05-19T00:00:00Z","updatedAt":"2026-05-19T00:00:00Z"
	}`)
	var t1 RefundTicket
	if err := json.Unmarshal(raw, &t1); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if t1.RefundTicketMerchantExternalID == nil || *t1.RefundTicketMerchantExternalID != "REF-2026-00012" {
		t.Fatalf("expected refundTicketMerchantExternalId=REF-2026-00012, got %v", t1.RefundTicketMerchantExternalID)
	}

	raw2 := []byte(`{"id":"TKT_xxx","type":"refund","status":"pending","subjectId":"PAY_xxx","submitterId":"x","submitterType":"buyer","currentVersionId":null,"reviewerId":null,"reviewedAt":null,"reviewNote":null,"rejectReason":null,"executedAt":null,"metadata":{},"versionNumber":null,"versionData":null,"refundTicketMerchantExternalId":null,"createdAt":"x","updatedAt":"x"}`)
	var t2 RefundTicket
	if err := json.Unmarshal(raw2, &t2); err != nil {
		t.Fatalf("unmarshal nil: %v", err)
	}
	if t2.RefundTicketMerchantExternalID != nil {
		t.Fatalf("expected nil pointer for null json, got %v", t2.RefundTicketMerchantExternalID)
	}
}

func TestWebhookEventData_DualKey_Refund(t *testing.T) {
	raw := []byte(`{
		"orderId":"ORD_xxx","buyerEmail":"b@x.com","currency":"USD",
		"orderMerchantExternalId":"ORDER-REF-1",
		"refundTicketMerchantExternalId":"REF-2026-00012",
		"amount":"10.00","taxAmount":"0","productName":"P",
		"refundStatus":"succeeded"
	}`)
	var d WebhookEventData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if d.OrderMerchantExternalID == nil || *d.OrderMerchantExternalID != "ORDER-REF-1" {
		t.Fatalf("expected orderMerchantExternalId=ORDER-REF-1, got %v", d.OrderMerchantExternalID)
	}
	if d.RefundTicketMerchantExternalID == nil || *d.RefundTicketMerchantExternalID != "REF-2026-00012" {
		t.Fatalf("expected refundTicketMerchantExternalId=REF-2026-00012, got %v", d.RefundTicketMerchantExternalID)
	}
}

func TestWebhookEventData_DualKey_OrderOnly(t *testing.T) {
	raw := []byte(`{
		"orderId":"ORD_xxx","buyerEmail":"b@x.com","currency":"USD",
		"orderMerchantExternalId":"ORDER-REF-1",
		"amount":"10.00","taxAmount":"0","productName":"P"
	}`)
	var d WebhookEventData
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if d.OrderMerchantExternalID == nil {
		t.Fatal("expected orderMerchantExternalId set")
	}
	if d.RefundTicketMerchantExternalID != nil {
		t.Fatalf("non-refund event should not carry refundTicket key, got %v", d.RefundTicketMerchantExternalID)
	}
}

func TestCreateCheckoutSession_RejectsOrderMerchantExternalIDOver128(t *testing.T) {
	client, _, _ := newSignedTestClient(t)
	long := strings.Repeat("x", 129)
	if _, err := client.Checkout.CreateSession(context.Background(), CreateCheckoutSessionParams{
		ProductID:               "PROD_AbCdEfGhIjKlMnOpQrStUv",
		Currency:                "USD",
		OrderMerchantExternalID: &long,
	}); err == nil {
		t.Fatal("expected error for 129-char orderMerchantExternalId")
	}
}

func TestCustomerCreateRefundTicket_RejectsRefundTicketMerchantExternalIDOver128(t *testing.T) {
	client, _, _ := newSignedTestClient(t)
	long := strings.Repeat("x", 129)
	if _, err := client.Customer("token").CreateRefundTicket(context.Background(), CreateRefundTicketParams{
		PaymentID:                      "PAY_AbCdEfGhIjKlMnOpQrStUv",
		Reason:                         "defective",
		RequestedAmount:                RequestedAmount{Amount: "29.00", Currency: "USD"},
		RefundTicketMerchantExternalID: &long,
	}); err == nil {
		t.Fatal("expected error for 129-char refundTicketMerchantExternalId")
	}
}
