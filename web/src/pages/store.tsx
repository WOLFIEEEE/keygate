import { Link } from "react-router-dom"
import { PurchasePlans } from "@/components/purchase-plans"
import { FeatureComparison, FreeLink } from "@/components/store-layout"
import { freeDownload } from "@/lib/accessible-forms"

export default function StorePage() {
  return (
    <div className="af-shell af-purchase">
      <section className="af-product-overview" aria-labelledby="product-heading">
        <p className="af-eyebrow">WordPress plugin</p>
        <h1 id="product-heading">Accessible Forms</h1>
        <p className="af-summary">
          Build and publish forms with visible labels, linked validation errors and entry management. Pro adds
          conditional questions, steps, file uploads, custom styling and team workflows.
        </p>
      </section>

      <section className="af-free-option" aria-labelledby="free-heading">
        <div>
          <h2 id="free-heading">Start with Free</h2>
          <p>Unlimited forms, 13 field types, four themes, email notifications and spam controls.</p>
          <p className="af-requirements">
            Version {freeDownload.version} · WordPress {freeDownload.wordpress}+ · PHP {freeDownload.php}+
          </p>
        </div>
        <FreeLink />
      </section>

      <PurchasePlans />

      <section className="af-details" aria-label="Product and installation details">
        <details id="comparison" tabIndex={-1}>
          <summary>Compare Free and Pro</summary>
          <FeatureComparison />
        </details>
        <details id="installation" tabIndex={-1}>
          <summary>Installation and license details</summary>
          <div className="af-installation">
            <ol>
              <li>
                Download Free above. In WordPress, go to Plugins → Add New Plugin → Upload Plugin, upload the ZIP and
                activate it.
              </li>
              <li>
                After purchasing Pro, open <Link to="/portal">My account</Link> to download the Pro ZIP and copy your
                license key. Upload and activate Pro alongside Free.
              </li>
              <li>
                In Accessible Forms → Pro license, connect your key. Eligible updates appear in WordPress’s usual
                Plugins and Updates screens.
              </li>
            </ol>
            <p>Pro requires the Free plugin, WordPress 6.5+ and PHP 8.1+.</p>
            <p>
              Installed Pro features keep working when a paid period ends. An active license provides eligible updates
              and support. Manage downloads, site activations, invoices and subscriptions in your account. To move a
              license, remove the old site’s activation before connecting the new site.
            </p>
          </div>
        </details>
      </section>
    </div>
  )
}
