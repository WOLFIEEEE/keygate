# Accessible.org licensing deployment

This folder runs the licensing and billing service for Accessible Forms Pro at `https://license.accessible.org`. It has one public page for the product description, comparison and payment, plus customer and admin screens for management. The main Accessible.org website remains separate from this utility. This service runs on an always-on Docker host because Keygate also runs payment recovery, email delivery, reconciliation and cleanup workers.

## What you supply

Use a Linux host with Docker Engine and the Compose plugin, persistent disk, and inbound TCP ports 80 and 443. Point the `license.accessible.org` DNS record at that host. Keep port 5432 private. A separate staging domain and database are recommended for Stripe test mode; a store refuses switching an existing database between Stripe test and live accounts.

Copy `.env.example` to `.env` and fill these values. Single-quote values containing `$`, `#` or spaces so Docker does not expand them. Internal secrets and product signing keys are generated automatically. No Stripe or server secrets go inside WordPress.

| Value | What to enter |
| --- | --- |
| `BOOTSTRAP_OWNER_EMAIL` | Your own address; receives the admin sign-in code |
| `STRIPE_SECRET_KEY` | Secret key for the intended Stripe account and mode; product, price, webhook, account-read and billing-portal permissions are needed |
| `BOOTSTRAP_PLANS_JSON` | Approved prices, billing periods, currency and site limits; see below |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | Your email provider's connection details and approved sender |
| `STRIPE_AUTOMATIC_TAX` | Optional Stripe Tax, after configuring it in the Stripe account |
| `BACKUP_REMOTE` | Optional rclone destination for encrypted off-site copies; place its credentials in `secrets/rclone.conf` |

`LICENSE_DOMAIN` is already `license.accessible.org`. Leave `STRIPE_WEBHOOK_SECRET` blank: the server creates its endpoint and retains the signing secret encrypted in PostgreSQL. A Stripe CLI forwarding session in local development has a separate secret. The `RELEASE_PUBLISH_KEY` prefix is Keygate's API-key format; Stripe mode is determined by the Stripe key.

Plans are a JSON array, quoted in the environment file. Amounts use the currency's smallest unit, as required by Stripe. `sites: 0` means unlimited sites. Intervals are `month`, `year` and `lifetime`; lifetime is a one-time purchase with updates for life. Subscription updates last through the paid period and any configured grace period. The following is an **example, not an approved Accessible.org price**:

```dotenv
BOOTSTRAP_PLANS_JSON='[{"slug":"personal","name":"Personal","sites":1,"amount":9900,"currency":"usd","interval":"year"}]'
```

## Start the store

The handoff includes the raw Pro installer in `artifacts/`. From the repository root:

```sh
./deployment/start.sh
```

The command validates configuration and checks SMTP connectivity without sending email, creates secrets, builds the service, obtains HTTPS certificates, provisions the owner, product, Stripe catalog and customer portal, uploads the installer, validates its WordPress headers, signs it, and checks `/ready`. The configured installer and its public receipt are saved in `deployment/artifacts/configured/`. Distribute **that configured installer** to customers. It contains the store address, permitted download origins and public verification keys. Distribute the Free installer separately or direct customers to its WordPress.org listing when available.

Once started:

- Product description, Free download and Pro plans: `https://license.accessible.org/`
- Direct link to payment options: `https://license.accessible.org/#store-plans`
- Customer licenses, installer downloads and billing: `https://license.accessible.org/portal`. The first download does not use a site activation slot; connect the key after installation.
- Admin: sign in with the configured owner email at `https://license.accessible.org/login`, then use the admin navigation.
- Configuration readiness: `https://license.accessible.org/ready`; process health: `/health`.

The single purchase page includes the verified Free 1.0.0 installer, concise product details, configured Pro plans, an expandable Free/Pro comparison and installation instructions. The customer portal links back to that page beside the protected Pro download. Product details reflect the packaged plugin source; prices and site limits come from the configured public plan catalogue. Purchasing remains closed until readiness succeeds, while the Free download stays available. Canceled Stripe purchases return to the plans on this page.

There are no separate public product, pricing, guide or SEO pages, social preview metadata or canonical links. The HTML and frontend responses request no indexing, and `robots.txt` disallows crawling. Old `/pricing`, `/guide` and product links only redirect to sections of the purchase page so existing plugin links and bookmarks continue to work. Keygate attribution remains on the purchase page, management screens and API. Administration and account pages load on demand, so purchase visitors do not download the analytics charts.

Checkout stays closed while prices, Stripe, email, the webhook or a signed deliverable are missing. A configured email provider still needs a real delivery check; SMTP authentication alone does not establish inbox delivery. Test checkout, purchase email, activation, renewal and refunds with your intended Stripe account before opening the paid store. Live billing was not exercised during local development.

