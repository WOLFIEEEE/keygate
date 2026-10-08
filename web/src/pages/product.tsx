import { ArrowRight, Check, ChevronRight, Download, Layers, ShieldCheck } from "lucide-react"
import { Link } from "react-router-dom"
import { FormExample } from "@/components/form-example"
import { FeatureComparison, FreeLink, ProLink, StoreCallout, StoreFAQ, StoreMeta } from "@/components/store-layout"
import { useWordPressRelease } from "@/hooks/use-store-catalog"
import { coreFieldTypes, freeDownload } from "@/lib/accessible-forms"

export default function ProductPage({ pro = false }: { pro?: boolean }) {
  const release = useWordPressRelease()
  const title = pro ? "Accessible Forms Pro" : "Accessible Forms"
  return (
    <>
      <StoreMeta
        title={title}
        description={
          pro
            ? "Add conditional questions, multi-step forms, private uploads, custom styling, delivery rules and team workflows to Accessible Forms for WordPress."
            : "A free WordPress form builder with unlimited forms, 13 field types, four themes, clear validation and entry management. Download Accessible Forms."
        }
      />
      <div className="af-shell af-breadcrumb">
        <Link to="/">Store</Link>
        <ChevronRight aria-hidden="true" size={14} />
        <span>{title}</span>
      </div>
      <section className="af-shell af-product-hero" aria-labelledby="product-heading">
        <div>
          <p className="af-eyebrow">{pro ? "The advanced add-on for WordPress" : "The free WordPress form builder"}</p>
          <h1 id="product-heading">
            {title}
            <span>{pro ? "More possibilities. Same clear experience." : "Clarity from the first question."}</span>
          </h1>
          <p className="af-lead">
            {pro
              ? "Make complex forms feel manageable with relevant questions, smaller steps and a useful workflow for every submission."
              : "Create contact forms, enquiries and everyday requests with visible labels, helpful errors and organized submissions."}
          </p>
          <div className="af-actions">
            {pro ? <ProLink /> : <FreeLink />}
            <Link className="af-text-link" to={pro ? "/products/accessible-forms" : "/products/accessible-forms-pro"}>
              {pro ? "Explore the Free plugin" : "See what Pro adds"}
              <ArrowRight aria-hidden="true" size={17} />
            </Link>
          </div>
          <p className="af-small-note">
            {pro
              ? "Requires Free. All Pro plans include every Pro feature."
              : `Free ${freeDownload.version} · No license key required.`}
          </p>
        </div>
        <aside className="af-product-facts" aria-label="Product requirements">
          <img src="/accessible-forms-icon.png" alt="" width="92" height="92" />
          <h2>Made for WordPress</h2>
          <dl>
            <div>
              <dt>WordPress</dt>
              <dd>{pro ? release.data?.wordpress_metadata?.requires || "6.5" : freeDownload.wordpress}+ </dd>
            </div>
            <div>
              <dt>PHP</dt>
              <dd>{pro ? release.data?.wordpress_metadata?.requires_php || "8.1" : freeDownload.php}+ </dd>
            </div>
            <div>
              <dt>{pro ? "Free plugin" : "Current version"}</dt>
              <dd>{pro ? "Installed and active" : freeDownload.version}</dd>
            </div>
            <div>
              <dt>License</dt>
              <dd>GPLv2 or later</dd>
            </div>
          </dl>
          {pro && <p>Latest Pro release: {release.data?.version || "available in your account"}</p>}
          <Link to="/guide">
            Installation guide <ArrowRight aria-hidden="true" size={16} />
          </Link>
        </aside>
      </section>
      <section className="af-feature-band">
        <div className="af-shell af-benefit-grid">
          <article>
            <ShieldCheck aria-hidden="true" />
            <h2>Safeguards built in</h2>
            <p>
              Visible labels, linked error summaries, protected focus styles and contrast checks stay part of the
              experience.
            </p>
          </article>
          <article>
            <Layers aria-hidden="true" />
            <h2>One connected workflow</h2>
            <p>
              {pro
                ? "Advanced settings appear in the same form editor and entries you already use."
                : "Build, publish, receive notifications and manage entries inside WordPress."}
            </p>
          </article>
          <article>
            <Download aria-hidden="true" />
            <h2>{pro ? "Updates on your terms" : "A complete free foundation"}</h2>
            <p>
              {pro
                ? "Your license covers eligible private updates and support. Installed features keep working after expiry."
                : "Build unlimited forms without a paid plan. Add Pro when you need its advanced tools."}
            </p>
          </article>
        </div>
      </section>
      <section className="af-shell af-section" id="features" tabIndex={-1} aria-labelledby="features-heading">
        <div className="af-section-heading">
          <p className="af-eyebrow">The complete feature set</p>
          <h2 id="features-heading">From simple forms to team workflows.</h2>
          <p>Free provides the foundation. Pro adds advanced controls while keeping the same core safeguards.</p>
        </div>
        <FeatureComparison />
        <div className="af-field-types">
          <h3>Every core field type, included in Free.</h3>
          <p>Use labels, help text and required answers to make each question clear.</p>
          <ul>
            {coreFieldTypes.map((type) => (
              <li key={type}>{type}</li>
            ))}
          </ul>
        </div>
      </section>
      {pro && (
        <section className="af-shell af-section af-pro-details" aria-labelledby="pro-detail-heading">
          <div className="af-section-heading">
            <p className="af-eyebrow">A closer look at Pro</p>
            <h2 id="pro-detail-heading">Keep the useful details.</h2>
          </div>
          <div className="af-detail-grid">
            <article>
              <h3>Conditional logic and review</h3>
              <p>
                Use AND/OR question and notification rules, including number, date and time comparisons. Validate
                related answers, split long forms into named steps and offer an optional final review with Edit buttons.
              </p>
            </article>
            <article>
              <h3>Private uploads and saved progress</h3>
              <p>
                Accept PDF, JPEG, PNG and text files. Uploads are quarantined for review and stay outside the media
                library. Visitors can save answers for seven days using a private resume link.
              </p>
            </article>
            <article>
              <h3>Delivery you can inspect</h3>
              <p>
                Choose SMTP, formatted messages and conditional recipients. Use optional WordPress mail fallback,
                inspect route history, send visitor receipts or connect a signed webhook to your own endpoint.
              </p>
            </article>
            <article>
              <h3>A working inbox for your team</h3>
              <p>
                Assign entries, set follow-up dates, add private notes and reply from the submission screen. Save views,
                prepare CSV exports in the background and review form Insights.
              </p>
            </article>
          </div>
        </section>
      )}
      <section className="af-example-section">
        <div className="af-shell af-split">
          <div>
            <p className="af-eyebrow">Try the experience</p>
            <h2>
              Clear questions.
              <br />
              Useful feedback.
            </h2>
            <p className="af-lead">
              Tab through the example. Try submitting it empty, then follow an error back to its field.
            </p>
            <p>
              Switch to conditional questions and choose a website project to reveal the relevant budget field. This is
              a browser-only example of the interaction; it does not send or save answers.
            </p>
            <ul className="af-check-list">
              <li>
                <Check aria-hidden="true" />
                Visible labels remain above each answer
              </li>
              <li>
                <Check aria-hidden="true" />
                Errors link to the questions that need attention
              </li>
              <li>
                <Check aria-hidden="true" />
                Conditional changes receive a polite announcement
              </li>
            </ul>
          </div>
          <FormExample />
        </div>
      </section>
      <section className="af-shell af-section af-faq-section" aria-labelledby="product-faq-heading">
        <div className="af-section-heading">
          <p className="af-eyebrow">Know what to expect</p>
          <h2 id="product-faq-heading">Product and license questions.</h2>
          <p>Installation, updates, privacy and accessibility, explained.</p>
        </div>
        <StoreFAQ />
      </section>
      {release.data?.release_notes && (
        <section className="af-shell af-section af-release-notes" aria-labelledby="release-heading">
          <p className="af-eyebrow">From the release channel</p>
          <h2 id="release-heading">Pro {release.data.version} release notes</h2>
          <p>{release.data.release_notes}</p>
        </section>
      )}
      <StoreCallout />
    </>
  )
}
