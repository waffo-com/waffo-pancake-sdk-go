// Plan change contract tests.
//
// A plan change is the create-session endpoint in the mode selected by
// OriginOrderID. The tests below pin the three things callers depend on:
// the plan change fields reach the request body unchanged, the returned URL is
// the change confirmation page, and the new-purchase params stay free of plan
// change fields (the anonymous alias derives from them, so it is covered too).
package pancake

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

const (
	testOriginOrderID = "ORD_AbCdEfGhIjKlMnOpQrStUv"
	testTargetProduct = "PROD_AbCdEfGhIjKlMnOpQrStUv"
)

func planChangeParams() CreatePlanChangeSessionParams {
	return CreatePlanChangeSessionParams{
		OriginOrderID: testOriginOrderID,
		ProductID:     testTargetProduct,
		Currency:      "USD",
	}
}

func changeSessionResponder(req recordedRequest) (int, any) {
	switch {
	case strings.HasSuffix(req.Path, "/issue-session-token"):
		return 200, map[string]any{"data": map[string]any{"token": "JWT", "expiresAt": "2026-05-13T01:00:00Z"}}
	case strings.HasSuffix(req.Path, "/create-session"):
		return 200, map[string]any{"data": map[string]any{
			"sessionId":   "ses_change",
			"checkoutUrl": "https://pancake.example/store/my-store/change/ses_change",
			"expiresAt":   "2026-05-13T00:45:00Z",
		}}
	}
	return 404, map[string]any{"data": nil, "errors": []map[string]string{{"message": "no", "layer": "gateway"}}}
}

func requestBody(t *testing.T, server *recordingServer, pathSuffix string) map[string]any {
	t.Helper()
	for _, req := range server.requests() {
		if strings.HasSuffix(req.Path, pathSuffix) {
			var body map[string]any
			if err := json.Unmarshal(req.Body, &body); err != nil {
				t.Fatalf("unmarshal %s body: %v", pathSuffix, err)
			}
			return body
		}
	}
	t.Fatalf("no request to %s", pathSuffix)
	return nil
}

func TestCheckoutCreatePlanChangeSession_SendsChangeFields(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = changeSessionResponder

	p := planChangeParams()
	p.ChangeTiming = Ptr(ChangeTimingImmediate)
	p.ChangeAmount = Ptr("12.00")

	res, err := client.Checkout.CreatePlanChangeSession(context.Background(), p)
	if err != nil {
		t.Fatalf("plan change: %v", err)
	}

	body := requestBody(t, server, "/create-session")
	if body["originOrderId"] != testOriginOrderID {
		t.Errorf("originOrderId = %v want %s", body["originOrderId"], testOriginOrderID)
	}
	if body["changeTiming"] != "immediate" {
		t.Errorf("changeTiming = %v want immediate", body["changeTiming"])
	}
	if body["changeAmount"] != "12.00" {
		t.Errorf("changeAmount = %v want 12.00", body["changeAmount"])
	}
	// The path segment after the store slug is `change`, not `checkout`.
	if !strings.Contains(res.CheckoutURL, "/change/") {
		t.Errorf("CheckoutURL is not a change link: %s", res.CheckoutURL)
	}
}

func TestCheckoutCreatePlanChangeSession_ForwardsBothAmountsUnchanged(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = changeSessionResponder

	p := planChangeParams()
	p.ChangeAmount = Ptr("12.00")
	p.ChangeCreditAmount = Ptr("8.00")

	if _, err := client.Checkout.CreatePlanChangeSession(context.Background(), p); err != nil {
		t.Fatalf("plan change: %v", err)
	}

	// The two are mutually exclusive, but the SDK neither rejects nor drops one
	// side — the platform answers with a 400.
	body := requestBody(t, server, "/create-session")
	if body["changeAmount"] != "12.00" || body["changeCreditAmount"] != "8.00" {
		t.Errorf("expected both amount fields forwarded, got %v", body)
	}
}

func TestCreatePlanChangeSessionParams_OmitsUnsetFields(t *testing.T) {
	out, err := json.Marshal(planChangeParams())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)
	for _, field := range []string{"changeTiming", "changeAmount", "changeCreditAmount", "withTrial", "priceSnapshot"} {
		if strings.Contains(got, field) {
			t.Errorf("expected omitempty to drop %s, got %s", field, got)
		}
	}
	if !strings.Contains(got, `"originOrderId":"`+testOriginOrderID+`"`) {
		t.Errorf("originOrderId must always be sent, got %s", got)
	}
}

