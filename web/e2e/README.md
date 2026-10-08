# Purchase and account browser checks

Start the development store with `compose.dev.yaml`. From this directory, run
`npm ci`, `npx playwright install chromium firefox webkit`, then `npm test`.
Set `STORE_BASE_URL` to use another **unconfigured test store**. The suite expects
paid checkout to be closed before intercepting the public catalogue and readiness
responses with browser-only plan fixtures; it does not make a purchase.

The checks cover the single public purchase page, axe accessibility,
320/375/768px reflow, 200% zoom, keyboard-operated comparison and installation
details, old-link redirects and focus, disabled search indexing, Free availability
before launch, readiness gating, currency display, plan filters, catalogue
failure and recovery, installer checksums, attachment delivery and missing-file
responses. Customer licenses, refused private downloads, purchase and renewal
confirmation, and verification-error recovery are also checked.
Chromium, Firefox and WebKit run the same checks. Screenshots and
`results.json` go to `tmp/storefront-checks/`; `STORE_CHECK_OUTPUT` overrides that.

`purchase-*.png` show the actual unconfigured local service.
`configured-plans-fixture.png` uses simulated prices only for layout testing;
these are not approved product prices and do not enable a real Stripe purchase.

The Free package fixture is the reviewed 1.0.0 installer. Change its expected
version and SHA-256 when deliberately replacing the public installer.
