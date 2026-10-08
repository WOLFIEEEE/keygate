// Product details reflect the packaged WordPress plugins at source commit
// 17e123d93a6b763061c3b082408a3fc772c90b4a. Only include released capabilities.
import type { PublicPlan } from "@/hooks/use-store-catalog"

export const approvedProPlan: PublicPlan = {
  id: "single-site-annual-preset",
  name: "Single site",
  license_type: "subscription",
  billing_interval: "year",
  max_sites: 1,
  price: 2900,
  currency: "usd",
  // An approved offer is displayed before configuration. Only the published
  // catalogue can supply a real checkout ID and authorize a payment link.
  checkout_id: "",
}

export const freeDownload = {
  version: "1.0.0",
  url: "/downloads/accessible-forms-by-accessible-org-1.0.0.zip",
  sha256: "5ae230e4679d8173916af5097836f6bb7ed24693ce75efb2cb0731dfc8486bec",
  wordpress: "6.4",
  php: "8.1",
}

export const featureGroups = [
  {
    id: "build",
    title: "Form builder",
    free: "Unlimited forms, 13 field types, templates and help text.",
    pro: "Conditional questions, steps, private uploads, review, save and resume, import/export.",
  },
  {
    id: "style",
    title: "Styling",
    free: "Four themes, contrast checks and protected focus styles.",
    pro: "Custom colors, spacing, field layouts, label positions and brand presets.",
  },
  {
    id: "publish",
    title: "Publishing",
    free: "Drafts, published versions, block, shortcode and guided placement.",
    pro: "Scheduling, submission limits, placement checker and version history.",
  },
  {
    id: "deliver",
    title: "Notifications",
    free: "WordPress mail, test emails, staging inbox and delivery retry.",
    pro: "SMTP, conditional notifications, visitor receipts and signed webhooks.",
  },
  {
    id: "entries",
    title: "Entries",
    free: "Search, filters, review state and bulk actions.",
    pro: "Assignments, follow-up dates, notes, replies, CSV exports and Insights.",
  },
  {
    id: "privacy",
    title: "Access & privacy",
    free: "Permissions, retention, data export/erasure, honeypot and rate limits.",
    pro: "Team roles, per-form entry access, Turnstile and blocklists.",
  },
] as const
