# Accessible.org licensing development

This is the Accessible.org fork of [Keygate](https://github.com/tabloy/keygate).
The first branch adds a WordPress server adapter for Accessible Forms Pro.
The server remains a standalone Go application with PostgreSQL, hosted separately
from WordPress. The existing Keygate attribution, notices and license are retained.

## Local server

Install Docker with Compose. You do not need Go or Bun on the host to run the
complete application or backend tests.

```sh
cp .env.development.example .env.development
openssl rand -hex 32
openssl rand -hex 32
openssl rand -hex 32
```

Put the three different generated values in `JWT_SECRET`, `LICENSE_SIGNING_KEY`
and `RELEASE_KEY_ENCRYPTION_KEY` in `.env.development`. Keep this file private;
it is ignored by Git. Then:

```sh
make local-up
```

Open <http://localhost:9100> and complete the first-run setup with your local
admin email, store name and product. For Accessible Forms Pro, use:

- Product name: **Accessible Forms Pro**
- Product slug: **accessible-forms-pro**
- Product type: **Desktop** (Keygate's current model for installed software;
  it enables activations and releases)

Turn on the product's **Require license for update feeds** setting before
uploading any plugin releases. Site activations use the plan's maximum
activations as the site's license limit. Staging, subdirectories, `www` and
multisite installations currently consume their own slots.

The development stack builds this checkout. Only the application is published
to the host, on `127.0.0.1:9100`; databases stay inside its Docker network.
The development database persists across restarts. Payments, outbound SMTP
and artifact storage are disabled in this initial setup. With SMTP disabled,
login codes appear in `make local-logs`.

```sh
make local-down   # stop services and keep the development database
make local-up     # rebuild after changes
```

## Validation

```sh
make local-test
```

This runs every Go package against a separate, disposable PostgreSQL 18
database. The tests do not use the development database or Stripe credentials.
The test database lives in a temporary filesystem; stopping and recreating its
container resets it. Shared Go dependency/build caches speed subsequent runs.

For frontend changes, install Bun and run:

```sh
cd web
bun install --frozen-lockfile
bun run typecheck
bun run lint
bun run build
```

CI runs the frontend checks, Go vet, unit tests, all database integration tests
and a backend build. Backend packages run serially when sharing the test database.

## Next milestones

1. Integrate the [WordPress API](wordpress.md) into Accessible Forms Pro's
   license page and updater. Refresh download URLs during installation,
   verify the package hash, retain the core-version compatibility guard,
   and keep existing Pro form features working when a license expires.
2. Add private artifact storage and validate a real signed plugin ZIP upload,
   download and upgrade. Configure release-specific WordPress/PHP requirements
   before relying on unattended upgrades.
3. Connect Stripe **test mode** and exercise purchases, duplicate webhook
   delivery, renewal, cancellation, failed payment and refund. Choose the
   commercial model deliberately: subscription billing, or perpetual use with
   a renewable maintenance period.
4. Deploy the reviewed build at `https://license.accessible.org` with a private
   database/storage bucket, production secrets, SMTP, HTTPS, backups and
   health monitoring. Register the Stripe webhook against that public URL,
   complete the end-to-end tests, then enable live checkout.

Live payments and production deployment are separate milestones. This initial
branch is a development foundation, not a completed replacement for the
Accessible Forms Pro Freemius client.
