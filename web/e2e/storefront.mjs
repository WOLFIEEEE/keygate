import assert from "node:assert/strict"
import { createHash } from "node:crypto"
import { mkdir, writeFile } from "node:fs/promises"
import AxeBuilder from "@axe-core/playwright"
import { chromium, firefox, webkit } from "playwright"

const base = process.env.STORE_BASE_URL || "http://localhost:9100"
const output = process.env.STORE_CHECK_OUTPUT || "../../tmp/storefront-checks"
await mkdir(output, { recursive: true })
const report = { base, publicPages: ["/"], engines: [], checks: 0 }
function check(label, value) { assert(value, label); report.checks++; console.log(`PASS ${label}`) }
async function accessible(page, label) {
  const axe = await new AxeBuilder({ page }).analyze()
  if (axe.violations.length) console.log(JSON.stringify(axe.violations.map((v) => ({ id: v.id, nodes: v.nodes.map((n) => n.target) }))))
  check(label, axe.violations.length === 0)
}
const plans = [
  { id: "annual", name: "Single site", price: 2900, currency: "usd", max_sites: 1, license_type: "subscription", billing_interval: "year", checkout_id: "plannual" },
  { id: "monthly", name: "Team", price: 1499, currency: "usd", max_sites: 5, license_type: "subscription", billing_interval: "month", checkout_id: "plmonth1" },
  { id: "once", name: "Lifetime", price: 5000, currency: "jpy", max_sites: 0, license_type: "perpetual", billing_interval: "", checkout_id: "pllifet1" },
]
const legacyLinks = {
  "/pricing": "/#store-plans",
  "/pricing/": "/#store-plans",
  "/guide": "/#installation",
  "/products/accessible-forms": "/",
  "/products/accessible-forms-pro": "/#comparison",
}

