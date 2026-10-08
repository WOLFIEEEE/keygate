package merchant

import (
	"context"
	"github.com/tabloy/keygate/internal/config"
	"github.com/tabloy/keygate/internal/storage"
	"github.com/tabloy/keygate/internal/store"
)

// Readiness is distinct from process health: a healthy empty database must not
// accept payment for a product without a deliverable, signed WordPress release.
func Readiness(ctx context.Context, st *store.Store, cfg *config.Config, settings *Settings, webhookSecret string, objects storage.Storage) []string {
	issues := []string{}
	if err := st.DB.PingContext(ctx); err != nil {
		return []string{"database_unavailable"}
	}
	if settings == nil {
		return []string{"store_configuration_missing"}
	}
	if cfg.StripeSecretKey == "" {
		issues = append(issues, "stripe_key_missing")
	}
	if webhookSecret == "" {
		issues = append(issues, "stripe_webhook_pending")
	}
	if cfg.SMTPHost == "" || cfg.SMTPFrom == "" {
		issues = append(issues, "email_configuration_missing")
	}
	if !cfg.IsStorageEnabled() {
		issues = append(issues, "private_storage_missing")
	}
	if len(settings.Plans) == 0 {
		issues = append(issues, "plan_prices_missing")
	}
	if complete, _ := st.GetSetting(ctx, "setup_complete"); complete != "true" {
		issues = append(issues, "owner_setup_pending")
	}
	var plans int
	if err := st.DB.NewRaw("SELECT count(*) FROM plans p JOIN products pr ON pr.id=p.product_id WHERE pr.slug=? AND p.active AND p.stripe_price_id<>''", settings.ProductSlug).Scan(ctx, &plans); err != nil || plans < len(settings.Plans) {
		issues = append(issues, "catalog_setup_pending")
	}
	var artifact struct {
		FileKey  string
		FileSize int64
	}
	if err := st.DB.NewRaw("SELECT a.file_key,a.file_size FROM releases r JOIN products p ON p.id=r.product_id JOIN release_artifacts a ON a.release_id=r.id WHERE p.slug=? AND p.feed_license_required AND p.require_signing AND r.status='published' AND r.channel='stable' AND a.platform='wordpress' AND a.sha256<>'' AND a.ed25519_sig<>'' AND a.wordpress_metadata IS NOT NULL ORDER BY r.published_at DESC LIMIT 1", settings.ProductSlug).Scan(ctx, &artifact); err != nil || artifact.FileKey == "" {
		issues = append(issues, "published_plugin_missing")
	} else if objects == nil {
		issues = append(issues, "plugin_file_unavailable")
	} else if file, err := objects.Head(ctx, artifact.FileKey); err != nil || file.Size != artifact.FileSize {
		issues = append(issues, "plugin_file_unavailable")
	}
	return issues
}
