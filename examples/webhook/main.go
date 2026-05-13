// Webhook example: HTTP handler that verifies an inbound Waffo Pancake event.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/waffo-com/waffo-pancake-sdk-go"
)

func main() {
	http.HandleFunc("/webhooks", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		sig := r.Header.Get("X-Waffo-Signature")

		event, err := pancake.VerifyWebhook(string(body), sig, nil)
		if err != nil {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		// Respond immediately and process asynchronously.
		w.WriteHeader(http.StatusOK)
		go handleEvent(event)
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleEvent(event *pancake.WebhookEvent) {
	switch pancake.WebhookEventType(event.EventType) {
	case pancake.WebhookEventTypeOrderCompleted:
		var data pancake.WebhookEventData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			log.Printf("decode event data: %v", err)
			return
		}
		total := data.Amount
		if data.Total != nil {
			total = *data.Total
		}
		log.Printf("order %s completed: %s %s", data.OrderID, total, data.Currency)
	case pancake.WebhookEventTypeRefundSucceeded:
		log.Printf("refund succeeded for store %s", event.StoreID)
	}
}