for (const [name, engine] of Object.entries({ chromium, firefox, webkit })) {
  const start = report.checks
  const browser = await engine.launch()
  const context = await browser.newContext({ locale: "en-US", viewport: { width: 1280, height: 900 }, reducedMotion: "reduce" })
  const page = await context.newPage()
  const errors = []
  page.on("pageerror", (error) => errors.push(error.message))
  try {
    const root = await page.goto(base + "/")
    await page.getByRole("heading", { level: 1, name: "Accessible Forms", exact: true }).waitFor()
    await page.getByRole("status").filter({ hasText: "purchasing is currently unavailable" }).waitFor()
    await page.getByRole("heading", { name: "Single site", exact: true }).waitFor()
    check(`${name}: approved offer is $29 per year for one site`, (await page.locator(".af-plan-price").innerText()).includes("$29.00") && (await page.locator(".af-plan-price").innerText()).includes("/ year") && (await page.locator(".af-plan-sites").innerText()) === "1 site")
    check(`${name}: approved offer renews at the same price and stays disabled before setup`, (await page.locator(".af-plan-description").innerText()).includes("Renews at $29.00 per year") && await page.getByRole("button", { name: "Purchasing unavailable", exact: true }).isDisabled())
    check(`${name}: one product heading and visible attribution`, await page.locator("h1").count() === 1 && await page.getByRole("link", { name: "Powered by Keygate" }).isVisible())
    check(`${name}: header only links home and account`, await page.locator("header a").count() === 2 && await page.getByRole("link", { name: "My account", exact: true }).first().getAttribute("href") === "/portal")
    check(`${name}: no links to separate public pages`, await page.locator('a[href^="/products/"], a[href="/guide"], a[href="/pricing"]').count() === 0)
    check(`${name}: search indexing is disabled`, (await page.locator('meta[name="robots"]').getAttribute("content")) === "noindex, nofollow" && root.headers()["x-robots-tag"] === "noindex, nofollow")
    check(`${name}: no search/social metadata or canonical links`, await page.locator('meta[name="description"], meta[property^="og:"], meta[name^="twitter:"], link[rel="canonical"]').count() === 0)
    const robots = await context.request.get(base + "/robots.txt")
    check(`${name}: robots excludes the utility`, robots.ok() && (await robots.text()).trim() === "User-agent: *\nDisallow: /")
    check(`${name}: unconfigured checkout is closed and Free available`, await page.locator('a[href^="/pay/"]').count() === 0 && await page.getByRole("link", { name: "Download Free", exact: true }).isVisible())
    check(`${name}: optional details start collapsed`, await page.locator("#comparison").evaluate((n) => !n.open) && await page.locator("#installation").evaluate((n) => !n.open))
    await accessible(page, `${name}: compact page passes axe`)
    if (name === "chromium") {
      await page.screenshot({ path: output + "/purchase-preview.png" })
      await page.screenshot({ path: output + "/purchase-desktop.png", fullPage: true })
      await page.setViewportSize({ width: 375, height: 900 })
      await page.screenshot({ path: output + "/purchase-mobile.png", fullPage: true })
      await page.setViewportSize({ width: 1280, height: 900 })
    }
    await page.locator("#comparison summary").focus()
    await page.keyboard.press("Enter")
    check(`${name}: keyboard opens the comparison`, await page.locator("#comparison").evaluate((n) => n.open))
    check(`${name}: comparison labels columns and six feature rows`, await page.getByRole("columnheader").count() === 3 && await page.locator(".af-comparison tbody tr").count() === 6)
    await page.locator("#installation summary").click()
    check(`${name}: installation expands all three steps`, await page.locator("#installation li").count() === 3 && await page.locator("#installation").evaluate((n) => n.open))
    await accessible(page, `${name}: expanded details pass axe`)
    for (const width of [320, 375, 768]) {
      await page.setViewportSize({ width, height: 900 })
      check(`${name}: expanded page reflows at ${width}px`, await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth))
    }
    await page.setViewportSize({ width: 1280, height: 900 })
    await page.evaluate(() => { document.body.style.zoom = "2" })
    check(`${name}: page reflows at 200% zoom`, await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth))
    await page.evaluate(() => { document.body.style.zoom = "" })
    for (const [path, target] of Object.entries(legacyLinks)) {
      const response = await context.request.get(base + path, { maxRedirects: 0 })
      check(`${name}: ${path} only redirects`, response.status() === 302 && response.headers().location === target)
      await page.goto(base + path)
      await page.waitForURL(base + target)
      await page.getByRole("heading", { level: 1, name: "Accessible Forms", exact: true }).waitFor()
      const hash = target.split("#")[1]
      if (hash) {
        await page.waitForFunction((id) => document.activeElement?.id === id, hash)
        check(`${name}: ${path} focuses its destination`, await page.locator(`#${hash}`).evaluate((n) => n === document.activeElement && (!(n instanceof HTMLDetailsElement) || n.open)))
      }
    }
    await page.goto(base + "/#%E0")
    await page.getByRole("heading", { level: 1, name: "Accessible Forms", exact: true }).waitFor()

    let ready = false
    let catalogStatus = 200
    let publicPlans = []
    await page.route("**/api/v1/products/accessible-forms-pro/plans", (route) => route.fulfill({ status: catalogStatus, json: catalogStatus === 200 ? { success: true, data: { plans: publicPlans } } : { success: false } }))
    await page.route("**/ready", (route) => route.fulfill({ status: ready ? 200 : 503, json: { ready } }))
    ready = true
    await page.goto(base + "/")
    await page.getByText("No Pro plans are available at the moment. Please check again later.", { exact: true }).waitFor()
    check(`${name}: readiness alone cannot authorize the preset offer`, await page.locator('a[href^="/pay/"]').count() === 0 && await page.locator(".af-plan-card").count() === 0)
    publicPlans = [plans[0]]
    await page.reload()
    await page.getByRole("link", { name: "Choose Single site" }).waitFor()
    check(`${name}: published annual offer uses Stripe catalogue terms`, await page.locator('a[href^="/pay/"]').count() === 1 && await page.getByRole("link", { name: "Choose Single site" }).getAttribute("href") === "/pay/plannual" && (await page.locator(".af-plan-price").innerText()).includes("$29.00"))
    if (name === "chromium") await page.screenshot({ path: output + "/configured-single-plan-fixture.png", fullPage: true })
    ready = false
    publicPlans = plans
    await page.goto(base + "/")
    await page.getByRole("button", { name: "Purchasing unavailable", exact: true }).first().waitFor()
    check(`${name}: catalog alone cannot enable payment`, await page.locator('a[href^="/pay/"]').count() === 0 && await page.getByRole("button", { name: "Purchasing unavailable", exact: true }).count() === 3)
    ready = true
    await page.getByRole("button", { name: "Check availability", exact: true }).click()
    await page.getByRole("link", { name: "Choose Single site" }).waitFor()
    check(`${name}: availability retry enables configured plans`, await page.locator('a[href^="/pay/"]').count() === 3)
    check(`${name}: decimal currencies and billing periods render`, (await page.locator("main").innerText()).includes("$29.00") && (await page.locator("main").innerText()).includes("$14.99"))
    check(`${name}: zero-decimal price and unlimited sites render`, (await page.locator("main").innerText()).includes("5,000") && (await page.locator("main").innerText()).includes("Unlimited sites"))
    check(`${name}: checkout uses the public plan ID`, await page.getByRole("link", { name: "Choose Single site" }).getAttribute("href") === "/pay/plannual")
    await page.getByRole("button", { name: "Annual", exact: true }).click()
    check(`${name}: annual filter preserves Free`, await page.locator('a[href^="/pay/"]').count() === 1 && await page.getByRole("link", { name: "Download Free", exact: true }).isVisible())
    await page.getByRole("button", { name: "Monthly", exact: true }).click()
    check(`${name}: monthly filter selects Team`, await page.getByRole("link", { name: "Choose Team" }).isVisible() && await page.locator('a[href^="/pay/"]').count() === 1)
    await page.getByRole("button", { name: "One-time", exact: true }).click()
    check(`${name}: one-time filter selects Lifetime`, await page.getByRole("link", { name: "Choose Lifetime" }).isVisible() && await page.locator('a[href^="/pay/"]').count() === 1)
    await page.getByRole("button", { name: "All plans", exact: true }).click()
    await accessible(page, `${name}: configured page passes axe`)
    for (const width of [320, 375, 768]) {
      await page.setViewportSize({ width, height: 900 })
      check(`${name}: configured plans reflow at ${width}px`, await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth))
    }
    await page.setViewportSize({ width: 1280, height: 900 })
    if (name === "chromium") await page.screenshot({ path: output + "/configured-plans-fixture.png", fullPage: true })
    publicPlans = [...plans, { ...plans[0], id: "incomplete", name: "Unpriced", price: null, checkout_id: "plunpr01" }]
    await page.reload()
    await page.getByRole("heading", { name: "Unpriced", exact: true }).waitFor()
    check(`${name}: incomplete prices cannot be purchased`, await page.getByRole("link", { name: "Choose Unpriced" }).count() === 0 && await page.getByRole("button", { name: "Purchasing unavailable", exact: true }).isDisabled())
    catalogStatus = 503
    await page.reload()
    await page.getByRole("alert").filter({ hasText: "Pro plans could not be loaded" }).waitFor()
    check(`${name}: catalog failure preserves Free and account`, await page.getByRole("link", { name: "Download Free", exact: true }).isVisible() && await page.getByRole("link", { name: "My account", exact: true }).first().isVisible())
    publicPlans = plans
    catalogStatus = 200
    await page.getByRole("button", { name: "Try again", exact: true }).click()
    await page.getByRole("link", { name: "Choose Single site" }).waitFor()
    check(`${name}: catalog recovers on retry`, await page.getByRole("alert").count() === 0 && await page.locator('a[href^="/pay/"]').count() === 3)

    const installer = await context.request.get(base + "/downloads/accessible-forms-by-accessible-org-1.0.0.zip")
    check(`${name}: Free installer has the reviewed checksum`, installer.ok() && createHash("sha256").update(await installer.body()).digest("hex") === "5ae230e4679d8173916af5097836f6bb7ed24693ce75efb2cb0731dfc8486bec")
    check(`${name}: Free installer is an attachment`, installer.headers()["content-disposition"]?.startsWith("attachment;"))
    const missing = await context.request.get(base + "/downloads/missing.zip")
    check(`${name}: missing installer is a 404`, missing.status() === 404)
    const user = { id: "fixture-customer", email: "visitor@example.test", name: "Example Visitor", avatar_url: "", is_admin: false, role: "user" }
    await page.route("**/api/v1/portal/me", (route) => route.fulfill({ json: { success: true, data: user } }))
    await page.route("**/api/v1/portal/licenses", (route) => route.fulfill({ json: { success: true, data: { licenses: [], renewals_enabled: true } } }))
    await page.goto(base + "/portal")
    await page.getByRole("link", { name: "View product and plans", exact: true }).waitFor()
    check(`${name}: empty account returns to the purchase page`, await page.getByRole("link", { name: "View product and plans", exact: true }).getAttribute("href") === "/")
    const license = { id: "fixture-license", email: user.email, license_key: "EXAMPLE-KEY-NOT-A-REAL-LICENSE", product_id: "fixture-product", plan_id: "fixture-plan", status: "active", valid_from: "2026-01-01T00:00:00Z", valid_until: "2099-01-01T00:00:00Z", activations: [], product: { id: "fixture-product", name: "Accessible Forms Pro", slug: "accessible-forms-pro", type: "desktop" }, plan: { id: "fixture-plan", name: "Single site", license_type: "subscription", max_activations: 1, active: true } }
    await page.route("**/api/v1/portal/licenses/*/activations", (route) => route.fulfill({ json: { success: true, data: { activations: [], max: 1 } } }))
    await page.route("**/api/v1/portal/licenses", (route) => route.fulfill({ json: { success: true, data: { licenses: [license], renewals_enabled: true } } }))
    await page.reload()
    await page.getByRole("button", { name: "Download Accessible Forms Pro" }).waitFor()
    check(`${name}: account links to inline product and installation details`, await page.getByRole("link", { name: "Product and plans", exact: true }).getAttribute("href") === "/" && await page.getByRole("link", { name: "Installation instructions", exact: true }).getAttribute("href") === "/#installation")
    check(`${name}: account keeps Free and protected Pro downloads`, await page.getByRole("link", { name: "Download the Free plugin" }).isVisible())
    await accessible(page, `${name}: license overview passes axe`)
    await page.setViewportSize({ width: 320, height: 900 })
    check(`${name}: account reflows at 320px`, await page.evaluate(() => document.documentElement.scrollWidth <= 320))
    await page.setViewportSize({ width: 1280, height: 900 })
    await page.route("**/api/v1/portal/downloads/wordpress", (route) => route.fulfill({ status: 403, json: { success: false, error: "Download unavailable" } }))
    await page.getByRole("button", { name: "Download Accessible Forms Pro" }).click()
    await page.getByRole("alert").filter({ hasText: "The installer could not be downloaded" }).waitFor()
    check(`${name}: refused private download gives a useful error`, await page.getByRole("button", { name: "Download Accessible Forms Pro" }).isEnabled())
    await page.route("**/api/v1/checkout/verify?*", (route) => route.fulfill({ json: { success: true, data: { status: "ok", kind: "purchase", email: user.email } } }))
    await page.goto(base + "/checkout/success?session_id=fixture-session")
    await page.getByText("Your next steps", { exact: true }).waitFor()
    check(`${name}: purchase has setup steps and same-page help`, await page.locator("main ol li").count() === 3 && await page.getByRole("link", { name: "Installation instructions" }).getAttribute("href") === "/#installation")
    await accessible(page, `${name}: purchase confirmation passes axe`)
    await page.route("**/api/v1/checkout/verify?*", (route) => route.fulfill({ json: { success: true, data: { status: "ok", kind: "renewal", email: user.email } } }))
    await page.reload()
    await page.locator('a[href="/portal"]').waitFor()
    check(`${name}: renewal returns to management without installation steps`, await page.getByText("Your next steps", { exact: true }).count() === 0)
    await page.goto(base + "/checkout/success")
    await page.getByRole("link", { name: "Back to product and plans", exact: true }).waitFor()
    check(`${name}: failed verification offers account and purchase-page recovery`, await page.getByRole("link", { name: "Back to product and plans", exact: true }).getAttribute("href") === "/" && await page.getByRole("link", { name: "Open your account", exact: true }).isVisible())
    check(`${name}: no client-side exceptions`, errors.length === 0)
    report.engines.push({ name, checks: report.checks - start, failures: 0 })
  } catch (error) {
    report.engines.push({ name, checks: report.checks - start, failures: 1, error: String(error), clientErrors: errors })
    throw error
  } finally {
    await writeFile(output + "/results.json", JSON.stringify(report, null, 2) + "\n")
    await browser.close()
  }
}
console.log(`${report.checks} checks; 0 failures.`)