func TestCheckoutCreatePlanChangeSession_ValidatesInput(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = changeSessionResponder

	cases := []struct {
		name   string
		mutate func(*CreatePlanChangeSessionParams)
	}{
		{"missing originOrderId", func(p *CreatePlanChangeSessionParams) { p.OriginOrderID = "" }},
		{"originOrderId is not an order id", func(p *CreatePlanChangeSessionParams) { p.OriginOrderID = testTargetProduct }},
		{"malformed changeAmount", func(p *CreatePlanChangeSessionParams) { p.ChangeAmount = Ptr("twelve") }},
		{"malformed changeCreditAmount", func(p *CreatePlanChangeSessionParams) { p.ChangeCreditAmount = Ptr("") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := planChangeParams()
			tc.mutate(&p)
			if _, err := client.Checkout.CreatePlanChangeSession(context.Background(), p); err == nil {
				t.Fatal("expected an SDK validation error")
			}
		})
	}
	if len(server.requests()) != 0 {
		t.Fatalf("expected no request to be sent, got %d", len(server.requests()))
	}
}

func TestCheckoutAuthenticatedCreatePlanChange_AppendsTokenFragment(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = changeSessionResponder

	res, err := client.Checkout.Authenticated.CreatePlanChange(context.Background(), AuthenticatedPlanChangeParams{
		CreatePlanChangeSessionParams: planChangeParams(),
		BuyerIdentity:                 "user-1",
	})
	if err != nil {
		t.Fatalf("plan change: %v", err)
	}
	if !strings.HasSuffix(res.CheckoutURL, "/change/ses_change#token=JWT") {
		t.Fatalf("CheckoutURL is not a tokenized change link: %s", res.CheckoutURL)
	}

	sessionBody := requestBody(t, server, "/create-session")
	if sessionBody["originOrderId"] != testOriginOrderID {
		t.Errorf("originOrderId = %v want %s", sessionBody["originOrderId"], testOriginOrderID)
	}
	if _, present := sessionBody["buyerIdentity"]; present {
		t.Error("buyerIdentity must not leak into the session body")
	}

	tokenBody := requestBody(t, server, "/issue-session-token")
	if tokenBody["buyerIdentity"] != "user-1" || tokenBody["productId"] != testTargetProduct {
		t.Errorf("unexpected token request body: %v", tokenBody)
	}
	if _, present := tokenBody["originOrderId"]; present {
		t.Error("originOrderId must not be sent to issue-session-token")
	}
}

func TestCheckoutAuthenticatedCreatePlanChange_RequiresBuyerIdentity(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = changeSessionResponder

	_, err := client.Checkout.Authenticated.CreatePlanChange(context.Background(), AuthenticatedPlanChangeParams{
		CreatePlanChangeSessionParams: planChangeParams(),
	})
	if err == nil {
		t.Fatal("expected an SDK validation error")
	}
	if len(server.requests()) != 0 {
		t.Fatalf("expected no request to be sent, got %d", len(server.requests()))
	}
}

func TestChangeTimingConstants(t *testing.T) {
	cases := map[ChangeTiming]string{
		ChangeTimingImmediate:  "immediate",
		ChangeTimingNextPeriod: "next_period",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Fatalf("expected %q, got %q", want, string(got))
		}
	}
}

