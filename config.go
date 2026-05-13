package pancake

import "net/http"

// DefaultBaseURL is the production API base URL applied when Config.BaseURL
// is empty.
const DefaultBaseURL = "https://api.waffo.ai"

// Config configures a Client.
type Config struct {
	// MerchantID in MER_{base62} format (sent as X-Merchant-Id).
	MerchantID string
	// PrivateKey is the RSA private key in PEM format. The SDK normalizes
	// common input variants (literal "\n", Windows line endings, raw base64,
	// PKCS#1, PKCS#8) before use.
	PrivateKey string
	// BaseURL overrides the API host. Defaults to DefaultBaseURL.
	BaseURL string
	// HTTPClient overrides the underlying *http.Client. Defaults to
	// http.DefaultClient.
	HTTPClient *http.Client
	// WebhookPublicKey configures keys for webhook signature verification.
	// When unset, built-in test/prod keys are used.
	WebhookPublicKey WebhookPublicKeys
}
