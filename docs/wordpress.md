# WordPress adapter

The fork adds a server API for installed WordPress plugins. It reuses Keygate's
licenses, activation limits, Stripe lifecycle and release maintenance cutoffs.
No additional WordPress licensing-server plugin is needed.

## Product and release configuration

Use a `desktop` or `hybrid` product so activations and releases are enabled.
For Accessible Forms Pro, use slug `accessible-forms-pro`. Turn on
`feed_license_required` before publishing releases and keep the artifact
bucket private: the generic desktop feeds can otherwise serve public package
links. A bounded maintenance plan also requires this setting and the existing
Keygate maintenance feature switch. Existing public feed URLs must expire
before a maintenance cutoff can be enforced; configure this on a new product
before publishing.

Upload the plugin ZIP as a release artifact with platform **wordpress**.
Use stable semantic versions such as `1.0.0`, and preserve the normal
`accessible-forms-pro/` directory inside the ZIP. These endpoints offer only
published stable releases that have an uploaded WordPress artifact; draft,
yanked, beta and desktop-only releases are not offered.

Keep release signing enabled and configure the product's signing key. The
normal release publication workflow computes the file hash and signs it.
The WordPress client must check the hash from freshly authorized metadata;
the separate client/signature-verification integration remains a next milestone.

## Requests

All five routes are `POST` under:

```text
/api/v1/wordpress/{product_slug}
```

Send JSON. The license key is the credential, and it stays in the request body.
Do not embed a merchant API key or put a license key in a URL. Use HTTPS in
production. Responses use Keygate's usual success/error envelope and carry
`Cache-Control: private, no-store`.

| Route | Purpose | Required body fields |
|---|---|---|
| `/activate` | Claim a site's slot; repeated activation reuses it | `license_key`, `site_url` |
| `/verify` | Check an already activated site; return plan, validity and token | `license_key`, `site_url` |
| `/deactivate` | Release this site's slot | `license_key`, `site_url` |
| `/update` | Find a newer stable version inside the license's update period | `license_key`, `site_url`, `version` (installed version) |
| `/download` | Recheck entitlement and get a fresh URL for an exact version | `license_key`, `site_url`, `version` (requested version) |

Use the site's `home_url('/')` consistently. Each normalized URL represents one
installation. HTTP-to-HTTPS changes, hostname casing, default ports and trailing
slashes reuse the slot. `www`, subdirectories, staging subdomains and
nonstandard ports remain separate. URLs may not contain credentials, query
parameters or fragments; the server never fetches the site URL. The activation
record contains a bounded hash identifier and a readable site URL label.

The WordPress routes share the existing license API's rate-limit budget and
brute-force guard. Bodies are capped at 16 KiB. Activation retries are safe
through the existing atomic activation store, without an idempotency header.
Product scoping ensures a key sold for another product cannot be used here.

Example activation request:

```sh
curl http://localhost:9100/api/v1/wordpress/accessible-forms-pro/activate \
  -H 'Content-Type: application/json' \
  --data '{"license_key":"YOUR_LOCAL_TEST_KEY","site_url":"https://example.test/"}'
```

Example update response:

```json
{
  "success": true,
  "data": {
    "update_available": true,
    "update": {
      "name": "Accessible Forms Pro",
      "slug": "accessible-forms-pro",
      "version": "1.1.0",
      "url": "https://accessible.org/forms/",
      "package": "https://private-storage.example/signed-package-url",
      "package_expires_at": "2026-10-08T12:10:00Z",
      "sha256": "64-character-sha256-of-the-zip",
      "release_notes": "Improved accessible validation."
    },
    "meta": {"server": "Keygate", "url": "https://keygate.app"}
  }
}
```

With no newer entitled release, the response is successful with
`update_available: false` and `update: null`. An expired maintenance period
still allows older releases published inside that period. A suspended,
revoked, expired or unactivated license receives `404 LICENSE_NOT_FOUND`
when checking or downloading. Pinning a release beyond the maintenance
period receives `403 UPDATES_EXPIRED`. Canceled subscriptions remain usable
until the end of the paid period, according to the existing Keygate rules.

## WordPress client contract

The metadata fields `slug`, `version`, `url` and `package` fit
[`update_plugins_{$hostname}`](https://developer.wordpress.org/reference/hooks/update_plugins_hostname/).
The Pro plugin should retain its own `Update URI` header and scope the filter
to its exact plugin file. Render release notes as escaped/sanitized content.
Release-specific WordPress/PHP compatibility metadata and the details modal
are not provided by this initial API.

WordPress caches update checks longer than a short-lived storage URL. At
[`upgrader_pre_download`](https://developer.wordpress.org/reference/hooks/upgrader_pre_download/),
the client must call `/download` with the **exact cached version** and its
stored license/site, download the newly authorized package, verify its
SHA-256, and return the validated local ZIP path. Refuse the upgrade when
fresh authorization or integrity verification fails. Do not retry the cached
package URL or add the license key to it. This also rechecks a revocation or
deactivation that happened after the original update check.

The license API grants updates and support. Keep existing Pro form features
working on expiry or a network failure. A site URL is a client-reported
identity; it is not proof of domain ownership or an anti-tampering mechanism.
Staging allowances, multisite policy and automated migration from Freemius
are deliberately left for the client-integration milestone.
