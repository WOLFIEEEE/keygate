package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tabloy/keygate/internal/model"
	"github.com/tabloy/keygate/internal/storage"
	"github.com/tabloy/keygate/internal/store"
	"github.com/tabloy/keygate/pkg/apperr"
)

func TestNormalizeWordPressSite(t *testing.T) {
	groups := [][]string{
		{"http://Example.COM/", "https://example.com", "https://example.com:443/", "http://example.com:80", "https://example.com./"},
		{"https://example.com/shop/", "http://example.com/shop", "https://example.com/a/../shop"},
		{"https://bücher.example/", "https://xn--bcher-kva.example"},
		{"http://[::1]:9100/", "https://[0:0:0:0:0:0:0:1]:9100"},
	}
	seen := map[string]bool{}
	for _, group := range groups {
		first, err := normalizeWordPressSite(group[0])
		if err != nil {
			t.Fatalf("%q: %v", group[0], err)
		}
		if seen[first.identifier] {
			t.Fatalf("distinct installations collided: %q", group[0])
		}
		seen[first.identifier] = true
		for _, raw := range group {
			site, err := normalizeWordPressSite(raw)
			if err != nil || site.identifier != first.identifier {
				t.Errorf("equivalent site %q: %+v %v", raw, site, err)
			}
			if !strings.HasPrefix(site.label, "http") || !strings.HasPrefix(site.identifier, "wp:") || len(site.identifier) != 67 {
				t.Errorf("invalid activation identity for %q: %+v", raw, site)
			}
		}
	}
	for _, raw := range []string{"https://www.example.com", "https://staging.example.com", "https://example.com/Shop", "https://example.com:8443", "https://example.com/another"} {
		site, err := normalizeWordPressSite(raw)
		if err != nil || seen[site.identifier] {
			t.Errorf("separate installation %q collided or failed: %+v %v", raw, site, err)
		}
		seen[site.identifier] = true
	}
	for _, raw := range []string{
		"", "example.com", "//example.com", "ftp://example.com", "https://user:pass@example.com",
		"https://example.com?secret=x", "https://example.com?", "https://example.com#fragment", "https://example.com#",
		"https://example.com:0", "https://example.com:65536", "https://example.com:", "https://example.com:abc",
		"https://example.com\\evil", "https://example.com/%0a", "https://example.com/a%2Fb", "https://example.com/a%5Cb",
		"https://example..com", "https://[fe80::1%25eth0]", "https://exa mple.com", "https://example.com/" + strings.Repeat("a", 2048),
	} {
		if site, err := normalizeWordPressSite(raw); err == nil {
			t.Errorf("invalid site %q accepted: %+v", raw, site)
		}
	}
}

type wordpressTestStorage struct {
	storage.Disabled
	gets int
}

func (s *wordpressTestStorage) PresignedGet(_ context.Context, key, _ string, _ time.Duration) (string, error) {
	s.gets++
	return fmt.Sprintf("https://downloads.example.test/package.zip?key=%s&signature=%d", url.QueryEscape(key), s.gets), nil
}

type wordpressFixture struct {
	st      *store.Store
	svc     *WordPressService
	product *model.Product
	plan    *model.Plan
	license *model.License
	blobs   *wordpressTestStorage
	input   WordPressInput
}

