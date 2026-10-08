import { Check } from "lucide-react"
import { useState } from "react"
import { Link } from "react-router-dom"
import { FeatureComparison, FreeLink, StoreFAQ, StoreMeta } from "@/components/store-layout"
import { type PublicPlan, usePublicPlans, useStoreReady } from "@/hooks/use-store-catalog"

// Stripe uses two decimal API amounts except for these zero-decimal currencies.
// ISK and UGX retain the two-decimal API representation.
const zeroDecimal = new Set([
  "bif",
  "clp",
  "djf",
  "gnf",
  "jpy",
  "kmf",
  "krw",
  "mga",
  "pyg",
  "rwf",
  "vnd",
  "vuv",
  "xaf",
  "xof",
  "xpf",
])

export function priceLabel(plan: PublicPlan) {
  if (plan.price === null || !plan.currency) return "Price available at checkout"
  const currency = plan.currency.toLowerCase()
  const digits = zeroDecimal.has(currency) ? 0 : 2
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency.toUpperCase(),
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(plan.price / 10 ** digits)
}

type Period = "all" | "year" | "month" | "one-time"
const periodOptions: { value: Period; label: string }[] = [
  { value: "all", label: "All plans" },
  { value: "year", label: "Annual" },
  { value: "month", label: "Monthly" },
  { value: "one-time", label: "One-time" },
]
function periodFor(plan: PublicPlan) {
  return plan.billing_interval === "year" || plan.billing_interval === "month" ? plan.billing_interval : "one-time"
}

