package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tabloy/keygate/internal/model"
	"github.com/tabloy/keygate/internal/service"
	"github.com/tabloy/keygate/internal/storage"
	"github.com/tabloy/keygate/internal/store"
)

type portalArtifactStorage struct{ storage.Disabled }

func (portalArtifactStorage) PresignedGet(context.Context, string, string, time.Duration) (string, error) {
	return "https://downloads.example.test/private.zip?signature=fixture", nil
}

func TestPortalInstallerOwnershipAndEntitlement(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	st, err := store.New(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err = st.RunMigrations("../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	gin.SetMode(gin.TestMode)
	product := &model.Product{Name: "Accessible Forms Pro", Slug: "accessible-forms-pro", Type: model.ProductTypeDesktop}
	if err = st.CreateProduct(ctx, product); err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, table := range []string{"licenses", "releases", "plans", "products"} {
			column := "product_id"
			if table == "products" {
				column = "id"
			}
			st.DB.NewRaw("DELETE FROM "+table+" WHERE "+column+"=?", product.ID).Exec(ctx)
		}
	}()
	plan := &model.Plan{ProductID: product.ID, Name: "Personal", Slug: "portal-download-test", LicenseType: "perpetual", LicenseModel: "standard", MaxActivations: 1}
	if err = st.CreatePlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	lic := &model.License{ProductID: product.ID, PlanID: plan.ID, LicenseKey: "KG-PORTAL-FIXTURE", Email: "owner@example.test", Status: model.StatusActive, UpdatesTermsSet: true}
	if err = st.CreateLicense(ctx, lic); err != nil {
		t.Fatal(err)
	}
	rel := &model.Release{ProductID: product.ID, Version: "1.0.0", Channel: "stable", Status: "draft"}
	if err = st.CreateRelease(ctx, rel); err != nil {
		t.Fatal(err)
	}
	artifact := &model.ReleaseArtifact{ReleaseID: rel.ID, Platform: "wordpress", FileKey: "fixture.zip", FileSize: 100, SHA256: strings.Repeat("a", 64), Filename: "pro.zip"}
	if err = st.CreateArtifact(ctx, artifact); err != nil {
		t.Fatal(err)
	}
	if _, err = st.DB.NewRaw("UPDATE releases SET status='published',published_at=now() WHERE id=?", rel.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	h := &PortalDownloadsHandler{Store: st, Releases: service.NewReleaseService(service.ReleaseServiceConfig{Store: st, Storage: portalArtifactStorage{}, Logger: slog.Default()})}
	call := func(email string, id string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("email", email)
		c.Request = httptest.NewRequest(http.MethodPost, "/portal/downloads/wordpress", strings.NewReader(`{"license_id":"`+id+`"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		h.Download(c)
		return w
	}
	for _, tc := range []struct {
		email, id string
		status    int
	}{{"", lic.ID, 401}, {"other@example.test", lic.ID, 404}, {lic.Email, "missing", 404}, {"OWNER@example.test", lic.ID, 200}} {
		w := call(tc.email, tc.id)
		if w.Code != tc.status {
			t.Fatalf("expected %d got %d: %s", tc.status, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "private, no-store" || strings.Contains(w.Body.String(), lic.LicenseKey) {
			t.Fatal("credential or cache privacy failed")
		}
	}
	var activations int
	if err = st.DB.NewRaw("SELECT count(*) FROM activations WHERE license_id=?", lic.ID).Scan(ctx, &activations); err != nil || activations != 0 {
		t.Fatal("initial installer used an activation slot")
	}
	if _, err = st.DB.NewRaw("UPDATE licenses SET status='revoked' WHERE id=?", lic.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if w := call(lic.Email, lic.ID); w.Code != 404 {
		t.Fatal("revoked license downloaded an installer")
	}
}
