package payment

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/tabloy/keygate/internal/service"
	"github.com/tabloy/keygate/internal/store"
)

// A paid checkout must durably record both the license and its delivery email.
// Force a queue write failure, then replay the same signed Stripe event.
func TestWebhook_CheckoutConfirmationSurvivesQueueFailure(t *testing.T) {
	st, ctx := openStore(t)
	plan := seedPlan(t, st, ctx, "confirmation", "subscription")
	suffix := strings.ReplaceAll(store.NewID(), "-", "")
	email := "confirmation-" + suffix + "@example.test"
	constraint := "confirmation_queue_" + suffix
	t.Cleanup(func() {
		_, _ = st.DB.Exec("ALTER TABLE email_queue DROP CONSTRAINT IF EXISTS " + constraint)
		st.Close()
	})
	if _, err := st.DB.Exec(fmt.Sprintf("ALTER TABLE email_queue ADD CONSTRAINT %s CHECK (to_addr <> '%s') NOT VALID", constraint, email)); err != nil {
		t.Fatal(err)
	}
	downloadURL := "https://license.example.test/portal"
	if _, err := st.DB.NewRaw("UPDATE products SET download_url = ? WHERE id = ?", downloadURL, plan.ProductID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	mailer := service.NewEmailService("", "587", "", "", "", slog.New(slog.NewTextHandler(io.Discard, nil)), st)
	mailer.SetBaseURL("https://license.example.test")
	h := &StripeHandler{Store: st, Email: mailer}
	secret := "whsec_confirmation_fixture"
	h.SetWebhookSecret(secret)
	eventID, sessionID := "evt_confirmation_"+suffix, "cs_confirmation_"+suffix
	checkout := map[string]any{
		"id": sessionID, "object": "checkout.session", "payment_status": "paid",
		"customer_details": map[string]any{"email": email},
		"metadata":         map[string]string{"plan_id": plan.ID, metaLicenseType: plan.LicenseType},
	}
	if code, _ := signedWebhook(t, h, "whsec_wrong_fixture", eventID, "checkout.session.completed", checkout); code != http.StatusBadRequest {
		t.Fatalf("invalid signature accepted: %d", code)
	}
	code, out := signedWebhook(t, h, secret, eventID, "checkout.session.completed", checkout)
	if code != http.StatusInternalServerError || out["retry"] != true {
		t.Fatalf("queue failure must ask Stripe to retry, got %d %v", code, out)
	}
	if licenses, err := st.ListLicensesByEmail(ctx, email); err != nil || len(licenses) != 0 {
		t.Fatalf("license must roll back with its email: count=%d err=%v", len(licenses), err)
	}
	for _, claim := range []struct{ provider, id string }{
		{sessionClaimProvider, sessionID}, {fulfilledSessionProvider, sessionID},
		{processedEventClaimProvider, eventID}, {processedEventDoneProvider, eventID},
	} {
		if st.IsEventProcessed(ctx, claim.provider, claim.id) {
			t.Fatalf("failed delivery left a claim or completion marker: %s", claim.provider)
		}
	}
	if _, err := st.DB.Exec("ALTER TABLE email_queue DROP CONSTRAINT " + constraint); err != nil {
		t.Fatal(err)
	}
	code, out = signedWebhook(t, h, secret, eventID, "checkout.session.completed", checkout)
	if code != http.StatusOK || out["received"] != true || out["skipped"] == true {
		t.Fatalf("retry must fulfill checkout, got %d %v", code, out)
	}
	code, out = signedWebhook(t, h, secret, eventID, "checkout.session.completed", checkout)
	if code != http.StatusOK || out["skipped"] != true {
		t.Fatalf("duplicate event must be skipped, got %d %v", code, out)
	}
	if ok, err := h.fulfillCheckout(ctx, email, "", "", "", map[string]string{"plan_id": plan.ID, "session_id": sessionID}, "verify"); !ok || err != nil {
		t.Fatalf("success-page fallback must find existing fulfillment: %t %v", ok, err)
	}
	licenses, err := st.ListLicensesByEmail(ctx, email)
	if err != nil || len(licenses) != 1 {
		t.Fatalf("want one license after retry and replays, got %d %v", len(licenses), err)
	}
	var subscriptions, emails int
	if err := st.DB.NewRaw("SELECT count(*) FROM subscriptions WHERE license_id = ?", licenses[0].ID).Scan(ctx, &subscriptions); err != nil || subscriptions != 1 {
		t.Fatalf("want one subscription, got %d %v", subscriptions, err)
	}
	if err := st.DB.NewRaw("SELECT count(*) FROM email_queue WHERE to_addr = ?", email).Scan(ctx, &emails); err != nil || emails != 1 {
		t.Fatalf("want one confirmation email, got %d %v", emails, err)
	}
	var mail store.QueuedEmail
	if err := st.DB.NewRaw("SELECT to_addr, subject, body, status FROM email_queue WHERE to_addr = ?", email).Scan(ctx, &mail); err != nil {
		t.Fatal(err)
	}
	if mail.Status != "pending" || !strings.Contains(mail.Subject, "confirmation") {
		t.Fatalf("confirmation was not queued with the product name: %q %q", mail.Status, mail.Subject)
	}
	for label, expected := range map[string]string{"license key": st.DecryptLicenseKey(licenses[0]), "plan": plan.Name, "portal/download link": downloadURL} {
		if !strings.Contains(mail.Body, expected) {
			t.Errorf("confirmation email is missing %s", label)
		}
	}
}
