// Checkout example: create an authenticated checkout session.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/waffo-com/waffo-pancake-sdk-go"
)

func main() {
	client, err := pancake.New(pancake.Config{
		MerchantID: os.Getenv("WAFFO_MERCHANT_ID"),
		PrivateKey: os.Getenv("WAFFO_PRIVATE_KEY"),
	})
	if err != nil {
		log.Fatalf("init client: %v", err)
	}

	ctx := context.Background()

	res, err := client.Checkout.Authenticated.Create(ctx, pancake.AuthenticatedCheckoutParams{
		CreateCheckoutSessionParams: pancake.CreateCheckoutSessionParams{
			ProductID:  os.Getenv("WAFFO_PRODUCT_ID"),
			Currency:   "USD",
			BuyerEmail: pancake.Ptr("customer@example.com"),
		},
		BuyerIdentity: "user-123",
	})
	if err != nil {
		log.Fatalf("create checkout: %v", err)
	}

	fmt.Println("Redirect the buyer to:")
	fmt.Println(res.CheckoutURL)
}
