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
	// Environment is the environment customer sessions operate in, sent as
	// X-Environment alongside the session token. API Key requests do not need
	// it — the gateway derives their environment from the key. Session tokens
	// carry none, so the gateway requires the header and answers HTTP 400
	// without it.
	//
	// There is no default: a wrong guess would route the call to the other
	// environment. Set it here, or per session via
	// [Client.CustomerWithEnvironment].
	Environment Environment
	// HTTPClient overrides the underlying *http.Client. Defaults to
	// http.DefaultClient.
	HTTPClient *http.Client
	// WebhookPublicKey configures keys for webhook signature verification.
	// When unset, built-in test/prod keys are used.
	WebhookPublicKey WebhookPublicKeys
}
