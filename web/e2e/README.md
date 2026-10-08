# Storefront browser checks

Start the development store with `compose.dev.yaml`. From this directory, run
`npm ci`, `npx playwright install chromium firefox webkit`, then `npm test`.
Set `STORE_BASE_URL` to use another **unconfigured test store**. The suite expects
paid checkout to be closed before intercepting the public catalogue and readiness
responses with browser-only plan fixtures; it does not make a purchase.

The checks cover all five public pages, axe accessibility, 320/375/768px reflow,
linked error focus, conditional question announcements, a local-only form demo,
Free availability before launch, currency display, plan filters, catalogue
failure/retry, actual installer checksums, attachment delivery and missing-file
responses. Chromium, Firefox and WebKit run the same checks. Screenshots and
`results.json` go to `tmp/storefront-checks/`; `STORE_CHECK_OUTPUT` overrides that.

The Free package fixture is the reviewed 1.0.0 installer. Change its expected
version and SHA-256 when deliberately replacing the public installer.
