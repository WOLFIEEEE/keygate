import { ArrowRight, Check, Download, KeyRound, SlidersHorizontal } from "lucide-react"
import { Link } from "react-router-dom"
import { FreeLink, ProLink, StoreFAQ, StoreMeta } from "@/components/store-layout"
import { freeDownload } from "@/lib/accessible-forms"

export default function GuidePage() {
  return (
    <>
      <StoreMeta
        title="Accessible Forms installation and help"
        description="Install Accessible Forms and Pro, connect your license, publish your first form and understand updates, delivery and data handling."
      />
      <section className="af-shell af-page-intro">
        <p className="af-eyebrow">Getting started</p>
        <h1>A clear path to your first form.</h1>
        <p className="af-lead">Install Free, publish a reviewed form and add Pro when you need advanced tools.</p>
        <nav className="af-jump-links" aria-label="Guide topics">
          <a href="#install">Installation</a>
          <a href="#first-form">Your first form</a>
          <a href="#updates">Licenses and updates</a>
          <a href="#data">Data and privacy</a>
          <a href="#questions">Common questions</a>
        </nav>
      </section>
      <section className="af-shell af-section" id="install" tabIndex={-1} aria-labelledby="install-heading">
        <div className="af-section-heading">
          <p className="af-eyebrow">Install the plugins</p>
          <h2 id="install-heading">Free first. Pro alongside it.</h2>
          <p>Free needs WordPress 6.4+ and PHP 8.1+. Pro needs WordPress 6.5+, PHP 8.1+ and Accessible Forms 1.0.0+.</p>
        </div>
        <div className="af-install-grid">
          <article>
            <Download aria-hidden="true" />
            <h3>Install Accessible Forms</h3>
            <ol>
              <li>Download the Free {freeDownload.version} ZIP below.</li>
              <li>
                In WordPress, open <strong>Plugins → Add New Plugin → Upload Plugin</strong>.
              </li>
              <li>
                Select the ZIP, install it and choose <strong>Activate Plugin</strong>.
              </li>
              <li>
                Open <strong>Accessible Forms → Settings</strong> and review delivery, privacy and spam settings.
              </li>
            </ol>
            <FreeLink />
          </article>
          <article>
            <KeyRound aria-hidden="true" />
            <h3>Add Accessible Forms Pro</h3>
            <ol>
              <li>Choose a Pro plan and purchase using your preferred account email.</li>
              <li>
                Open <Link to="/portal">My account</Link>, sign in with that email and download your Pro ZIP.
              </li>
              <li>Upload and activate Pro using the same WordPress plugin upload screen. Keep Free active.</li>
              <li>
                Copy your key into <strong>Accessible Forms → Pro license</strong> and connect this site.
              </li>
            </ol>
            <ProLink />
          </article>
        </div>
      </section>
      <section className="af-feature-band" id="first-form" tabIndex={-1} aria-labelledby="first-form-heading">
        <div className="af-shell af-section af-split">
          <div>
            <p className="af-eyebrow">From draft to published form</p>
            <h2 id="first-form-heading">Make the first form a small one.</h2>
            <p className="af-lead">
              A short contact form is a useful way to check questions, notifications and placement before collecting
              real messages.
            </p>
            <SlidersHorizontal aria-hidden="true" className="af-guide-icon" />
          </div>
          <ol className="af-guide-steps">
            <li>
              <h3>Build the questions</h3>
              <p>
                Create a form from a starter template. Use clear visible labels, mark required answers and add help text
                only where it is useful.
              </p>
            </li>
            <li>
              <h3>Review appearance and delivery</h3>
              <p>
                Choose a theme, check contrast, preview the notification and send a test email. On staging, use test
                delivery to avoid sending real messages.
              </p>
            </li>
            <li>
              <h3>Publish and place the form</h3>
              <p>
                Publish the reviewed draft. Add an Accessible Form block or shortcode, or use the guided page and post
                placement.
              </p>
            </li>
            <li>
              <h3>Test the visitor’s experience</h3>
              <p>
                Use a keyboard, submit missing and invalid answers, and review the success message. Check your theme and
                assistive technology, then verify the saved entry and delivery history.
              </p>
            </li>
          </ol>
        </div>
      </section>
      <section className="af-shell af-section" id="updates" tabIndex={-1} aria-labelledby="updates-heading">
        <div className="af-section-heading">
          <p className="af-eyebrow">Licenses and updates</p>
          <h2 id="updates-heading">Manage the connection in one place.</h2>
        </div>
        <div className="af-detail-grid">
          <article>
            <h3>Connect and move sites</h3>
            <p>
              Your plan’s site allowance covers distinct site addresses. Connect a key in the Pro license screen. To
              move it, disconnect the old site or remove its activation in your account before connecting the new site.
            </p>
          </article>
          <article>
            <h3>Install eligible updates</h3>
            <p>
              After connecting, Pro updates appear on the normal WordPress Updates and Plugins screens. Downloaded
              packages are verified before installation. If Pro needs a newer Free version, update Free first; an
              incompatible Pro update leaves the current version installed.
            </p>
          </article>
          <article>
            <h3>Renewal and cancellation</h3>
            <p>
              Subscription plans renew automatically. Manage cancellation, payment details and invoices through your
              account. The current plan and paid period determine your update eligibility. Expiry or a temporary
              license-service outage does not disable installed Pro features.
            </p>
          </article>
          <article>
            <h3>Deactivating Pro</h3>
            <p>
              Forms, completed entries and Pro settings are kept. Advanced behavior pauses: uploads are hidden,
              conditions stop hiding questions, steps show on one page and only the main notification is sent. Temporary
              saved progress and prepared exports are removed. Per-form entry restrictions remain enforced.
            </p>
          </article>
        </div>
        <Link to="/portal" className="af-button af-button-secondary">
          Open licenses and billing <ArrowRight aria-hidden="true" size={17} />
        </Link>
      </section>
      <section className="af-example-section" id="data" tabIndex={-1} aria-labelledby="data-heading">
        <div className="af-shell af-section af-split">
          <div>
            <p className="af-eyebrow">Data and privacy</p>
            <h2 id="data-heading">Know where information goes.</h2>
            <p className="af-lead">The licensing connection is separate from visitors’ form answers.</p>
          </div>
          <div className="af-editorial-list">
            <article>
              <h3>Forms and entries</h3>
              <p>
                Entries are managed in your WordPress site. Configure retention and access permissions. Notification
                emails, your chosen SMTP provider and optional webhooks receive the information required by the features
                you enable.
              </p>
            </article>
            <article>
              <h3>Licenses and checkout</h3>
              <p>
                The license service receives the key and site address for activation and updates, and the requesting
                server’s IP address. It does not receive form answers or entries through licensing. Stripe handles
                checkout, payment details, subscriptions and invoices.
              </p>
            </article>
            <article>
              <h3>Spam services and uploads</h3>
              <p>
                Optional Turnstile sends browser and network information to Cloudflare. Pro upload review is a
                quarantine and approval step, not a malware scan. Test each enabled service and keep retention suitable
                for the information you collect.
              </p>
            </article>
          </div>
        </div>
      </section>
      <section
        className="af-shell af-section af-faq-section"
        id="questions"
        tabIndex={-1}
        aria-labelledby="guide-faq-heading"
      >
        <div className="af-section-heading">
          <p className="af-eyebrow">Common questions</p>
          <h2 id="guide-faq-heading">Find your next step.</h2>
          <p>
            For product enquiries, visit{" "}
            <a className="af-inline-link" href="https://accessible.org/">
              Accessible.org
            </a>
            . For purchased licenses, sign in using the email you entered at checkout.
          </p>
          <p className="af-guide-check">
            <Check aria-hidden="true" size={17} /> Keep a site backup before plugin updates.
          </p>
        </div>
        <StoreFAQ />
      </section>
    </>
  )
}
