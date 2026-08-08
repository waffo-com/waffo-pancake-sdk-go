package pancake

import (
	"net/http"

	"github.com/waffo-com/waffo-pancake-sdk-go/internal/signing"
)

// Client is the main Waffo Pancake SDK entry point. Construct it with [New]
// and access resource namespaces through its exported fields. Methods that
// perform I/O take a context.Context first parameter.
type Client struct {
	// Auth issues customer session tokens.
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
	// ContentSafety scans user prompts before AIGC generation.
	ContentSafety *ContentSafetyResource

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
	cl.ContentSafety = &ContentSafetyResource{http: h}
	return cl, nil
}

// Customer returns a CustomerSession backed by the given session token, running
// in Config.Environment. The token is issued by Auth.IssueSessionToken and is
// sent as a Bearer Authorization header, alongside X-Environment, on every
// customer request. No I/O is performed by this call.
//
// Session tokens expire 5 minutes after issuance, so issue one right before use
// rather than caching it.
//
// Config.Environment must be set; otherwise the first session method returns an
// SDK-layer *Error. Use CustomerWithEnvironment to override it per session.
//
// Example:
//
//	res, err := client.Auth.IssueSessionToken(ctx, pancake.IssueSessionTokenParams{
//		StoreID:       "STO_...",
//		BuyerIdentity: "customer@example.com",
//	})
//	session := client.Customer(res.Token)
//	out, err := session.CancelSubscription(ctx, pancake.CancelSubscriptionParams{OrderID: "ORD_..."})
func (c *Client) Customer(token string) *CustomerSession {
	return c.CustomerWithEnvironment(token, c.config.Environment)
}

// CustomerWithEnvironment returns a CustomerSession backed by the given session
// token, running in the named environment instead of Config.Environment. No I/O
// is performed by this call.
//
// Example:
//
//	session := client.CustomerWithEnvironment(res.Token, pancake.EnvironmentTest)
func (c *Client) CustomerWithEnvironment(token string, environment Environment) *CustomerSession {
	baseURL := c.config.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	httpC := c.config.HTTPClient
	if httpC == nil {
		httpC = http.DefaultClient
	}
	ch := newCustomerHTTPClient(token, environment, baseURL, httpC)
	return newCustomerSession(ch)
}

// Buyer returns a CustomerSession backed by the given session token.
//
// Deprecated: Use Customer instead.
func (c *Client) Buyer(token string) *CustomerSession {
	return c.Customer(token)
}