## Daily database health check

The server automatically reads its migration table once after startup and every 24 hours while it is running, using the existing `DATABASE_URL`. Each check has a ten-second timeout. Search the server logs for `database_healthcheck`: successful checks have `status=ok`; failures have `status=error` and a safe reason (`timeout` or `query_failed`). Connection credentials and raw driver errors are never included in these log entries. Failed checks are logged and the next scheduled check still runs. Server shutdown cancels any check in progress.

This works with local PostgreSQL, Render PostgreSQL and Supabase PostgreSQL. For a Render app connected to Supabase, use the Session pooler connection string on port 5432 with TLS, keep it in the server's private environment, and disable Supabase's unused Data API. The Compose startup helper in this folder currently provisions its own PostgreSQL; a Render deployment must supply the external connection through its environment instead.

The schedule requires an always-on server; it does not run while the host is stopped or sleeping. The startup check runs again after a restart, followed by the next 24-hour schedule. It is a database health check, not a backup or a guarantee against free-project pausing. Supabase determines activity eligibility and says a few user database requests each day are typically sufficient, so a single daily check alone cannot guarantee that a Free project stays active. Monitor Supabase's pause notices and retain independent backups. See [Supabase's project-pausing policy](https://supabase.com/docs/guides/platform/free-project-pausing).

## Subsequent releases and configuration changes

When releasing a new Free plugin, replace the versioned ZIP and its adjacent source manifest in `web/public/downloads/`, update `web/src/lib/accessible-forms.ts` with its version, checksum and requirements, and rebuild the server. Keep the comparison and installation details consistent with released capabilities. The public Free ZIP is independent of the licensed Pro artifact and must never contain private configuration or publishing keys. Update `web/e2e/storefront.mjs`'s installer checksum/version when replacing this reviewed package.

Build a new Pro ZIP with its version header, runtime constant, stable tag, translation metadata and changelog updated together. Use a new version for changed bytes; published artifacts are immutable. Copy it into `deployment/artifacts/`, then:

```sh
cd deployment
docker compose --profile tools run --rm publisher artifacts/accessible-forms-pro-1.0.1.zip --notes /deployment/release-notes.md
```

The publisher repeats safely after interruption. It never embeds its publishing credential. The WordPress updater obtains a fresh authorization for each installation, checks SHA-256 and Ed25519 signatures with pinned keys, and checks WordPress, PHP and the Free plugin's extension API before replacing the working plugin. An expired license or temporary server outage leaves installed Pro features available.

Run `start.sh` again after configuration changes. Existing catalog resources are reused. Once a plan has customers, change its price or site limit by creating a **new plan slug**; existing customers retain their terms. Omitting an old plan disables it for new purchases. Do not rotate product signing keys directly: existing plugin installers pin those keys. Key rotation requires a transition release that trusts the next key before activating it. Keep the master encryption key unchanged unless performing an explicit data re-encryption migration.

To deploy a new server revision, build from the fork and restart with `docker compose up -d --build --wait keygate`. This deployment points update notices to the fork, preserving the WordPress adapter. Do not replace it with an upstream binary lacking the adapter.

## Backups and recovery

The worker encrypts a PostgreSQL dump, private ZIPs and the configuration daily, retries failures every five minutes, and retains fourteen days locally. If `BACKUP_REMOTE` is set, it copies each encrypted archive off-site. Local backups alone do not survive loss of the server disk. Keep `.env`, `secrets/backup-age-key.txt` and any rclone credentials in your password manager or recovery vault, away from the same host.

Trigger and inspect a backup:

```sh
cd deployment
docker compose run --rm backup --once
docker compose logs --tail=50 backup
```

Recovery replaces the store database, artifacts and configuration, so run it only for an intended recovery. Place a trusted encrypted archive under `backups/` and the original recovery identity at `secrets/backup-age-key.txt`, then:

```sh
./deployment/restore.sh backups/TIMESTAMP.tar.age --confirm-replace
```

On a replacement host, copy the original `.env` first so Compose can resolve the database credentials. The helper decrypts before stopping the application, restores the database in one transaction, restores artifacts, and restarts the services. Check `/ready`, admin sign-in and a licensed download afterward. Local development successfully restored a real encrypted backup into a separate database; a production-host recovery still depends on its credentials and storage.

## Source and attribution

The server fork is AGPL-3.0 and retains the upstream `Powered by Keygate` notices and API attribution. Its source, `LICENSE` and `NOTICE` are included. The WordPress client is independently authored GPL-2.0-or-later code and contains no copied Keygate implementation or Freemius SDK.