func TestNewPurchaseParams_CarryNoPlanChangeFields(t *testing.T) {
	// AnonymousCheckoutParams is an alias of CreateCheckoutSessionParams, so one
	// marshal covers both: no plan change field can reach the anonymous path.
	out, err := json.Marshal(AnonymousCheckoutParams{ProductID: testTargetProduct, Currency: "USD"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, field := range []string{"originOrderId", "changeTiming", "changeAmount", "changeCreditAmount"} {
		if strings.Contains(string(out), field) {
			t.Fatalf("new-purchase params must not carry %s, got %s", field, out)
		}
	}
}

func TestGroupRulesInput_MergeSemantics(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	// update-group merges rules field by field, so the response carries the stored
	// sharedTrial (true) even though the request only flipped selfServicePlanChange.
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"group": map[string]any{
			"id":          "GRP_x",
			"storeId":     "STO_AbCdEfGhIjKlMnOpQrStUv",
			"name":        "G",
			"rules":       map[string]any{"sharedTrial": true, "selfServicePlanChange": true},
			"productIds":  []any{},
			"environment": "test",
			"createdAt":   "z",
			"updatedAt":   "z",
		}}}
	}

	res, err := client.SubscriptionProductGroups.Update(context.Background(), UpdateSubscriptionProductGroupParams{
		ID:    "GRP_x",
		Rules: &GroupRulesInput{SelfServicePlanChange: Ptr(true)},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	body := requestBody(t, server, "/update-group")
	rules, ok := body["rules"].(map[string]any)
	if !ok {
		t.Fatalf("rules missing from request body: %v", body)
	}
	if _, present := rules["sharedTrial"]; present {
		t.Errorf("an unset switch must be omitted, got %v", rules)
	}
	if rules["selfServicePlanChange"] != true {
		t.Errorf("selfServicePlanChange = %v want true", rules["selfServicePlanChange"])
	}
	// The entity always carries both switches, so reading one never needs a fallback.
	if !res.Group.Rules.SharedTrial || !res.Group.Rules.SelfServicePlanChange {
		t.Errorf("entity rules = %+v want both switches on", res.Group.Rules)
	}
}

func TestGroupRulesInput_SendsBothSwitches(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{"group": map[string]any{
			"id":          "GRP_x",
			"storeId":     "STO_AbCdEfGhIjKlMnOpQrStUv",
			"name":        "G",
			"rules":       map[string]any{"sharedTrial": true, "selfServicePlanChange": false},
			"productIds":  []any{},
			"environment": "test",
			"createdAt":   "z",
			"updatedAt":   "z",
		}}}
	}

	res, err := client.SubscriptionProductGroups.Create(context.Background(), CreateSubscriptionProductGroupParams{
		StoreID: "STO_AbCdEfGhIjKlMnOpQrStUv",
		Name:    "G",
		Rules:   &GroupRulesInput{SharedTrial: Ptr(true), SelfServicePlanChange: Ptr(false)},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	body := requestBody(t, server, "/create-group")
	rules, ok := body["rules"].(map[string]any)
	if !ok {
		t.Fatalf("rules missing from request body: %v", body)
	}
	if rules["sharedTrial"] != true || rules["selfServicePlanChange"] != false {
		t.Errorf("unexpected rules payload: %v", rules)
	}
	if res.Group.Rules.SelfServicePlanChange {
		t.Error("entity selfServicePlanChange should be false")
	}
}

func TestCustomerCreatePlanChangeSession_UsesBearerAndNoIdempotencyKey(t *testing.T) {
	_, customer, srv := newCustomerTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{
			"sessionId":   "ses_self_service",
			"checkoutUrl": "https://pancake.example/store/my-store/change/ses_self_service",
			"expiresAt":   "2026-05-13T00:45:00Z",
		}}
	}

	res, err := customer.CreatePlanChangeSession(context.Background(), CustomerPlanChangeParams{
		OriginOrderID: testOriginOrderID,
		ProductID:     testTargetProduct,
		Currency:      "USD",
		ChangeTiming:  Ptr(ChangeTimingNextPeriod),
	})
	if err != nil {
		t.Fatalf("plan change: %v", err)
	}
	if !strings.Contains(res.CheckoutURL, "/change/") {
		t.Errorf("CheckoutURL is not a change link: %s", res.CheckoutURL)
	}

	reqs := srv.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]
	if req.Path != "/v1/actions/checkout/create-session" {
		t.Errorf("path = %q", req.Path)
	}
	if got := req.Headers.Get("Authorization"); got != "Bearer JWT_CUSTOMER_TOKEN" {
		t.Errorf("Authorization = %q", got)
	}
	if got := req.Headers.Get("X-Environment"); got != "test" {
		t.Errorf("X-Environment = %q", got)
	}
	// Customer session requests are outside the gateway idempotency cache, and
	// carry no merchant signature — the platform sees a customer issuer.
	if got := req.Headers.Get("X-Idempotency-Key"); got != "" {
		t.Errorf("expected no idempotency key, got %q", got)
	}
	if got := req.Headers.Get("X-Signature"); got != "" {
		t.Errorf("expected no merchant signature, got %q", got)
	}

	var body map[string]any
	if err := json.Unmarshal(req.Body, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["originOrderId"] != testOriginOrderID || body["changeTiming"] != "next_period" {
		t.Errorf("unexpected body: %v", body)
	}
}

