package pancake

import "context"

// ContentSafetyResource scans user prompts for content-safety compliance
// before AIGC generation.
type ContentSafetyResource struct {
	http *httpClient
}

// ScanPrompt scans a user's text prompt for content-safety compliance before
// AIGC generation. Call this before invoking your image/video model and
// continue only when the returned Action is [ScanActionAllow].
//
// The check is stateless — prompt text is never stored. If the safety service
// is briefly unavailable, the verdict fails closed to [ScanActionReview] so an
// unmoderated prompt is never let through.
//
// Example:
//
//	verdict, err := client.ContentSafety.ScanPrompt(ctx, pancake.ScanPromptParams{
//	    Prompt: "a cat riding a bike",
//	})
//	if err != nil {
//	    // handle error
//	}
//	if verdict.Action != pancake.ScanActionAllow {
//	    // do not generate — verdict.Action is "review" or "block"
//	}
func (r *ContentSafetyResource) ScanPrompt(ctx context.Context, p ScanPromptParams) (*ScanResult, error) {
	if err := validateRequired("prompt", p.Prompt); err != nil {
		return nil, err
	}
	out, warnings, err := postAction[ScanResult](ctx, r.http, "/v1/actions/verification/scan-prompt", p, nil)
	if err != nil {
		return nil, err
	}
	out.Warnings = warnings
	return out, nil
}
