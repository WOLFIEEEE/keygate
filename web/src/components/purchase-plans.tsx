import { useState } from "react"
import { Link } from "react-router-dom"
import { type PublicPlan, usePublicPlans, useStoreReady } from "@/hooks/use-store-catalog"
import { approvedProPlan } from "@/lib/accessible-forms"

// ISK and UGX retain Stripe's two-decimal API representation.
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

export function PurchasePlans() {
  const catalog = usePublicPlans()
  const availability = useStoreReady()
  const [period, setPeriod] = useState<Period>("all")
  const ready = availability.data === true
  const publishedPlans = catalog.data || []
  const plans = publishedPlans.length === 0 && !ready ? [approvedProPlan] : publishedPlans
  const periods = new Set(plans.map(periodFor))
  const selectedPeriod = period === "all" || periods.has(period) ? period : "all"
  const visiblePlans = selectedPeriod === "all" ? plans : plans.filter((plan) => periodFor(plan) === selectedPeriod)

  return (
    <section className="af-plans" id="store-plans" tabIndex={-1} aria-labelledby="plans-heading">
      <h2 id="plans-heading">Choose a Pro license</h2>
      <p className="af-plan-intro">
        {plans.length === 1
          ? "All Pro features, with eligible updates and support for your paid period."
          : "Every Pro plan includes the same features. Choose your site allowance and billing period."}
      </p>
      {periods.size > 1 && (
        <fieldset className="af-period-control">
          <legend className="af-sr-only">Filter plans by billing period</legend>
          {periodOptions
            .filter((option) => option.value === "all" || periods.has(option.value))
            .map((option) => (
              <button
                type="button"
                key={option.value}
                aria-pressed={selectedPeriod === option.value}
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
            <strong>Pro purchasing is currently unavailable.</strong> Free downloads and existing customer accounts are
            still available.
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
          <p>Pro plans could not be loaded. You can still download Free or open your account.</p>
          <button type="button" onClick={() => catalog.refetch()}>
            Try again
          </button>
        </div>
      )}
      {catalog.isPending && (
        <p className="af-plan-loading" role="status">
          Loading Pro plans…
        </p>
      )}
      {!catalog.isPending && !catalog.isError && (
        <div className="af-plan-grid">
          {visiblePlans.map((plan) => (
            <article className={`af-plan-card${plans.length === 1 ? " af-plan-card-single" : ""}`} key={plan.id}>
              <div className="af-plan-info">
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
                    : plan.price !== null &&
                        plan.currency &&
                        (plan.billing_interval === "year" || plan.billing_interval === "month")
                      ? `Renews at ${priceLabel(plan)} per ${plan.billing_interval}; cancel from your account.`
                      : "Updates for your paid period. Renews automatically; cancel from your account."}
                </p>
              </div>
              {ready && plan.checkout_id && plan.price !== null && plan.currency ? (
                <a className="af-button" href={`/pay/${encodeURIComponent(plan.checkout_id)}`}>
                  Choose {plan.name}
                </a>
              ) : (
                <button className="af-button" type="button" disabled>
                  {availability.isPending ? "Checking availability…" : "Purchasing unavailable"}
                </button>
              )}
            </article>
          ))}
        </div>
      )}
      {!catalog.isPending && !catalog.isError && plans.length === 0 && ready && (
        <p className="af-empty-plans">No Pro plans are available at the moment. Please check again later.</p>
      )}
      <p className="af-pricing-footnote">
        Pro requires Free, WordPress 6.5+ and PHP 8.1+. Stripe confirms the final price and any applicable tax before
        payment. Already purchased? <Link to="/portal">Manage your license and billing</Link>.
      </p>
      {plans.length === 1 && plans[0].max_sites === 1 && (
        <p className="af-pricing-footnote">
          Each additional site needs its own license. Manage all your licenses in one account.
        </p>
      )}
    </section>
  )
}
