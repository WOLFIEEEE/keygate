import assert from "node:assert/strict"
import { createHash } from "node:crypto"
import { mkdir, writeFile } from "node:fs/promises"
import AxeBuilder from "@axe-core/playwright"
import { chromium, firefox, webkit } from "playwright"

const base = process.env.STORE_BASE_URL || "http://localhost:9100"
const output = process.env.STORE_CHECK_OUTPUT || "../../tmp/storefront-checks"
await mkdir(output, { recursive: true })
const report = { base, engines: [], checks: 0 }
function check(label, value) { assert(value, label); report.checks++; console.log(`PASS ${label}`) }
const paths = ["/", "/products/accessible-forms", "/products/accessible-forms-pro", "/pricing", "/guide"]
const plans = [
  { id: "annual", name: "Personal", price: 9900, currency: "usd", max_sites: 1, license_type: "subscription", billing_interval: "year", checkout_id: "public-personal" },
  { id: "monthly", name: "Team", price: 1499, currency: "usd", max_sites: 5, license_type: "subscription", billing_interval: "month", checkout_id: "public-team" },
  { id: "once", name: "Lifetime", price: 5000, currency: "jpy", max_sites: 0, license_type: "perpetual", billing_interval: "", checkout_id: "public-lifetime" },
]

for (const [name, engine] of Object.entries({ chromium, firefox, webkit })) {
  const start = report.checks
  const browser = await engine.launch()
  const context = await browser.newContext({ locale: "en-US", viewport: { width: 1280, height: 900 }, reducedMotion: "reduce" })
  const page = await context.newPage()
  const errors = []
  page.on("pageerror", (error) => errors.push(error.message))
  try {
    for (const path of paths) {
      await page.goto(base + path)
      await page.locator("h1").waitFor()
      await page.waitForFunction(() => document.title.includes("Accessible.org"))
      check(`${name}: ${path} has one main heading`, await page.locator("h1").count() === 1)
      check(`${name}: ${path} keeps attribution visible`, await page.getByRole("link", { name: "Powered by Keygate" }).isVisible())
      check(`${name}: ${path} has a Free installer link`, await page.locator('a[href="/downloads/accessible-forms-by-accessible-org-1.0.0.zip"]').count() > 0)
      const axe = await new AxeBuilder({ page }).analyze()
      if (axe.violations.length) console.log(JSON.stringify(axe.violations.map((v) => ({ id: v.id, nodes: v.nodes.map((n) => n.target) }))))
      check(`${name}: ${path} passes axe`, axe.violations.length === 0)
      for (const width of [320, 375, 768]) {
        await page.setViewportSize({ width, height: 900 })
        check(`${name}: ${path} reflows at ${width}px`, await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth))
      }
      await page.setViewportSize({ width: 1280, height: 900 })
    }
    await page.goto(base + "/")
    const posted = []
    // The shared signed-in account provider may refresh its session in the
    // background. That request carries no demo answers and is unrelated to
    // form submission. Any other POST during the preview is a failure.
    const collect = (request) => { if (request.method() === "POST" && new URL(request.url()).pathname !== "/api/v1/auth/refresh") posted.push(request.url()) }
    page.on("request", collect)
    await page.getByRole("button", { name: "Try this form" }).click()
    await page.getByRole("alert").waitFor()
    check(`${name}: error summary takes focus`, await page.getByRole("alert").evaluate((node) => node === document.activeElement))
    await page.getByRole("link", { name: "Enter a name for this example." }).click()
    check(`${name}: error link focuses the matching answer`, await page.locator("#example-name").evaluate((node) => node === document.activeElement))
    const errorsAxe = await new AxeBuilder({ page }).analyze()
    check(`${name}: linked-error form passes axe`, errorsAxe.violations.length === 0)
    await page.locator("#example-name").fill("Example Visitor")
    await page.locator("#example-email").fill("visitor@example.test")
    await page.locator("#example-message").fill("An example message.")
    await page.getByRole("button", { name: "Try this form" }).click()
    await page.getByRole("heading", { name: "Your example is complete." }).waitFor()
    check(`${name}: local demo confirms without posting`, posted.length === 0)
    await page.getByRole("button", { name: "Try another example" }).click()
    await page.getByRole("button", { name: "Conditional questions Pro" }).click()
    await page.locator("#example-request").selectOption("website")
    check(`${name}: conditional question appears`, await page.locator("#example-budget").isVisible())
    check(`${name}: conditional question is announced`, await page.getByRole("status").filter({ hasText: "A project budget question is now available." }).isVisible())
    await page.locator("#example-request").selectOption("general")
    check(`${name}: irrelevant question is removed`, await page.locator("#example-budget").count() === 0)
    page.off("request", collect)
    await page.getByRole("navigation", { name: "Store navigation" }).getByRole("link", { name: "Features" }).click()
    await page.waitForURL("**/products/accessible-forms#features")
    await page.waitForFunction(() => document.activeElement?.id === "features")
    check(`${name}: feature deep link moves focus`, await page.locator("#features").evaluate((node) => node === document.activeElement))
    await page.goto(base + "/pricing")
    await page.getByRole("status").filter({ hasText: "purchasing is currently unavailable" }).waitFor()
    check(`${name}: unconfigured checkout stays closed`, await page.locator('a[href^="/pay/"]').count() === 0)
    check(`${name}: Free is available before paid launch`, await page.getByRole("link", { name: "Download Free", exact: true }).isVisible())
    await page.route("**/api/v1/products/accessible-forms-pro/plans", (route) => route.fulfill({ json: { success: true, data: { plans } } }))
    await page.route("**/ready", (route) => route.fulfill({ json: { ready: true } }))
    await page.reload()
    await page.getByRole("link", { name: "Choose Personal" }).waitFor()
    check(`${name}: Stripe amounts and periods render`, (await page.locator("main").innerText()).includes("$99.00") && (await page.locator("main").innerText()).includes("$14.99"))
    check(`${name}: zero-decimal price and unlimited sites render`, (await page.locator("main").innerText()).includes("5,000") && (await page.locator("main").innerText()).includes("Unlimited sites"))
    check(`${name}: checkout uses public plan ID`, await page.getByRole("link", { name: "Choose Personal" }).getAttribute("href") === "/pay/public-personal")
    await page.getByRole("button", { name: "Annual", exact: true }).click()
    check(`${name}: annual filter preserves the Free option`, await page.locator('a[href^="/pay/"]').count() === 1 && await page.getByRole("link", { name: "Download Free", exact: true }).isVisible())
    await page.getByRole("button", { name: "One-time", exact: true }).click()
    check(`${name}: one-time filter chooses the lifetime plan`, await page.getByRole("link", { name: "Choose Lifetime" }).isVisible() && await page.locator('a[href^="/pay/"]').count() === 1)
    await page.getByRole("button", { name: "All plans" }).click()
    const readyAxe = await new AxeBuilder({ page }).analyze()
    check(`${name}: configured pricing passes axe`, readyAxe.violations.length === 0)
    await page.route("**/api/v1/products/accessible-forms-pro/plans", (route) => route.fulfill({ status: 503, json: { success: false } }))
    await page.reload()
    await page.getByRole("alert").filter({ hasText: "Pro plans could not be loaded" }).waitFor()
    check(`${name}: failed catalog still offers Free`, await page.getByRole("link", { name: "Download Free", exact: true }).isVisible())
    check(`${name}: failed catalog offers a retry`, await page.getByRole("button", { name: "Try again", exact: true }).isVisible())
    const response = await context.request.get(base + "/downloads/accessible-forms-by-accessible-org-1.0.0.zip")
    check(`${name}: public installer has reviewed checksum`, response.ok() && createHash("sha256").update(await response.body()).digest("hex") === "5ae230e4679d8173916af5097836f6bb7ed24693ce75efb2cb0731dfc8486bec")
    check(`${name}: public installer is served as an attachment`, response.headers()["content-disposition"]?.startsWith("attachment;"))
    const missing = await context.request.get(base + "/downloads/missing.zip")
    check(`${name}: missing installer is a 404`, missing.status() === 404)
    const user = { id: "fixture-customer", email: "visitor@example.test", name: "Example Visitor", avatar_url: "", is_admin: false, role: "user" }
    await page.route("**/api/v1/portal/me", (route) => route.fulfill({ json: { success: true, data: user } }))
    await page.route("**/api/v1/portal/licenses", (route) => route.fulfill({ json: { success: true, data: { licenses: [], renewals_enabled: true } } }))
    await page.goto(base + "/portal")
    await page.getByRole("link", { name: "Browse products" }).waitFor()
    check(`${name}: an empty account links to the store`, await page.getByRole("link", { name: "Browse products" }).getAttribute("href") === "/")
    const license = { id: "fixture-license", email: user.email, license_key: "EXAMPLE-KEY-NOT-A-REAL-LICENSE", product_id: "fixture-product", plan_id: "fixture-plan", status: "active", valid_from: "2026-01-01T00:00:00Z", valid_until: "2099-01-01T00:00:00Z", activations: [], product: { id: "fixture-product", name: "Accessible Forms Pro", slug: "accessible-forms-pro", type: "desktop" }, plan: { id: "fixture-plan", name: "Personal", license_type: "subscription", max_activations: 1, active: true } }
    await page.route("**/api/v1/portal/licenses/*/activations", (route) => route.fulfill({ json: { success: true, data: { activations: [], max: 1 } } }))
    await page.route("**/api/v1/portal/licenses", (route) => route.fulfill({ json: { success: true, data: { licenses: [license], renewals_enabled: true } } }))
    await page.reload()
    await page.getByRole("button", { name: "Download Accessible Forms Pro" }).waitFor()
    check(`${name}: the account connects product, Free installer and guide`, await page.getByRole("link", { name: "View product details" }).isVisible() && await page.getByRole("link", { name: "Download the Free plugin" }).isVisible() && await page.getByRole("link", { name: "Installation and help" }).isVisible())
    const portalAxe = await new AxeBuilder({ page }).analyze()
    if (portalAxe.violations.length) console.log(JSON.stringify(portalAxe.violations.map((v) => ({ id: v.id, nodes: v.nodes.map((n) => n.target) }))))
    check(`${name}: customer license overview passes axe`, portalAxe.violations.length === 0)
    await page.setViewportSize({ width: 320, height: 900 })
    check(`${name}: the account reflows at 320px`, await page.evaluate(() => document.documentElement.scrollWidth <= 320))
    await page.setViewportSize({ width: 1280, height: 900 })
    await page.route("**/api/v1/portal/downloads/wordpress", (route) => route.fulfill({ status: 403, json: { success: false, error: "Download unavailable" } }))
    await page.getByRole("button", { name: "Download Accessible Forms Pro" }).click()
    await page.getByRole("alert").filter({ hasText: "The installer could not be downloaded" }).waitFor()
    check(`${name}: a refused private download gives a useful error`, await page.getByRole("button", { name: "Download Accessible Forms Pro" }).isEnabled())
    await page.route("**/api/v1/checkout/verify?*", (route) => route.fulfill({ json: { success: true, data: { status: "ok", kind: "purchase", email: user.email } } }))
    await page.goto(base + "/checkout/success?session_id=fixture-session")
    await page.getByText("Your next steps", { exact: true }).waitFor()
    check(`${name}: a purchase shows installation steps`, await page.getByRole("link", { name: "Read the installation guide" }).isVisible())
    const checkoutAxe = await new AxeBuilder({ page }).analyze()
    if (checkoutAxe.violations.length) console.log(JSON.stringify(checkoutAxe.violations.map((v) => ({ id: v.id, nodes: v.nodes.map((n) => n.target) }))))
    check(`${name}: purchase confirmation passes axe`, checkoutAxe.violations.length === 0)
    await page.route("**/api/v1/checkout/verify?*", (route) => route.fulfill({ json: { success: true, data: { status: "ok", kind: "renewal", email: user.email } } }))
    await page.reload()
    await page.locator('a[href="/portal"]').waitFor()
    check(`${name}: a renewal returns to the account without initial-install steps`, await page.getByText("Your next steps", { exact: true }).count() === 0)
    check(`${name}: no client-side exceptions`, errors.length === 0)
    if (name === "chromium") {
      await page.goto(base + "/")
      await page.screenshot({ path: output + "/store-preview.png" })
      await page.screenshot({ path: output + "/store-desktop.png", fullPage: true })
      await page.setViewportSize({ width: 375, height: 900 })
      await page.screenshot({ path: output + "/store-mobile.png", fullPage: true })
      await page.setViewportSize({ width: 1280, height: 900 })
      await page.goto(base + "/products/accessible-forms-pro")
      await page.screenshot({ path: output + "/pro-product.png", fullPage: true })
    }
    report.engines.push({ name, checks: report.checks - start, failures: 0 })
  } finally { await browser.close() }
}
await writeFile(output + "/results.json", JSON.stringify(report, null, 2) + "\n")
console.log(`${report.checks} checks; 0 failures.`)
