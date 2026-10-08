import { ArrowRight, Check, ClipboardList, Keyboard, Send } from "lucide-react"
import { Link } from "react-router-dom"
import { FormExample } from "@/components/form-example"
import { FreeLink, ProLink, StoreCallout, StoreFAQ, StoreMeta } from "@/components/store-layout"

export default function StorePage() {
  return (
    <>
      <StoreMeta
        title="Accessible Forms for WordPress"
        description="Build WordPress forms with visible labels, linked validation errors and a clear submission workflow. Start free or add advanced tools with Accessible Forms Pro."
      />
      <section className="af-shell af-hero" aria-labelledby="store-heading">
        <div className="af-hero-copy">
          <p className="af-eyebrow">Accessible Forms for WordPress</p>
          <h1 id="store-heading">
            Good forms.
            <br />
            <span>For everyone.</span>
          </h1>
          <p className="af-lead">
            Make every question clear, every error useful and every submission easier to manage.
          </p>
          <p className="af-hero-detail">
            A form builder with accessibility safeguards built in. Start with Free, then add Pro when your forms and
            team need more.
          </p>
          <div className="af-actions">
            <FreeLink />
            <Link className="af-text-link" to="/products/accessible-forms-pro">
              Meet Accessible Forms Pro <ArrowRight aria-hidden="true" size={17} />
            </Link>
          </div>
          <p className="af-small-note">Free is a standalone plugin. Pro works alongside it.</p>
        </div>
        <FormExample />
      </section>
      <div className="af-proof-strip">
        <div className="af-shell">
          <span>
            <Keyboard aria-hidden="true" size={18} /> Keyboard-friendly forms
          </span>
          <span>
            <ClipboardList aria-hidden="true" size={18} /> Clear validation and review
          </span>
          <span>
            <Send aria-hidden="true" size={18} /> Entries managed in WordPress
          </span>
        </div>
      </div>
      <section className="af-shell af-section" id="products" tabIndex={-1} aria-labelledby="products-heading">
        <div className="af-section-heading">
          <p className="af-eyebrow">Choose your starting point</p>
          <h2 id="products-heading">One form builder. Room to grow.</h2>
          <p>Both plugins use the same forms, editor and core accessibility safeguards.</p>
        </div>
        <div className="af-product-grid">
          <article className="af-product-card">
            <div className="af-product-title">
              <img src="/accessible-forms-icon.png" alt="" width="56" height="56" />
              <span className="af-badge">Free plugin</span>
            </div>
            <h3>Accessible Forms</h3>
            <p>Build, publish and manage everyday forms with a thoughtful set of essentials.</p>
            <ul className="af-check-list">
              <li>
                <Check aria-hidden="true" />
                Unlimited forms and 13 field types
              </li>
              <li>
                <Check aria-hidden="true" />
                Four themes with contrast checks
              </li>
              <li>
                <Check aria-hidden="true" />
                Notifications, spam controls and entries
              </li>
              <li>
                <Check aria-hidden="true" />
                Block, shortcode and guided publishing
              </li>
            </ul>
            <div className="af-card-actions">
              <FreeLink />
              <Link to="/products/accessible-forms" className="af-text-link">
                Explore Free <ArrowRight aria-hidden="true" size={16} />
              </Link>
            </div>
          </article>
          <article className="af-product-card af-product-pro">
            <div className="af-product-title">
              <img src="/accessible-forms-icon.png" alt="" width="56" height="56" />
              <span className="af-badge af-badge-blue">Pro add-on</span>
            </div>
            <h3>Accessible Forms Pro</h3>
            <p>Ask smarter questions and give submissions a clear path through your team.</p>
            <ul className="af-check-list">
              <li>
                <Check aria-hidden="true" />
                Conditional logic, steps and uploads
              </li>
              <li>
                <Check aria-hidden="true" />
                Custom styling and brand presets
              </li>
              <li>
                <Check aria-hidden="true" />
                SMTP, notification rules and webhooks
              </li>
              <li>
                <Check aria-hidden="true" />
                Assignments, CSV exports and Insights
              </li>
            </ul>
            <div className="af-card-actions">
              <ProLink />
              <Link to="/products/accessible-forms-pro" className="af-text-link">
                Explore Pro <ArrowRight aria-hidden="true" size={16} />
              </Link>
            </div>
          </article>
        </div>
      </section>
      <section className="af-feature-band">
        <div className="af-shell af-split">
          <div>
            <p className="af-eyebrow">From question to follow-up</p>
            <h2>A better experience on both sides of the form.</h2>
            <p className="af-lead">
              Visitors get clear questions and useful feedback. Your team gets organized submissions and tools to act on
              them.
            </p>
            <Link to="/products/accessible-forms#features" className="af-text-link">
              See the full feature comparison <ArrowRight aria-hidden="true" size={17} />
            </Link>
          </div>
          <div className="af-editorial-list">
            <article>
              <h3>Before publishing</h3>
              <p>Build a draft, preview the theme, test notifications and place the reviewed form.</p>
            </article>
            <article>
              <h3>While answering</h3>
              <p>Keep labels visible, connect errors to their fields and show a clear next step.</p>
            </article>
            <article>
              <h3>After submitting</h3>
              <p>Search and review entries. With Pro, assign a teammate, add notes and track follow-up dates.</p>
            </article>
          </div>
        </div>
      </section>
      <section className="af-shell af-section af-faq-section" aria-labelledby="store-faq-heading">
        <div className="af-section-heading">
          <p className="af-eyebrow">Before you choose</p>
          <h2 id="store-faq-heading">A few useful answers.</h2>
          <p>Learn how the two plugins work together and what your license includes.</p>
          <Link to="/guide" className="af-text-link">
            Read the getting-started guide <ArrowRight aria-hidden="true" size={16} />
          </Link>
        </div>
        <StoreFAQ compact />
      </section>
      <StoreCallout />
    </>
  )
}
