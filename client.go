package pancake

import (
	"net/http"

	"github.com/waffo-com/waffo-pancake-sdk-go/internal/signing"
)

// Client is the main Waffo Pancake SDK entry point. Construct it with [New]
// and access resource namespaces through its exported fields. Methods that
// perform I/O take a context.Context first parameter.
type Client struct {
	// Auth issues buyer session tokens.
	Auth *AuthResource
	// Stores manages stores.
	Stores *StoresResource
	// StoreMerchants manages store membership (coming soon — endpoints return 501).
	StoreMerchants *StoreMerchantsResource
	// OnetimeProducts manages one-time products.
	OnetimeProducts *OnetimeProductsResource
	// SubscriptionProducts manages subscription products.
	SubscriptionProducts *SubscriptionProductsResource
	// SubscriptionProductGroups manages subscription groups.
	SubscriptionProductGroups *SubscriptionProductGroupsResource
	// Orders manages orders.
	Orders *OrdersResource
	// Checkout creates checkout sessions.
	Checkout *CheckoutResource
	// GraphQL runs GraphQL queries at merchant scope.
	GraphQL *GraphQLResource
	// Webhooks manages webhook endpoints and verifies inbound signatures.
	Webhooks *WebhooksResource

	http   *httpClient
	config Config
}

// New constructs a Client. It validates MerchantID format, normalizes the
// RSA private key, and wires every resource namespace.
func New(c Config) (*Client, error) {
	if err := validateShortID("merchantId", c.MerchantID, "MER"); err != nil {
		return nil, err
	}
	if c.PrivateKey == "" {
		return nil, newSDKError("Missing required field: privateKey")
	}
	normalized, err := signing.NormalizePrivateKey(c.PrivateKey)
	if err != nil {
		return nil, newSDKError("%s", err.Error())
	}
	parsed, err := signing.ParsePrivateKey(normalized)
	if err != nil {
		return nil, newSDKError("%s", err.Error())
	}

	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	httpC := c.HTTPClient
	if httpC == nil {
		httpC = http.DefaultClient
	}

	h := newHTTPClient(c.MerchantID, parsed, baseURL, httpC)

	cl := &Client{http: h, config: c}
	cl.Auth = &AuthResource{http: h}
	cl.Stores = &StoresResource{http: h}
	cl.StoreMerchants = &StoreMerchantsResource{http: h}
	cl.OnetimeProducts = &OnetimeProductsResource{http: h}
	cl.SubscriptionProducts = &SubscriptionProductsResource{http: h}
	cl.SubscriptionProductGroups = &SubscriptionProductGroupsResource{http: h}
	cl.Orders = &OrdersResource{http: h}
	cl.Checkout = newCheckoutResource(h)
	cl.GraphQL = &GraphQLResource{http: h}
	cl.Webhooks = &WebhooksResource{http: h, publicKeys: c.WebhookPublicKey}
	return cl, nil
}

// Buyer returns a BuyerSession backed by the given session token. The token
// is issued by Auth.IssueSessionToken and is sent as a Bearer Authorization
// header on every buyer request. No I/O is performed by this call.
func (c *Client) Buyer(token string) *BuyerSession {
	baseURL := c.config.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	httpC := c.config.HTTPClient
	if httpC == nil {
		httpC = http.DefaultClient
	}
	bh := newBuyerHTTPClient(token, baseURL, httpC)
	return newBuyerSession(bh)
}
