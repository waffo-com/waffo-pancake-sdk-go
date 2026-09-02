// Package pancake is the official Go SDK for the Waffo Pancake Merchant of
// Record (MoR) payment platform.
//
// All merchant API requests are auto-signed with RSA-SHA256 and carry
// deterministic idempotency keys derived from the merchant ID, path, and body.
// Webhook verification, GraphQL queries, and customer self-service flows are
// supported out of the box. The SDK has zero external runtime dependencies —
// only the Go standard library.
//
// # Quickstart
//
//	client, err := pancake.New(pancake.Config{
//	    MerchantID: os.Getenv("WAFFO_MERCHANT_ID"), // MER_{base62}
//	    PrivateKey: os.Getenv("WAFFO_PRIVATE_KEY"), // RSA PEM
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	ctx := context.Background()
//
//	storeRes, err := client.Stores.Create(ctx, pancake.CreateStoreParams{
//	    Name: "My Store",
//	})
//
//	checkout, err := client.Checkout.Authenticated.Create(ctx, pancake.AuthenticatedCheckoutParams{
//	    CreateCheckoutSessionParams: pancake.CreateCheckoutSessionParams{
//	        ProductID: "PROD_...",
//	        Currency:  "USD",
//	    },
//	    BuyerIdentity: "customer@example.com",
//	})
//
//	event, err := pancake.VerifyWebhook(rawBody, signatureHeader, nil)
//
// Feature parity with @waffo/pancake-ts@0.20.x.
package pancake
