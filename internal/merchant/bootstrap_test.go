package merchant

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stripe/stripe-go/v82"
	"github.com/tabloy/keygate/internal/config"
	keycrypto "github.com/tabloy/keygate/internal/crypto"
	"github.com/tabloy/keygate/internal/service"
	"github.com/tabloy/keygate/internal/storage"
	"github.com/tabloy/keygate/internal/store"
)

func merchantDB(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for database integration")
	}
	root, err := store.New(dsn)
	if err != nil {
		t.Fatal(err)
	}
	var random [8]byte
	rand.Read(random[:])
	name := "merchant_" + hex.EncodeToString(random[:])
	if _, err = root.DB.Exec("CREATE DATABASE " + name); err != nil {
		root.Close()
		t.Fatal(err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + name
	st, err := store.New(parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close(); root.DB.Exec("DROP DATABASE " + name + " WITH (FORCE)"); root.Close() })
	if err = st.RunMigrations("../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	return st
}
func TestProvisionStoreAndSignedRelease(t *testing.T) {
	st := merchantDB(t)
	ctx := context.Background()
	calls := 0
	var product map[string]any
	prices := map[string]map[string]any{}
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		var response any
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/account":
			response = map[string]any{"id": "acct_fixture", "object": "account", "charges_enabled": true}
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/v1/products/"):
			if product == nil {
				w.WriteHeader(404)
				json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"type": "invalid_request_error", "code": "resource_missing", "message": "missing product"}})
				return
			}
			response = product
		case r.Method == "POST" && r.URL.Path == "/v1/products":
			product = map[string]any{"id": r.Form.Get("id"), "object": "product", "name": r.Form.Get("name"), "metadata": map[string]string{"keygate_base_url": r.Form.Get("metadata[keygate_base_url]"), "keygate_product_slug": r.Form.Get("metadata[keygate_product_slug]")}}
			response = product
		case r.Method == "GET" && r.URL.Path == "/v1/prices":
			list := []any{}
			if p := prices[r.Form.Get("lookup_keys[0]")]; p != nil {
				list = append(list, p)
			}
			response = map[string]any{"object": "list", "data": list, "has_more": false}
		case r.Method == "POST" && r.URL.Path == "/v1/prices":
			amount, _ := strconv.ParseInt(r.Form.Get("unit_amount"), 10, 64)
			id := fmt.Sprintf("price_fixture_%d", len(prices)+1)
			p := map[string]any{"id": id, "object": "price", "active": true, "livemode": false, "currency": r.Form.Get("currency"), "unit_amount": amount, "product": map[string]any{"id": r.Form.Get("product")}}
			if interval := r.Form.Get("recurring[interval]"); interval != "" {
				p["recurring"] = map[string]any{"interval": interval, "interval_count": 1}
			}
			prices[r.Form.Get("lookup_key")] = p
			response = p
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/v1/billing_portal/configurations"):
			if r.Form.Get("features[subscription_cancel][mode]") != "at_period_end" {
				t.Error("portal cancels before paid period ends")
			}
			response = map[string]any{"id": "bpc_fixture", "object": "billing_portal.configuration", "active": true}
		default:
			t.Errorf("unexpected provider call %s %s", r.Method, r.URL.Path)
			w.WriteHeader(500)
			return
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mock.Close()
	oldBackend := stripe.GetBackend(stripe.APIBackend)
	oldKey := stripe.Key
	stripe.Key = "sk_test_fixture"
	stripe.SetBackend(stripe.APIBackend, stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{URL: stripe.String(mock.URL), HTTPClient: mock.Client(), MaxNetworkRetries: stripe.Int64(0)}))
	defer func() { stripe.SetBackend(stripe.APIBackend, oldBackend); stripe.Key = oldKey }()
	var fs *storage.Filesystem
	objects := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fs.ServeHTTP(w, r) }))
	defer objects.Close()
	fs, err := storage.NewFilesystem(t.TempDir(), objects.URL, bytes.Repeat([]byte{3}, 32), 50*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	aead, err := keycrypto.NewAESGCM(bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	signer := service.NewReleaseSigningService(service.ReleaseSigningServiceConfig{Store: st, Storage: fs, AEAD: aead, Logger: logger, MaxSignSize: 50 * 1024 * 1024})
	cfg := &config.Config{BaseURL: "https://license.example.test", StripeSecretKey: "sk_test_fixture", SMTPHost: "mail.example.test", SMTPFrom: "store@example.test", StorageLocalPath: t.TempDir(), ReleaseKeyEncryptionKey: strings.Repeat("1", 64)}
	settings := &Settings{OwnerEmail: "owner@example.test", OwnerName: "Owner", SiteName: "Test store", ProductName: "Accessible Forms Pro", ProductSlug: "accessible-forms-pro", PublisherKey: "kg_live_" + strings.Repeat("x", 64), Plans: []Plan{{Slug: "personal", Name: "Personal", Sites: 1, Amount: 9900, Currency: "usd", Interval: "year"}, {Slug: "lifetime", Name: "Lifetime", Sites: 0, Amount: 19900, Currency: "usd", Interval: "lifetime"}}}
	if err = Provision(ctx, st, signer, cfg, settings); err != nil {
		t.Fatal(err)
	}
	initialCalls := calls
	if err = Provision(ctx, st, signer, cfg, settings); err != nil {
		t.Fatal(err)
	}
	if calls != initialCalls {
		t.Fatal("restart recreated Stripe resources")
	}
	owners, err := st.CountOwners(ctx)
	if err != nil || owners != 1 {
		t.Fatalf("owners %d %v", owners, err)
	}
	if portal, _ := st.GetSetting(ctx, "stripe_portal_configuration_id"); portal != "bpc_fixture" {
		t.Fatal("portal was not configured")
	}
	_, key, err := st.FindProductByAPIKey(ctx, store.HashAPIKey(settings.PublisherKey))
	if err != nil || len(key.Scopes) != 1 || key.Scopes[0] != "releases:write" {
		t.Fatal("publisher key has wrong permissions")
	}
	blocked := Readiness(ctx, st, cfg, settings, "whsec_fixture", fs)
	if len(blocked) != 1 || blocked[0] != "published_plugin_missing" {
		t.Fatalf("unexpected pre-publication readiness %v", blocked)
	}
	prod, _ := st.FindProductBySlug(ctx, settings.ProductSlug)
	releaseService := service.NewReleaseService(service.ReleaseServiceConfig{Store: st, Storage: fs, Signer: signer, Logger: logger})
	release, err := releaseService.CreateRelease(ctx, service.CreateReleaseInput{ProductID: prod.ID, Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	var zipBytes bytes.Buffer
	zw := zip.NewWriter(&zipBytes)
	php, _ := zw.Create("accessible-forms-pro/accessible-forms-pro.php")
	php.Write([]byte("<?php\n/**\n * Plugin Name: Accessible Forms Pro\n * Version: 1.0.0\n * Requires at least: 6.5\n * Requires PHP: 8.1\n */\nconst MIN_API = 5;\n"))
	zw.Close()
	created, err := releaseService.AddArtifact(ctx, service.AddArtifactInput{ReleaseID: release.ID, Platform: "wordpress", Filename: "accessible-forms-pro-1.0.0.zip", ExpectedSize: int64(zipBytes.Len()), ContentType: "application/zip"})
	if err != nil {
		t.Fatal(err)
	}
	upload, _ := http.NewRequest("PUT", created.UploadURL, bytes.NewReader(zipBytes.Bytes()))
	response, err := objects.Client().Do(upload)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("upload %d", response.StatusCode)
	}
	artifact, err := releaseService.FinalizeArtifact(ctx, service.FinalizeArtifactInput{ArtifactID: created.Artifact.ID})
	if err != nil || artifact.WordPress == nil {
		t.Fatalf("ZIP finalization failed: %v", err)
	}
	if _, err = releaseService.Publish(ctx, release.ID); err != nil {
		t.Fatal(err)
	}
	if pending := Readiness(ctx, st, cfg, settings, "whsec_fixture", fs); len(pending) != 0 {
		t.Fatalf("completed store not ready: %v", pending)
	}
	if err := fs.Delete(ctx, artifact.FileKey); err != nil {
		t.Fatal(err)
	}
	if pending := Readiness(ctx, st, cfg, settings, "whsec_fixture", fs); len(pending) != 1 || pending[0] != "plugin_file_unavailable" {
		t.Fatalf("missing deliverable accepted: %v", pending)
	}
	other := *settings
	other.OwnerEmail = "other@example.test"
	if err = Provision(ctx, st, signer, cfg, &other); err == nil {
		t.Fatal("bootstrap silently promoted another owner")
	}
	cfg.StripeLivemode = true
	if err = Provision(ctx, st, signer, cfg, settings); err == nil {
		t.Fatal("live billing reused a test database")
	}
}
func TestPlanValidation(t *testing.T) {
	valid := Plan{Slug: "personal", Name: "Personal", Sites: 1, Amount: 9900, Currency: "usd", Interval: "year"}
	if err := validatePlans([]Plan{valid}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []Plan{{}, {Slug: "bad slug", Name: "X", Sites: 1, Amount: 100, Currency: "usd", Interval: "year"}, {Slug: "p", Name: "X", Sites: -1, Amount: 100, Currency: "usd", Interval: "year"}, {Slug: "p", Name: "X", Sites: 1, Amount: 0, Currency: "usd", Interval: "year"}, {Slug: "p", Name: "X", Sites: 1, Amount: 100, Currency: "usd", Interval: "week"}} {
		if err := validatePlans([]Plan{p}); err == nil {
			t.Fatal("invalid plan accepted")
		}
	}
	if err := validatePlans([]Plan{valid, valid}); err == nil {
		t.Fatal("duplicate plan slug accepted")
	}
}