export default function PricingPage() {
  const catalog = usePublicPlans()
  const availability = useStoreReady()
  const [period, setPeriod] = useState<Period>("all")
  const ready = availability.data === true
  const plans = catalog.data || []
  const periods = new Set(plans.map(periodFor))
  const visiblePlans = period === "all" ? plans : plans.filter((plan) => periodFor(plan) === period)
  return (
    <>
      <StoreMeta
        title="Accessible Forms Pro plans and pricing"
        description="Compare Accessible Forms Free and Pro. Choose a Pro license by site allowance and billing period, with the same advanced features in every plan."
      />
      <section className="af-shell af-page-intro">
        <p className="af-eyebrow">Plans and pricing</p>
        <h1>The right plan for your sites.</h1>
        <p className="af-lead">
          Start with Free. Every Pro plan includes the same advanced features—choose the site allowance and billing
          period that fit your work.
        </p>
        <p className="af-intro-note">
          Installed Pro features keep working when a paid period ends. Your active license provides eligible updates and
          support.
        </p>
      </section>
      <section className="af-shell af-pricing-section" id="store-plans" tabIndex={-1} aria-labelledby="plans-heading">
        <h2 id="plans-heading" className="af-sr-only">
          Accessible Forms plans
        </h2>
        {periods.size > 1 && (
          <fieldset className="af-period-control">
            <legend className="af-sr-only">Filter plans by billing period</legend>
            {periodOptions
              .filter((option) => option.value === "all" || periods.has(option.value))
              .map((option) => (
                <button
                  type="button"
                  key={option.value}
                  aria-pressed={period === option.value}
                  onClick={() => setPeriod(option.value)}
                >
                  {option.label}
                </button>
              ))}
          </fieldset>
        )}
        {!ready && !availability.isPending && (
          <div className="af-store-notice" role="status">
            <p>
              <strong>Pro purchasing is currently unavailable.</strong> You can download Free now. Existing customers
              can still manage licenses from their account.
            </p>
            <button
              type="button"
              onClick={() => {
                availability.refetch()
                catalog.refetch()
              }}
            >
              Check availability
            </button>
          </div>
        )}
        {catalog.isError && (
          <div className="af-store-notice af-store-error" role="alert">
            <p>Pro plans could not be loaded. Free and the product guide are still available.</p>
            <button type="button" onClick={() => catalog.refetch()}>
              Try again
            </button>
          </div>
        )}
        <div className="af-plan-grid">
          <article className="af-plan-card af-free-plan">
            <span className="af-badge">The complete foundation</span>
            <h3>Accessible Forms</h3>
            <p className="af-plan-price">Free</p>
            <p className="af-plan-sites">Use on any number of sites</p>
            <p className="af-plan-description">
              Build and publish everyday forms. No paid plan or license key required.
            </p>
            <ul className="af-check-list">
              <li>
                <Check aria-hidden="true" />
                Unlimited forms and 13 field types
              </li>
              <li>
                <Check aria-hidden="true" />
                Four contrast-checked themes
              </li>
              <li>
                <Check aria-hidden="true" />
                Notifications and entry management
              </li>
              <li>
                <Check aria-hidden="true" />
                Spam controls, retention and privacy tools
              </li>
            </ul>
            <FreeLink />
            <Link to="/products/accessible-forms" className="af-plan-detail">
              Explore Free
            </Link>
          </article>
          {catalog.isPending && (
            <div className="af-plan-loading" role="status">
              Loading Pro plans…
            </div>
          )}
          {!catalog.isPending &&
            !catalog.isError &&
            visiblePlans.map((plan) => (
              <article className="af-plan-card af-paid-plan" key={plan.id}>
                <span className="af-badge af-badge-blue">Accessible Forms Pro</span>
                <h3>{plan.name}</h3>
                <p className="af-plan-price">
                  {priceLabel(plan)}
                  <span>
                    {plan.billing_interval === "year"
                      ? "/ year"
                      : plan.billing_interval === "month"
                        ? "/ month"
                        : "one-time"}
                  </span>
                </p>
                <p className="af-plan-sites">
                  {plan.max_sites === 0
                    ? "Unlimited sites"
                    : `${plan.max_sites} ${plan.max_sites === 1 ? "site" : "sites"}`}
                </p>
                <p className="af-plan-description">
                  {plan.license_type === "perpetual"
                    ? "One-time purchase. Updates for life."
                    : "Updates for your paid period. Renews automatically; cancel from your account."}
                </p>
                <ul className="af-check-list">
                  <li>
                    <Check aria-hidden="true" />
                    Every Free feature, plus all Pro tools
                  </li>
                  <li>
                    <Check aria-hidden="true" />
                    Conditional questions, steps and uploads
                  </li>
                  <li>
                    <Check aria-hidden="true" />
                    Custom styling and delivery rules
                  </li>
                  <li>
                    <Check aria-hidden="true" />
                    Team workflows, exports and Insights
                  </li>
                </ul>
                {ready && plan.checkout_id && plan.price !== null && plan.currency ? (
                  <a className="af-button" href={`/pay/${encodeURIComponent(plan.checkout_id)}`}>
                    Choose {plan.name}
                  </a>
                ) : (
                  <button className="af-button" type="button" disabled>
                    {availability.isPending ? "Checking availability…" : "Purchasing unavailable"}
                  </button>
                )}
                <Link to="/products/accessible-forms-pro" className="af-plan-detail">
                  Explore every Pro feature
                </Link>
              </article>
            ))}
        </div>
        {!catalog.isPending && !catalog.isError && plans.length === 0 && (
          <p className="af-empty-plans">
            Pro plans will appear here when purchasing opens. Explore the features or get started with Free.
          </p>
        )}
        <p className="af-pricing-footnote">
          Pro requires Free, WordPress 6.5+ and PHP 8.1+. Stripe confirms the final price and any applicable tax before
          payment. Manage downloads, site activations, invoices and subscriptions in{" "}
          <Link to="/portal">My account</Link>.
        </p>
      </section>
      <section className="af-feature-band" aria-labelledby="pricing-features-heading">
        <div className="af-shell af-section">
          <div className="af-section-heading">
            <p className="af-eyebrow">What’s included</p>
            <h2 id="pricing-features-heading">Compare Free and Pro.</h2>
            <p>Use the same forms and editor. Add the tools your workflow needs.</p>
          </div>
          <FeatureComparison />
        </div>
      </section>
      <section className="af-shell af-section af-faq-section" aria-labelledby="pricing-faq-heading">
        <div className="af-section-heading">
          <p className="af-eyebrow">Before checkout</p>
          <h2 id="pricing-faq-heading">Understand your license.</h2>
          <p>Site allowances, renewals, updates and installation, explained.</p>
          <Link to="/guide" className="af-text-link">
            Installation and help
          </Link>
        </div>
        <StoreFAQ />
      </section>
    </>
  )
}