func newWordPressFixture(t *testing.T) *wordpressFixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("integration test requires TEST_DATABASE_URL")
	}
	st, err := store.New(dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.RunMigrations("../../db/migrations"); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	ctx := context.Background()
	suffix := strings.ReplaceAll(store.NewID(), "-", "")
	gatedAt := time.Now().Add(-72 * time.Hour)
	prod := &model.Product{
		Name: "Accessible Forms Pro", Slug: "wp-test-" + suffix, Type: model.ProductTypeDesktop,
		FeedLicenseRequired: true, FeedGatedAt: &gatedAt, DownloadURL: "https://accessible.org/forms/",
	}
	if err := st.CreateProduct(ctx, prod); err != nil {
		t.Fatalf("create product: %v", err)
	}
	t.Cleanup(func() {
		// Core product relations deliberately use RESTRICT, so remove this
		// fixture's dependents in order; never truncate a shared test database.
		for _, table := range []string{"licenses", "plans", "releases", "products"} {
			column := "product_id"
			if table == "products" {
				column = "id"
			}
			if _, err := st.DB.NewRaw("DELETE FROM "+table+" WHERE "+column+" = ?", prod.ID).Exec(ctx); err != nil {
				t.Errorf("clean up %s: %v", table, err)
			}
		}
	})
	plan := &model.Plan{
		ProductID: prod.ID, Name: "Single site", Slug: "single-site-" + suffix,
		LicenseType: "perpetual", LicenseModel: "standard", MaxActivations: 1, GraceDays: 0,
	}
	if err := st.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("create plan: %v", err)
	}
	key := "KG-WP-" + suffix
	lic := &model.License{ProductID: prod.ID, PlanID: plan.ID, Email: suffix + "@example.test", LicenseKey: key, Status: model.StatusActive}
	if err := st.CreateLicense(ctx, lic); err != nil {
		t.Fatalf("create license: %v", err)
	}
	_, signing, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	blobs := &wordpressTestStorage{}
	licenses := NewLicenseService(st, signing, slog.Default(), nil, nil)
	releases := NewReleaseService(ReleaseServiceConfig{Store: st, Storage: blobs})
	return &wordpressFixture{
		st: st, svc: NewWordPressService(st, licenses, releases), product: prod, plan: plan, license: lic, blobs: blobs,
		input: WordPressInput{ProductSlug: prod.Slug, LicenseKey: key, SiteURL: "http://Example.test/", IPAddress: "127.0.0.1", Version: "1.0.0"},
	}
}

func wordpressErrorCode(err error) string {
	var ae *apperr.AppError
	if errors.As(err, &ae) {
		return ae.Code
	}
	return fmt.Sprint(err)
}

