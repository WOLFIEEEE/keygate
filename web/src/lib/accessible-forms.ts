// Product copy is grounded in Accessible Forms' README and the Pro readme at
// source commit 17e123d93a6b763061c3b082408a3fc772c90b4a. Keep this in step with
// the packaged plugins; roadmap items do not belong in the public catalogue.
export const freeDownload = {
  version: "1.0.0",
  url: "/downloads/accessible-forms-by-accessible-org-1.0.0.zip",
  sha256: "5ae230e4679d8173916af5097836f6bb7ed24693ce75efb2cb0731dfc8486bec",
  wordpress: "6.4",
  php: "8.1",
}

export const coreFieldTypes = [
  "Text",
  "Email",
  "Telephone",
  "URL",
  "Number",
  "Date",
  "Time",
  "Section heading",
  "Message",
  "Dropdown",
  "Radio group",
  "Checkbox group",
  "Consent checkbox",
] as const

export const featureGroups = [
  {
    id: "build",
    title: "Ask the right questions",
    description: "Start with a short contact form. Add structure when the conversation needs more room.",
    free: "Unlimited forms, 13 field types, starter templates, help text and keyboard field reordering.",
    pro: "Private file uploads, conditional questions, named steps, final review, save and resume, reusable templates and form import/export.",
  },
  {
    id: "style",
    title: "Make it feel like your website",
    description: "Keep labels, focus and contrast clear while choosing a look that fits your brand.",
    free: "Four built-in themes: Classic light, Ocean blue, Forest green and Midnight. Contrast checks and protected focus styles.",
    pro: "Custom colors, text size, spacing, corners, field layouts, label positions and reusable brand presets.",
  },
  {
    id: "publish",
    title: "Publish with a clear review step",
    description: "Keep draft changes separate from the form your visitors are already using.",
    free: "Separate drafts and published versions, an Accessible Form block, shortcodes and guided placement in pages or posts.",
    pro: "Automatic placement, a placement checker, opening and closing dates, submission limits and version history.",
  },
  {
    id: "deliver",
    title: "Get each message to the right place",
    description: "Preview notifications and test delivery before collecting real messages.",
    free: "A main notification for each form, WordPress mail, test emails, a staging inbox, background delivery and retry.",
    pro: "Custom SMTP with optional mail fallback, formatted and conditional notifications, Bcc, visitor receipts and signed webhooks.",
  },
  {
    id: "entries",
    title: "Turn submissions into a workflow",
    description: "Find the message you need, see what needs attention and give your team a useful next step.",
    free: "Search, filters, read state, Needs review, Spam, Trash, entry details, bulk actions and supported Undo.",
    pro: "Assignments, follow-up dates, progress status, private notes, saved views, replies, background CSV exports and form Insights.",
  },
  {
    id: "privacy",
    title: "Keep control of access and retention",
    description: "Manage submissions in WordPress and decide who can see them and how long to keep them.",
    free: "Retention settings, per-form retention, WordPress personal data export and erasure, separate form and entry permissions, honeypot and rate limits.",
    pro: "Team roles, per-form entry access, Cloudflare Turnstile, blocklists and per-form spam limits.",
  },
] as const

export const commonQuestions = [
  {
    question: "Can I start with Free and add Pro later?",
    answer:
      "Yes. Free is a standalone form builder. Pro works alongside it and adds settings to the same forms, entries and editor. Install and activate Free before adding Pro.",
  },
  {
    question: "Does Pro include every advanced feature?",
    answer:
      "Every Pro plan includes the same advanced features. Choose a plan by its site allowance and billing period. The configured plans below show the available options.",
  },
  {
    question: "What happens when my license expires?",
    answer:
      "Installed Pro features keep working. An active license provides eligible private updates and support. Subscription plans renew automatically unless you cancel; lifetime plans, when offered, are a one-time purchase with updates for life.",
  },
  {
    question: "Can I move a license to another site?",
    answer:
      "Yes. Disconnect the old site on its Pro license screen or remove its activation in your account, then connect the new site. Each distinct site address uses a slot in your plan.",
  },
  {
    question: "Do form answers go to the licensing service?",
    answer:
      "No. Licensing sends your key and site address to activate the site and authorize updates. Form answers and entries are not sent through that connection. Notifications, SMTP and webhooks send only the information required by the features you configure.",
  },
  {
    question: "Does a form still work without JavaScript?",
    answer:
      "Public forms support server validation and a no-JavaScript submission flow. Pro steps appear on one page, and server rules still apply. Spam checks that require a browser token may hold a submission for review. Building and rearranging fields in the admin editor requires JavaScript.",
  },
  {
    question: "Does the plugin guarantee an accessible website?",
    answer:
      "It provides visible labels, linked validation errors, contrast checks, protected focus styles and other safeguards. Your wording, page theme, embedded content and configuration still matter. Preview the published form and test keyboard navigation, errors and assistive technology before sharing it.",
  },
  {
    question: "How do installation and updates work?",
    answer:
      "Download Free here and upload it in WordPress. After purchasing Pro, sign in to your account to download the Pro installer and copy your license key. Connect the key in Accessible Forms → Pro license. Compatible private updates then appear on WordPress’s normal Updates and Plugins screens.",
  },
] as const