func TestCustomerPlanChangeParams_CarryNoMerchantOnlyFields(t *testing.T) {
	// The platform silently drops every API-Key-only field on a customer-session
	// request, so they are absent from the struct instead of accepted and ignored.
	out, err := json.Marshal(CustomerPlanChangeParams{
		OriginOrderID: testOriginOrderID,
		ProductID:     testTargetProduct,
		Currency:      "USD",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, field := range []string{
		"changeAmount", "changeCreditAmount", "withTrial", "priceSnapshot",
		"expiresInSeconds", "metadata", "orderMerchantExternalId",
		"includePaymentMethods", "excludePaymentMethods", "buyerEmail", "billingDetail",
	} {
		if strings.Contains(string(out), field) {
			t.Errorf("customer params must not carry %s, got %s", field, out)
		}
	}

	// Compile-time cross-check: the merchant-only fields exist on the merchant type,
	// so their absence here is a deliberate narrowing, not an oversight.
	merchant := CreatePlanChangeSessionParams{ChangeAmount: Ptr("12.00"), WithTrial: Ptr(true)}
	if merchant.ChangeAmount == nil || merchant.WithTrial == nil {
		t.Fatal("merchant params should carry the API-Key-only fields")
	}
}

func TestCustomerCreatePlanChangeSession_ValidatesInput(t *testing.T) {
	_, customer, srv := newCustomerTestClient(t)

	cases := []struct {
		name   string
		params CustomerPlanChangeParams
	}{
		{"bad originOrderId", CustomerPlanChangeParams{OriginOrderID: "nope", ProductID: testTargetProduct, Currency: "USD"}},
		{"bad productId", CustomerPlanChangeParams{OriginOrderID: testOriginOrderID, ProductID: "nope", Currency: "USD"}},
		{"bad currency", CustomerPlanChangeParams{OriginOrderID: testOriginOrderID, ProductID: testTargetProduct, Currency: "usd"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := customer.CreatePlanChangeSession(context.Background(), tc.params); err == nil {
				t.Fatal("expected an SDK validation error")
			}
		})
	}
	if len(srv.requests()) != 0 {
		t.Fatalf("expected no request to be sent, got %d", len(srv.requests()))
	}
}

func TestPlanChange_IdempotencyKeyIsOptInAndScoped(t *testing.T) {
	client, _, server := newSignedTestClient(t)
	server.respond = changeSessionResponder

	// Default: no key on the wire.
	if _, err := client.Checkout.CreatePlanChangeSession(context.Background(), planChangeParams()); err != nil {
		t.Fatalf("plan change: %v", err)
	}
	if got := server.requests()[0].Headers.Get("X-Idempotency-Key"); got != "" {
		t.Errorf("expected no key by default, got %q", got)
	}

	// Explicit: sent verbatim.
	const key = "MER_plan-change-2026-00891"
	if _, err := client.Checkout.CreatePlanChangeSession(context.Background(), planChangeParams(), WithIdempotencyKey(key)); err != nil {
		t.Fatalf("plan change with key: %v", err)
	}
	if got := server.requests()[1].Headers.Get("X-Idempotency-Key"); got != key {
		t.Errorf("X-Idempotency-Key = %q want %q", got, key)
	}

	// Authenticated: the key addresses create-session only, never the token call.
	if _, err := client.Checkout.Authenticated.CreatePlanChange(context.Background(), AuthenticatedPlanChangeParams{
		CreatePlanChangeSessionParams: planChangeParams(),
		BuyerIdentity:                 "user-1",
	}, WithIdempotencyKey("MER_plan-change-2026-00892")); err != nil {
		t.Fatalf("authenticated plan change: %v", err)
	}
	for _, req := range server.requests()[2:] {
		got := req.Headers.Get("X-Idempotency-Key")
		switch {
		case strings.HasSuffix(req.Path, "/create-session"):
			if got != "MER_plan-change-2026-00892" {
				t.Errorf("create-session key = %q", got)
			}
		case strings.HasSuffix(req.Path, "/issue-session-token"):
			if got != "" {
				t.Errorf("token call must not reuse the key, got %q", got)
			}
		}
	}
}

func TestCustomerSession_IdempotencyKeyIsOptIn(t *testing.T) {
	_, customer, srv := newCustomerTestClient(t)
	srv.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{
			"sessionId":   "ses_self_service",
			"checkoutUrl": "https://pancake.example/store/my-store/change/ses_self_service",
			"expiresAt":   "2026-05-13T00:45:00Z",
		}}
	}

	p := CustomerPlanChangeParams{OriginOrderID: testOriginOrderID, ProductID: testTargetProduct, Currency: "USD"}
	if _, err := customer.CreatePlanChangeSession(context.Background(), p); err != nil {
		t.Fatalf("plan change: %v", err)
	}
	if got := srv.requests()[0].Headers.Get("X-Idempotency-Key"); got != "" {
		t.Errorf("expected no key by default, got %q", got)
	}

	const key = "MER_self-service-2026-00893"
	if _, err := customer.CreatePlanChangeSession(context.Background(), p, WithIdempotencyKey(key)); err != nil {
		t.Fatalf("plan change with key: %v", err)
	}
	if got := srv.requests()[1].Headers.Get("X-Idempotency-Key"); got != key {
		t.Errorf("X-Idempotency-Key = %q want %q", got, key)
	}
}