func (f *wordpressFixture) publish(t *testing.T, version, platform string, at time.Time) *model.Release {
	t.Helper()
	ctx := context.Background()
	rel := &model.Release{ProductID: f.product.ID, Version: version, Channel: model.ReleaseChannelStable, ReleaseNotes: "Improved accessible validation."}
	if err := f.st.CreateRelease(ctx, rel); err != nil {
		t.Fatal(err)
	}
	artifact := &model.ReleaseArtifact{ReleaseID: rel.ID, Platform: platform}
	if err := f.st.CreateArtifact(ctx, artifact); err != nil {
		t.Fatal(err)
	}
	if err := f.st.UpdateArtifactFile(ctx, artifact.ID, "releases/"+f.product.ID+"/"+version+"/"+platform+".zip", 100, strings.Repeat("ab", 32), "application/zip"); err != nil {
		t.Fatal(err)
	}
	if err := f.st.PublishRelease(ctx, rel.ID, false, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.st.DB.NewRaw("UPDATE releases SET published_at = ? WHERE id = ?", at, rel.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	return rel
}

func TestWordPressActivationLifecycle(t *testing.T) {
	f := newWordPressFixture(t)
	ctx := context.Background()
	first, err := f.svc.Activate(ctx, f.input)
	if err != nil || first.Status != "activated" || first.Token == "" {
		t.Fatalf("first activation: %+v %v", first, err)
	}
	f.input.SiteURL = "https://example.test:443"
	again, err := f.svc.Activate(ctx, f.input)
	if err != nil || again.Status != "already_activated" {
		t.Fatalf("HTTPS migration must reuse the activation: %+v %v", again, err)
	}
	if count, err := f.st.CountActivations(ctx, f.license.ID); err != nil || count != 1 {
		t.Fatalf("repeat activation used another slot: %d %v", count, err)
	}
	second := f.input
	second.SiteURL = "https://example.test/another"
	if _, err := f.svc.Activate(ctx, second); wordpressErrorCode(err) != "ACTIVATION_LIMIT" {
		t.Fatalf("second site escaped the one-site limit: %v", err)
	}
	if _, err := f.svc.Verify(ctx, second); wordpressErrorCode(err) != "LICENSE_NOT_FOUND" {
		t.Fatalf("an unactivated site verified: %v", err)
	}
	if update, err := f.svc.Update(ctx, f.input); err != nil || update.UpdateAvailable || update.Update != nil {
		t.Fatalf("a product with no published packages should have no update: %+v %v", update, err)
	}
	if err := f.svc.Deactivate(ctx, f.input); err != nil {
		t.Fatalf("deactivate first site: %v", err)
	}
	if _, err := f.svc.Activate(ctx, second); err != nil {
		t.Fatalf("freed slot cannot be reused: %v", err)
	}
	if _, err := f.svc.Verify(ctx, f.input); wordpressErrorCode(err) != "LICENSE_NOT_FOUND" {
		t.Fatalf("deactivated site still verified: %v", err)
	}
	if verified, err := f.svc.Verify(ctx, second); err != nil || verified.Status != model.StatusActive {
		t.Fatalf("new site cannot verify: %+v %v", verified, err)
	}

	other := newWordPressFixture(t)
	wrongProduct := second
	wrongProduct.ProductSlug = other.product.Slug
	if _, err := f.svc.Activate(ctx, wrongProduct); wordpressErrorCode(err) != "LICENSE_NOT_FOUND" {
		t.Fatalf("license accepted for a different product: %v", err)
	}
	if err := f.svc.Deactivate(ctx, wrongProduct); wordpressErrorCode(err) != "LICENSE_NOT_FOUND" {
		t.Fatalf("wrong product released an activation: %v", err)
	}
	if _, err := f.svc.Update(ctx, wrongProduct); wordpressErrorCode(err) != "LICENSE_NOT_FOUND" {
		t.Fatalf("wrong product received update metadata: %v", err)
	}
	if _, err := f.svc.Download(ctx, wrongProduct); wordpressErrorCode(err) != "LICENSE_NOT_FOUND" {
		t.Fatalf("wrong product received a package: %v", err)
	}
}

func TestWordPressUpdatesRespectActivationAndMaintenance(t *testing.T) {
	f := newWordPressFixture(t)
	ctx := context.Background()
	cutoff := time.Now().Add(-48 * time.Hour)
	if _, err := f.st.DB.NewRaw("UPDATE licenses SET updates_until = ? WHERE id = ?", cutoff, f.license.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	f.publish(t, "1.0.0", WordPressPlatform, cutoff.Add(-24*time.Hour))
	f.publish(t, "1.1.0", WordPressPlatform, cutoff.Add(-time.Hour))
	f.publish(t, "2.0.0", WordPressPlatform, cutoff.Add(time.Hour))
	f.publish(t, "9.0.0", "darwin-arm64", cutoff.Add(-time.Hour))
	yanked := f.publish(t, "1.2.0", WordPressPlatform, cutoff.Add(-time.Hour))
	if _, err := f.st.DB.NewRaw("UPDATE releases SET status = 'yanked' WHERE id = ?", yanked.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	draft := &model.Release{ProductID: f.product.ID, Version: "99.0.0", Channel: model.ReleaseChannelStable}
	if err := f.st.CreateRelease(ctx, draft); err != nil {
		t.Fatal(err)
	}
	if err := f.st.CreateArtifact(ctx, &model.ReleaseArtifact{
		ReleaseID: draft.ID, Platform: WordPressPlatform, FileKey: "draft/99.0.0.zip",
		SHA256: strings.Repeat("ab", 32), FileSize: 100,
	}); err != nil {
		t.Fatal(err)
	}
	beta := f.publish(t, "4.0.0", WordPressPlatform, cutoff.Add(-time.Hour))
	if _, err := f.st.DB.NewRaw("UPDATE releases SET channel = 'beta' WHERE id = ?", beta.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Update(ctx, f.input); wordpressErrorCode(err) != "LICENSE_NOT_FOUND" {
		t.Fatalf("unactivated site got an update: %v", err)
	}
	if _, err := f.svc.Download(ctx, f.input); wordpressErrorCode(err) != "LICENSE_NOT_FOUND" {
		t.Fatalf("unactivated site got a download: %v", err)
	}
	if f.blobs.gets != 0 {
		t.Fatal("unactivated requests signed a storage URL")
	}
	if _, err := f.svc.Activate(ctx, f.input); err != nil {
		t.Fatal(err)
	}
	update, err := f.svc.Update(ctx, f.input)
	if err != nil || !update.UpdateAvailable || update.Update == nil || update.Update.Version != "1.1.0" {
		t.Fatalf("expected newest entitled WordPress release: %+v %v", update, err)
	}
	if update.Update.Slug != f.product.Slug || update.Update.SHA256 != strings.Repeat("ab", 32) || update.Update.Package == "" || update.Meta["server"] != "Keygate" {
		t.Fatalf("incomplete update metadata: %+v", update)
	}
	current := f.input
	current.Version = "1.1.0"
	before := f.blobs.gets
	if update, err := f.svc.Update(ctx, current); err != nil || update.UpdateAvailable || update.Update != nil || f.blobs.gets != before {
		t.Fatalf("current installation was offered an update: %+v %v", update, err)
	}
	pinned := f.input
	pinned.Version = "2.0.0"
	if _, err := f.svc.Download(ctx, pinned); wordpressErrorCode(err) != "UPDATES_EXPIRED" || f.blobs.gets != before {
		t.Fatalf("newer pinned release escaped maintenance cutoff: %v", err)
	}
	fresh, err := f.svc.Download(ctx, current)
	if err != nil || fresh.URL == update.Update.Package || fresh.Version != "1.1.0" {
		t.Fatalf("download failed to refresh the exact package: %+v %v", fresh, err)
	}
	wrongSite := current
	wrongSite.SiteURL = "https://staging.example.test"
	before = f.blobs.gets
	if _, err := f.svc.Download(ctx, wrongSite); wordpressErrorCode(err) != "LICENSE_NOT_FOUND" || f.blobs.gets != before {
		t.Fatalf("unactivated staging site got a package: %v", err)
	}
	if _, err := f.st.DB.NewRaw("UPDATE licenses SET updates_until = ? WHERE id = ?", time.Now().Add(24*time.Hour), f.license.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if renewed, err := f.svc.Update(ctx, current); err != nil || renewed.Update == nil || renewed.Update.Version != "2.0.0" {
		t.Fatalf("renewal did not unlock the new release: %+v %v", renewed, err)
	}
	if _, err := f.st.DB.NewRaw("UPDATE licenses SET status = 'revoked' WHERE id = ?", f.license.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	before = f.blobs.gets
	if _, err := f.svc.Download(ctx, current); wordpressErrorCode(err) != "LICENSE_NOT_FOUND" || f.blobs.gets != before {
		t.Fatalf("revoked license got a fresh URL: %v", err)
	}
}

func TestWordPressDownloadsRejectUnusableLicenses(t *testing.T) {
	f := newWordPressFixture(t)
	ctx := context.Background()
	f.publish(t, "1.0.0", WordPressPlatform, time.Now().Add(-time.Hour))
	if _, err := f.svc.Activate(ctx, f.input); err != nil {
		t.Fatal(err)
	}
	past, future := time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour)
	for _, tt := range []struct {
		name, status string
		until        *time.Time
		allowed      bool
	}{
		{"active", model.StatusActive, nil, true},
		{"suspended", model.StatusSuspended, nil, false},
		{"revoked", model.StatusRevoked, nil, false},
		{"expired", model.StatusActive, &past, false},
		{"canceled after paid period", model.StatusCanceled, &past, false},
		{"canceled within paid period", model.StatusCanceled, &future, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := f.st.DB.NewRaw("UPDATE licenses SET status = ?, valid_until = ? WHERE id = ?", tt.status, tt.until, f.license.ID).Exec(ctx); err != nil {
				t.Fatal(err)
			}
			before := f.blobs.gets
			out, err := f.svc.Download(ctx, f.input)
			if tt.allowed {
				if err != nil || out.URL == "" {
					t.Fatalf("entitled license refused: %+v %v", out, err)
				}
			} else if wordpressErrorCode(err) != "LICENSE_NOT_FOUND" || f.blobs.gets != before {
				t.Fatalf("unusable license got a package: %+v %v", out, err)
			}
		})
	}
}
