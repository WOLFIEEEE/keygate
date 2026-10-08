// Package merchant provisions a configured standalone WordPress store. It does
// not change upstream installations unless BOOTSTRAP_OWNER_EMAIL is supplied.
package merchant

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"strings"

	"github.com/stripe/stripe-go/v82"
	portalconfig "github.com/stripe/stripe-go/v82/billingportal/configuration"
	stripeprice "github.com/stripe/stripe-go/v82/price"
	stripeproduct "github.com/stripe/stripe-go/v82/product"
	"github.com/tabloy/keygate/internal/config"
	"github.com/tabloy/keygate/internal/model"
	"github.com/tabloy/keygate/internal/service"
	"github.com/tabloy/keygate/internal/store"
	"github.com/uptrace/bun"
)

type Plan struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Sites    int    `json:"sites"`  // 0 explicitly means unlimited
	Amount   int64  `json:"amount"` // smallest currency unit, not a display price
	Currency string `json:"currency"`
	Interval string `json:"interval"` // month, year, or lifetime
	PriceID  string `json:"price_id,omitempty"`
}

type Settings struct {
	OwnerEmail   string
	OwnerName    string
	ProductName  string
	ProductSlug  string
	SiteName     string
	PublisherKey string
	Plans        []Plan
}

func FromEnvironment() (*Settings, error) {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_OWNER_EMAIL")))
	if email == "" {
		return nil, nil
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return nil, fmt.Errorf("BOOTSTRAP_OWNER_EMAIL must be one valid email address")
	}
	s := &Settings{OwnerEmail: email, OwnerName: env("BOOTSTRAP_OWNER_NAME", "Accessible.org"), ProductName: env("BOOTSTRAP_PRODUCT_NAME", "Accessible Forms Pro"), ProductSlug: env("BOOTSTRAP_PRODUCT_SLUG", "accessible-forms-pro"), SiteName: env("BOOTSTRAP_SITE_NAME", "Accessible.org Store"), PublisherKey: os.Getenv("RELEASE_PUBLISH_KEY")}
	if !regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`).MatchString(s.ProductSlug) {
		return nil, fmt.Errorf("BOOTSTRAP_PRODUCT_SLUG is invalid")
	}
	raw := os.Getenv("BOOTSTRAP_PLANS_JSON")
	if raw != "" {
		d := json.NewDecoder(strings.NewReader(raw))
		d.DisallowUnknownFields()
		if err := d.Decode(&s.Plans); err != nil {
			return nil, fmt.Errorf("BOOTSTRAP_PLANS_JSON is invalid: %w", err)
		}
		if err := d.Decode(new(any)); err != io.EOF {
			return nil, fmt.Errorf("BOOTSTRAP_PLANS_JSON must contain one JSON array")
		}
	}
	if err := validatePlans(s.Plans); err != nil {
		return nil, err
	}
	if s.PublisherKey != "" && (!strings.HasPrefix(s.PublisherKey, "kg_live_") || len(s.PublisherKey) < 40 || len(s.PublisherKey) > 256) {
		return nil, fmt.Errorf("RELEASE_PUBLISH_KEY must have 32 to 256 characters")
	}
	return s, nil
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func validatePlans(plans []Plan) error {
	if len(plans) > 20 {
		return fmt.Errorf("at most 20 bootstrap plans are supported")
	}
	seen := map[string]bool{}
	for _, p := range plans {
		if !regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`).MatchString(p.Slug) || seen[p.Slug] || len(p.Name) == 0 || len(p.Name) > 128 || p.Sites < 0 || p.Sites > 1000000 {
			return fmt.Errorf("each plan needs a unique slug, a name, and a nonnegative site limit")
		}
		seen[p.Slug] = true
		if p.Interval != "month" && p.Interval != "year" && p.Interval != "lifetime" {
			return fmt.Errorf("plan %s interval must be month, year or lifetime", p.Slug)
		}
		if !regexp.MustCompile(`^[a-z]{3}$`).MatchString(p.Currency) || p.Amount < 1 || p.Amount > 99999999 {
			return fmt.Errorf("plan %s requires a currency and a positive amount in its smallest unit", p.Slug)
		}
	}
	return nil
}

