// lockBuyerEmail marshal contract: nil is omitted, and both true and false are sent,
// because the server treats a missing value as "lock when buyerEmail is set".
package pancake

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCreateCheckoutSessionParams_LockBuyerEmail_Marshal(t *testing.T) {
	cases := []struct {
		name string
		lock *bool
		want string
	}{
		{name: "nil is omitted", lock: nil, want: ""},
		{name: "false is sent", lock: Ptr(false), want: `"lockBuyerEmail":false`},
		{name: "true is sent", lock: Ptr(true), want: `"lockBuyerEmail":true`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := CreateCheckoutSessionParams{
				ProductID:      "PROD_xxx",
				Currency:       "USD",
				BuyerEmail:     Ptr("prefill@example.com"),
				LockBuyerEmail: tc.lock,
			}
			out, err := json.Marshal(p)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			got := string(out)
			if tc.want == "" {
				if strings.Contains(got, "lockBuyerEmail") {
					t.Fatalf("expected lockBuyerEmail to be omitted, got %s", got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %s in payload, got %s", tc.want, got)
			}
		})
	}
}
