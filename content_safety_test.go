package pancake

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestContentSafety_ScanPrompt_Allow(t *testing.T) {
	ctx := context.Background()
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{
			"action":            "allow",
			"reasonCode":        "allowed",
			"matchedCategories": []any{},
			"requestId":         "req_123",
			"semanticStatus":    "disabled",
		}}
	}

	res, err := client.ContentSafety.ScanPrompt(ctx, ScanPromptParams{Prompt: "a cat riding a bike"})
	if err != nil {
		t.Fatalf("scan prompt: %v", err)
	}
	if res.Action != ScanActionAllow {
		t.Errorf("action = %q, want allow", res.Action)
	}
	if res.ReasonCode != ScanReasonCodeAllowed {
		t.Errorf("reasonCode = %q, want allowed", res.ReasonCode)
	}
	if res.RequestID != "req_123" {
		t.Errorf("requestId = %q, want req_123", res.RequestID)
	}
	if res.SemanticStatus != ScanSemanticStatusDisabled {
		t.Errorf("semanticStatus = %q, want disabled", res.SemanticStatus)
	}
	if len(res.MatchedCategories) != 0 {
		t.Errorf("matchedCategories = %v, want empty", res.MatchedCategories)
	}

	reqs := server.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Path != "/v1/actions/verification/scan-prompt" {
		t.Errorf("path = %q, want /v1/actions/verification/scan-prompt", reqs[0].Path)
	}
}

func TestContentSafety_ScanPrompt_LocaleAndSemantic(t *testing.T) {
	ctx := context.Background()
	client, _, server := newSignedTestClient(t)
	server.respond = func(_ recordedRequest) (int, any) {
		return 200, map[string]any{"data": map[string]any{
			"action":            "block",
			"reasonCode":        "restricted_content",
			"matchedCategories": []any{"adult_nsfw"},
			"requestId":         "req_456",
			"semanticStatus":    "scored",
		}}
	}

	res, err := client.ContentSafety.ScanPrompt(ctx, ScanPromptParams{
		Prompt:   "some prompt",
		Locale:   "ja",
		Semantic: ScanSemanticModeEnforce,
	})
	if err != nil {
		t.Fatalf("scan prompt: %v", err)
	}
	if res.Action != ScanActionBlock {
		t.Errorf("action = %q, want block", res.Action)
	}
	if len(res.MatchedCategories) != 1 || res.MatchedCategories[0] != ScanPolicyCategoryAdultNsfw {
		t.Errorf("matchedCategories = %v, want [adult_nsfw]", res.MatchedCategories)
	}

	reqs := server.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Path != "/v1/actions/verification/scan-prompt" {
		t.Errorf("path = %q, want /v1/actions/verification/scan-prompt", reqs[0].Path)
	}

	var body map[string]any
	if err := json.Unmarshal(reqs[0].Body, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["locale"] != "ja" {
		t.Errorf("body.locale = %v, want ja", body["locale"])
	}
	if body["semantic"] != "enforce" {
		t.Errorf("body.semantic = %v, want enforce", body["semantic"])
	}
	if body["prompt"] != "some prompt" {
		t.Errorf("body.prompt = %v, want some prompt", body["prompt"])
	}
}

func TestContentSafety_ScanPrompt_EmptyPrompt(t *testing.T) {
	ctx := context.Background()
	client, _, _ := newSignedTestClient(t)

	_, err := client.ContentSafety.ScanPrompt(ctx, ScanPromptParams{})
	if err == nil {
		t.Fatal("expected validation error for empty prompt, got nil")
	}
	var perr *Error
	if !errors.As(err, &perr) {
		t.Fatalf("expected *pancake.Error, got %T", err)
	}
	if len(perr.Errors) == 0 || perr.Errors[0].Layer != ErrorLayerSDK {
		t.Errorf("expected SDK-layer validation error, got %+v", perr.Errors)
	}
}
