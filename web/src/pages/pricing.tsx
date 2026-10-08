import { useQuery } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useSiteConfig } from "@/hooks/use-site-config"

interface PublicPlan {
  id: string
  name: string
  license_type: string
  billing_interval: string
  max_sites: number
  checkout_id: string
  price: number | null
  currency: string | null
}

// Stripe charge amounts are two-decimal except its supported zero-decimal
// currencies. ISK/UGX retain the two-decimal API representation.
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
  const digits = zeroDecimal.has(plan.currency) ? 0 : 2
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: plan.currency.toUpperCase(),
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(plan.price / 10 ** digits)
}

export default function PricingPage() {
  const { site_name, attribution_text, attribution_url } = useSiteConfig()
  const catalog = useQuery({
    queryKey: ["public-wordpress-plans"],
    queryFn: async () => {
      const response = await fetch("/api/v1/products/accessible-forms-pro/plans", {
        signal: AbortSignal.timeout(15000),
      })
      const body = await response.json()
      if (!response.ok || body.success !== true || !Array.isArray(body.data?.plans))
        throw new Error("Plans unavailable")
      return body.data.plans as PublicPlan[]
    },
  })
  const availability = useQuery({
    queryKey: ["store-ready"],
    queryFn: async () => {
      const response = await fetch("/ready", { signal: AbortSignal.timeout(10000) })
      const body = await response.json()
      return response.ok && body.ready === true
    },
  })
  const ready = availability.data === true
  return (
    <div className="min-h-screen bg-muted/30 flex flex-col">
      <a
        href="#store-plans"
        className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-50 bg-background p-3 rounded-md"
      >
        Skip to plans
      </a>
      <header className="border-b bg-background">
        <nav
          aria-label="Store navigation"
          className="max-w-5xl mx-auto px-5 py-5 flex items-center justify-between gap-3 flex-wrap"
        >
          <a href="/pricing" className="font-semibold text-lg">
            {site_name}
          </a>
          <Button variant="outline" asChild>
            <a href="/portal">Manage your account</a>
          </Button>
        </nav>
      </header>
      <main id="store-plans" className="w-full max-w-5xl mx-auto px-5 py-12 space-y-8 flex-1">
        <div className="space-y-3 max-w-2xl">
          <h1 className="text-3xl font-bold tracking-tight">Accessible Forms Pro</h1>
          <p className="text-muted-foreground text-lg">
            Add conditional logic, multi-step forms, uploads and entry workflows to Accessible Forms. Choose a license
            for the sites you manage.
          </p>
          <p className="text-sm text-muted-foreground">
            Your license provides updates and support. Installed Pro features keep working when a paid period ends.
          </p>
        </div>
        {catalog.isPending ? (
          <p role="status">Loading plans…</p>
        ) : catalog.isError ? (
          <div role="alert" className="space-y-3">
            <p>Plans could not be loaded. Please try again shortly.</p>
            <Button variant="outline" onClick={() => catalog.refetch()}>
              Try again
            </Button>
          </div>
        ) : (
          <>
            {!ready && (
              <p role="status" className="rounded-lg border bg-background p-4">
                Purchasing is currently unavailable. Existing customers can still manage their licenses from their
                account.
              </p>
            )}
            <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
              {catalog.data?.map((plan) => (
                <Card key={plan.id}>
                  <CardHeader>
                    <CardTitle className="text-xl">{plan.name}</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <p className="text-2xl font-semibold">
                      {priceLabel(plan)}
                      <span className="text-sm font-normal text-muted-foreground">
                        {plan.billing_interval === "year"
                          ? " / year"
                          : plan.billing_interval === "month"
                            ? " / month"
                            : " one-time"}
                      </span>
                    </p>
                    <p>
                      {plan.max_sites === 0
                        ? "Unlimited sites"
                        : `${plan.max_sites} ${plan.max_sites === 1 ? "site" : "sites"}`}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      {plan.license_type === "perpetual"
                        ? "Updates for life"
                        : "Updates for your paid period. Renews automatically; cancel from your account."}
                    </p>
                    {ready && plan.checkout_id && plan.price !== null ? (
                      <Button asChild className="w-full">
                        <a href={`/pay/${encodeURIComponent(plan.checkout_id)}`}>Choose {plan.name}</a>
                      </Button>
                    ) : (
                      <Button disabled className="w-full">
                        Purchasing unavailable
                      </Button>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
            {catalog.data?.length === 0 && <p>Plans will appear here when purchasing opens.</p>}
          </>
        )}
        <p className="text-sm text-muted-foreground">
          Requires the Free Accessible Forms plugin, WordPress 6.5 or later and PHP 8.1 or later. Stripe confirms the
          final price and any applicable tax before payment.
        </p>
      </main>
      <footer className="border-t px-5 py-6 text-center text-sm text-muted-foreground">
        <a className="underline" href={attribution_url} target="_blank" rel="noreferrer">
          {attribution_text}
        </a>
      </footer>
    </div>
  )
}