// Provision uses one session advisory lock across database and Stripe writes.
// Stable provider IDs/lookup keys recover resources created before a crash.
// It never promotes an additional owner on an already initialized database.
func Provision(ctx context.Context, st *store.Store, signer *service.ReleaseSigningService, cfg *config.Config, s *Settings) error {
	if s == nil {
		return nil
	}
	return st.WithAdvisoryLock(ctx, 7367618, func(ctx context.Context) error {
		if err := validatePlans(s.Plans); err != nil {
			return err
		}
		var owners []model.User
		if err := st.DB.NewSelect().Model(&owners).Where("role = 'owner'").Scan(ctx); err != nil {
			return err
		}
		if len(owners) > 0 {
			matches := false
			for _, u := range owners {
				if u.Email == s.OwnerEmail {
					matches = true
				}
			}
			if !matches {
				return fmt.Errorf("bootstrap owner differs from the existing owner; use the dashboard to transfer ownership")
			}
		}
		prod, err := st.FindProductBySlug(ctx, s.ProductSlug)
		if errors.Is(err, sql.ErrNoRows) {
			prod = &model.Product{Name: s.ProductName, Slug: s.ProductSlug, Type: model.ProductTypeDesktop, DownloadURL: strings.TrimRight(cfg.BaseURL, "/") + "/portal", RequireSigning: true, FeedLicenseRequired: true}
			if err = st.CreateProduct(ctx, prod); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if prod.Type != model.ProductTypeDesktop || !prod.FeedLicenseRequired || !prod.RequireSigning {
			return fmt.Errorf("bootstrap product must be desktop with license-gated feeds and required signing")
		}
		if _, err = st.FindActiveSigningKey(ctx, prod.ID); errors.Is(err, store.ErrActiveSigningKeyMissing) {
			if _, err = signer.GenerateForProduct(ctx, prod.ID); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if err = st.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if len(owners) == 0 {
				_, err := tx.NewRaw("INSERT INTO users (id,email,name,role) VALUES (?,?,?,'owner') ON CONFLICT (email) DO UPDATE SET role='owner',name=EXCLUDED.name", store.NewID(), s.OwnerEmail, s.OwnerName).Exec(ctx)
				if err != nil {
					return err
				}
			}
			for k, v := range map[string]string{"setup_complete": "true", "site_name": s.SiteName, "signup_mode": "licensed_only"} {
				if _, err := tx.NewRaw("INSERT INTO settings (key,value) VALUES (?,?) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value", k, v).Exec(ctx); err != nil {
					return err
				}
			}
			if s.PublisherKey != "" {
				key := &model.APIKey{ID: store.NewID(), ProductID: prod.ID, Name: "Configured release publisher", KeyHash: store.HashAPIKey(s.PublisherKey), Prefix: s.PublisherKey[:8], Scopes: []string{model.ScopeReleasesWrite}}
				if _, err := tx.NewInsert().Model(key).On("CONFLICT (key_hash) DO NOTHING").Exec(ctx); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		// A catalog fingerprint lets restarts survive a temporary Stripe outage.
		// It is never used to skip owner/product/signing checks above.
		encoded, _ := json.Marshal(struct {
			Plans          []Plan
			Base           string
			Live           bool
			KeyFingerprint string
		}{s.Plans, cfg.BaseURL, cfg.StripeLivemode, hex.EncodeToString(sumURL(cfg.StripeSecretKey))})
		sum := sha256.Sum256(encoded)
		fingerprint := hex.EncodeToString(sum[:])
		old, _ := st.GetSetting(ctx, "merchant_catalog_fingerprint")
		if old == fingerprint {
			return nil
		}
		if len(s.Plans) == 0 || cfg.StripeSecretKey == "" {
			return nil
		} // configuration pending, checkout stays closed
		mode := "test"
		if cfg.StripeLivemode {
			mode = "live"
		}
		previousMode, _ := st.GetSetting(ctx, "merchant_stripe_mode")
		if previousMode != "" && previousMode != mode {
			return fmt.Errorf("test and live stores must use separate databases; this database is already in %s mode", previousMode)
		}
		account := new(stripe.Account)
		if err := stripe.GetBackend(stripe.APIBackend).Call(http.MethodGet, "/v1/account", cfg.StripeSecretKey, &stripe.AccountParams{Params: stripe.Params{Context: ctx}}, account); err != nil {
			return fmt.Errorf("Stripe account verification failed: %w", err)
		}
		previousAccount, _ := st.GetSetting(ctx, "merchant_stripe_account_id")
		if previousAccount != "" && previousAccount != account.ID {
			return fmt.Errorf("this database belongs to another Stripe account; use a separate database")
		}
		if cfg.StripeLivemode && !account.ChargesEnabled {
			return fmt.Errorf("activate this Stripe account for live payments before starting checkout")
		}
		if err := st.SetSetting(ctx, "merchant_stripe_account_id", account.ID); err != nil {
			return err
		}
		productID := "accessible_" + hex.EncodeToString(sumURL(cfg.BaseURL)[:8]) + "_" + s.ProductSlug
		if len(productID) > 100 {
			productID = productID[:100]
		}
		remote, err := stripeproduct.Get(productID, &stripe.ProductParams{Params: stripe.Params{Context: ctx}})
		if err != nil {
			var se *stripe.Error
			if !errors.As(err, &se) || se.HTTPStatusCode != 404 {
				return err
			}
			params := &stripe.ProductParams{ID: stripe.String(productID), Name: stripe.String(s.ProductName), Params: stripe.Params{Context: ctx}}
			params.Metadata = map[string]string{"keygate_base_url": cfg.BaseURL, "keygate_product_slug": s.ProductSlug}
			params.SetIdempotencyKey("merchant-product-" + productID)
			remote, err = stripeproduct.New(params)
			if err != nil {
				return err
			}
		}
		if remote.Metadata["keygate_base_url"] != cfg.BaseURL {
			return fmt.Errorf("Stripe product belongs to another store")
		}
		for i, p := range s.Plans {
			price, err := ensurePrice(ctx, remote.ID, p, cfg.StripeLivemode)
			if err != nil {
				return fmt.Errorf("plan %s: %w", p.Slug, err)
			}
			typ, interval := "subscription", p.Interval
			if interval == "lifetime" {
				typ, interval = "perpetual", ""
			}
			plan := &model.Plan{ID: store.NewID(), ProductID: prod.ID, Slug: p.Slug, Name: p.Name, CheckoutID: store.ShortID(), LicenseType: typ, LicenseModel: "standard", BillingInterval: interval, MaxActivations: p.Sites, GraceDays: 7, StripePriceID: price.ID, Active: true, SortOrder: i}
			// Never change a sold license's commercial type. Existing customers retain
			// their old Stripe price; new checkouts use the new immutable price.
			existing := new(model.Plan)
			err = st.DB.NewSelect().Model(existing).Where("product_id=? AND slug=?", prod.ID, p.Slug).Scan(ctx)
			if err == nil && (existing.StripePriceID != price.ID || existing.MaxActivations != p.Sites || existing.BillingInterval != interval) {
				var sold int
				if err := st.DB.NewRaw("SELECT count(*) FROM licenses WHERE plan_id=?", existing.ID).Scan(ctx, &sold); err != nil {
					return err
				}
				if sold > 0 {
					return fmt.Errorf("plan %s has existing customers; create a new plan slug for changed prices or site limits", p.Slug)
				}
			}
			if err == nil && existing.LicenseType != typ {
				return fmt.Errorf("plan %s was already sold with another license type; add a new plan slug", p.Slug)
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			_, err = st.DB.NewInsert().Model(plan).On("CONFLICT (product_id, slug) DO UPDATE").Set("name=EXCLUDED.name").Set("stripe_price_id=EXCLUDED.stripe_price_id").Set("billing_interval=EXCLUDED.billing_interval").Set("max_activations=EXCLUDED.max_activations").Set("sort_order=EXCLUDED.sort_order").Set("active=true").Exec(ctx)
			if err != nil {
				return err
			}
		}
		activeSlugs := []string{}
		for _, plan := range s.Plans {
			activeSlugs = append(activeSlugs, plan.Slug)
		}
		if _, err := st.DB.NewUpdate().Model((*model.Plan)(nil)).Set("active=false").Where("product_id=?", prod.ID).Where("slug NOT IN (?)", bun.List(activeSlugs)).Exec(ctx); err != nil {
			return err
		}
		if err := ensurePortal(ctx, st, cfg.BaseURL, fingerprint); err != nil {
			return err
		}
		if err := st.SetSetting(ctx, "merchant_stripe_mode", mode); err != nil {
			return err
		}
		return st.SetSetting(ctx, "merchant_catalog_fingerprint", fingerprint)
	})
}
func sumURL(value string) []byte { sum := sha256.Sum256([]byte(value)); return sum[:] }

func ensurePrice(ctx context.Context, product string, p Plan, live bool) (*stripe.Price, error) {
	var result *stripe.Price
	var err error
	if p.PriceID != "" {
		result, err = stripeprice.Get(p.PriceID, &stripe.PriceParams{Params: stripe.Params{Context: ctx}})
	} else {
		raw := fmt.Sprintf("%s:%s:%s:%d:%s", product, p.Slug, p.Currency, p.Amount, p.Interval)
		digest := sha256.Sum256([]byte(raw))
		lookup := "accessible_" + hex.EncodeToString(digest[:16])
		it := stripeprice.List(&stripe.PriceListParams{LookupKeys: []*string{stripe.String(lookup)}, ListParams: stripe.ListParams{Context: ctx}})
		if it.Next() {
			result = it.Price()
		}
		if it.Err() != nil {
			return nil, it.Err()
		}
		if result == nil {
			params := &stripe.PriceParams{Product: stripe.String(product), Currency: stripe.String(p.Currency), UnitAmount: stripe.Int64(p.Amount), LookupKey: stripe.String(lookup), Params: stripe.Params{Context: ctx}}
			if p.Interval != "lifetime" {
				params.Recurring = &stripe.PriceRecurringParams{Interval: stripe.String(p.Interval), IntervalCount: stripe.Int64(1)}
			}
			params.SetIdempotencyKey("merchant-price-" + lookup)
			result, err = stripeprice.New(params)
		}
	}
	if err != nil {
		return nil, err
	}
	if result == nil || !result.Active || result.Livemode != live || string(result.Currency) != p.Currency || result.UnitAmount != p.Amount || result.Product == nil || result.Product.ID != product {
		return nil, fmt.Errorf("Stripe price does not match the configured product, amount, currency or mode")
	}
	if p.Interval == "lifetime" {
		if result.Recurring != nil {
			return nil, fmt.Errorf("lifetime plan requires a one-time price")
		}
	} else if result.Recurring == nil || string(result.Recurring.Interval) != p.Interval || result.Recurring.IntervalCount != 1 {
		return nil, fmt.Errorf("Stripe price billing interval differs")
	}
	return result, nil
}
func ensurePortal(ctx context.Context, st *store.Store, base, fingerprint string) error {
	id, _ := st.GetSetting(ctx, "stripe_portal_configuration_id")
	params := &stripe.BillingPortalConfigurationParams{DefaultReturnURL: stripe.String(strings.TrimRight(base, "/") + "/portal"), Params: stripe.Params{Context: ctx}, Features: &stripe.BillingPortalConfigurationFeaturesParams{
		InvoiceHistory:      &stripe.BillingPortalConfigurationFeaturesInvoiceHistoryParams{Enabled: stripe.Bool(true)},
		PaymentMethodUpdate: &stripe.BillingPortalConfigurationFeaturesPaymentMethodUpdateParams{Enabled: stripe.Bool(true)},
		SubscriptionCancel:  &stripe.BillingPortalConfigurationFeaturesSubscriptionCancelParams{Enabled: stripe.Bool(true), Mode: stripe.String("at_period_end")},
	}}
	var result *stripe.BillingPortalConfiguration
	var err error
	if id != "" {
		result, err = portalconfig.Update(id, params)
	} else {
		params.SetIdempotencyKey("merchant-portal-" + fingerprint)
		result, err = portalconfig.New(params)
	}
	if err != nil {
		return err
	}
	return st.SetSetting(ctx, "stripe_portal_configuration_id", result.ID)
}
