package payment

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
)

// Inspect Stripe's actual outgoing payload for both purchase entry points.
// Canceling checkout must return to plans on the single purchase page.
func TestCheckout_CancelReturnsToPurchasePage(t *testing.T) {
	s, ctx := openStore(t)
	defer s.Close()
	plan := seedPlan(t, s, ctx, "purchase-return", "perpetual")
	plan.StripePriceID = "price_purchase_return_" + plan.Slug
	plan.Active = true
	if err := s.UpdatePlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	const baseURL = "https://license.accessible.org"
	var calls atomic.Int32
	stubStripe(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/prices/"+plan.StripePriceID:
			fmt.Fprintf(w, `{"id":%q,"object":"price","type":"one_time"}`, plan.StripePriceID)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if got := r.Form.Get("cancel_url"); got != baseURL+"/#store-plans" {
				t.Errorf("cancel_url=%q, want purchase page plans", got)
			}
			if got := r.Form.Get("success_url"); got != baseURL+"/checkout/success?session_id={CHECKOUT_SESSION_ID}" {
				t.Errorf("success_url=%q, lost payment verification", got)
			}
			if got := r.Form.Get("line_items[0][price]"); got != plan.StripePriceID {
				t.Errorf("checkout changed the selected price: %q", got)
			}
			calls.Add(1)
			fmt.Fprint(w, `{"id":"cs_test_return","object":"checkout.session","url":"https://checkout.stripe.com/test-return"}`)
		default:
			t.Errorf("unexpected Stripe request: %s %s", r.Method, r.URL.Path)
			http.Error(w, `{"error":{"message":"unexpected request"}}`, http.StatusNotFound)
		}
	})
	gin.SetMode(gin.TestMode)
	h := &StripeHandler{Store: s, BaseURL: baseURL}
	r := gin.New()
	r.POST("/checkout", h.CreateCheckoutSession)
	r.GET("/pay/:checkout_id", h.CheckoutByPlan)
	request := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(fmt.Sprintf(`{"price_id":%q}`, plan.StripePriceID)))
	request.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)
	if w.Code != http.StatusOK {
		t.Fatalf("JSON checkout failed: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/pay/"+plan.CheckoutID, nil))
	if w.Code != http.StatusTemporaryRedirect || w.Header().Get("Location") != "https://checkout.stripe.com/test-return" {
		t.Fatalf("plan checkout failed: %d %s", w.Code, w.Body.String())
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("expected both purchase flows to reach Stripe, got %d", got)
	}
}
