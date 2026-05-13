// Basic example: create a client and a store, then create a one-time product.
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

	storeRes, err := client.Stores.Create(ctx, pancake.CreateStoreParams{Name: "Example Store"})
	if err != nil {
		log.Fatalf("create store: %v", err)
	}
	fmt.Printf("store: %s\n", storeRes.Store.ID)

	prodRes, err := client.OnetimeProducts.Create(ctx, pancake.CreateOnetimeProductParams{
		StoreID: storeRes.Store.ID,
		Name:    "E-Book",
		Prices: pancake.Prices{
			"USD": {Amount: "29.00", TaxCategory: pancake.TaxCategoryDigitalGoods},
		},
	})
	if err != nil {
		log.Fatalf("create product: %v", err)
	}
	fmt.Printf("product: %s\n", prodRes.Product.ID)
}
