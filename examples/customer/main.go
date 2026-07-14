// Customer example: issue a session token and use CustomerSession to cancel a
// subscription and submit a refund request.
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

	tok, err := client.Auth.IssueSessionToken(ctx, pancake.IssueSessionTokenParams{
		StoreID:       pancake.Ptr(os.Getenv("WAFFO_STORE_ID")),
		BuyerIdentity: "user-123",
	})
	if err != nil {
		log.Fatalf("issue session token: %v", err)
	}

	customer := client.Customer(tok.Token)

	cancelRes, err := customer.CancelSubscription(ctx, pancake.CancelSubscriptionParams{
		OrderID: os.Getenv("WAFFO_ORDER_ID"),
	})
	if err != nil {
		log.Fatalf("cancel subscription: %v", err)
	}
	fmt.Printf("subscription %s is now %s\n", cancelRes.OrderID, cancelRes.Status)
}
