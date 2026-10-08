import { ArrowRight, ArrowUpRight } from "lucide-react"
import { type ReactNode, useEffect, useRef } from "react"
import { Link, Outlet, useLocation } from "react-router-dom"
import { useSiteConfig } from "@/hooks/use-site-config"
import { commonQuestions, featureGroups, freeDownload } from "@/lib/accessible-forms"
import "@/store.css"

export function StoreLayout() {
  const { site_name, loading, attribution_text, attribution_url } = useSiteConfig()
  const { pathname, hash } = useLocation()
  const previousPath = useRef(pathname)
  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      if (hash) {
        let targetID = hash.slice(1)
        try {
          targetID = decodeURIComponent(targetID)
        } catch {
          /* A malformed fragment must not break the store. */
        }
        const target = document.getElementById(targetID)
        target?.scrollIntoView()
        target?.focus({ preventScroll: true })
      } else if (previousPath.current !== pathname) {
        window.scrollTo(0, 0)
        document.getElementById("store-main")?.focus({ preventScroll: true })
      }
      previousPath.current = pathname
    })
    return () => cancelAnimationFrame(frame)
  }, [pathname, hash])
  const storeName = loading || site_name === "Keygate" ? "Accessible.org Store" : site_name
  return (
    <div className="af-store">
      <a href="#store-main" className="af-skip">
        Skip to content
      </a>
      <header className="af-header">
        <div className="af-shell af-header-inner">
          <Link to="/" className="af-brand">
            <img src="/accessible-forms-icon.png" alt="" width="40" height="40" />
            <span>{storeName}</span>
          </Link>
          <nav aria-label="Store navigation" className="af-nav">
            <Link to="/#products" aria-current={pathname === "/" ? "page" : undefined}>
              Products
            </Link>
            <Link to="/products/accessible-forms#features">Features</Link>
            <Link to="/pricing" aria-current={pathname === "/pricing" ? "page" : undefined}>
              Pricing
            </Link>
            <Link to="/guide" aria-current={pathname === "/guide" ? "page" : undefined}>
              Guide
            </Link>
            <Link to="/portal" className="af-account">
              My account <ArrowUpRight aria-hidden="true" size={15} />
            </Link>
          </nav>
        </div>
      </header>
      <main id="store-main" tabIndex={-1}>
        <Outlet />
      </main>
      <footer className="af-footer">
        <div className="af-shell af-footer-top">
          <div>
            <Link to="/" className="af-brand-text">
              Accessible.org
            </Link>
            <p>Thoughtful tools for the forms people use.</p>
          </div>
          <nav aria-label="Footer navigation">
            <Link to="/products/accessible-forms">Accessible Forms</Link>
            <Link to="/products/accessible-forms-pro">Accessible Forms Pro</Link>
            <Link to="/pricing">Plans and pricing</Link>
            <Link to="/guide">Installation and help</Link>
            <Link to="/portal">Licenses and billing</Link>
            <a href="https://accessible.org/">
              Visit Accessible.org <ArrowUpRight aria-hidden="true" size={14} />
            </a>
          </nav>
        </div>
        <div className="af-shell af-footer-bottom">
          <p>Accessible Forms for WordPress · GPLv2 or later</p>
          <a href={attribution_url} target="_blank" rel="noreferrer">
            {attribution_text}
          </a>
        </div>
      </footer>
    </div>
  )
}

export function StoreMeta({ title, description }: { title: string; description: string }) {
  const { loading } = useSiteConfig()
  const { pathname } = useLocation()
  useEffect(() => {
    if (loading) return
    document.title = `${title} · Accessible.org`
    for (const [selector, value] of [
      ['meta[name="description"]', description],
      ['meta[property="og:title"]', `${title} · Accessible.org`],
      ['meta[property="og:description"]', description],
      ['meta[name="twitter:title"]', `${title} · Accessible.org`],
      ['meta[name="twitter:description"]', description],
    ])
      document.querySelector(selector)?.setAttribute("content", value)
    let canonical = document.querySelector<HTMLLinkElement>('link[rel="canonical"]')
    if (!canonical) {
      canonical = document.createElement("link")
      canonical.rel = "canonical"
      document.head.append(canonical)
    }
    canonical.href = new URL(pathname, window.location.origin).href
  }, [title, description, pathname, loading])
  return null
}

export function ProLink({ children = "Explore Pro plans" }: { children?: ReactNode }) {
  return (
    <Link className="af-button" to="/pricing">
      {children}
      <ArrowRight aria-hidden="true" size={18} />
    </Link>
  )
}

export function FreeLink({ children = "Download Free" }: { children?: ReactNode }) {
  return (
    <a className="af-button af-button-secondary" href={freeDownload.url} download>
      {children}
      <ArrowRight aria-hidden="true" size={18} />
    </a>
  )
}

export function FeatureComparison() {
  return (
    <div className="af-comparison">
      <div className="af-comparison-head">
        <span>What you can do</span>
        <span>Accessible Forms · Free</span>
        <span>Accessible Forms Pro adds</span>
      </div>
      {featureGroups.map((group) => (
        <section key={group.id} className="af-comparison-row" aria-labelledby={`compare-${group.id}`}>
          <h3 id={`compare-${group.id}`}>{group.title}</h3>
          <dl>
            <div>
              <dt>Free</dt>
              <dd>{group.free}</dd>
            </div>
            <div>
              <dt>Pro</dt>
              <dd>{group.pro}</dd>
            </div>
          </dl>
        </section>
      ))}
    </div>
  )
}

export function StoreFAQ({ compact = false }: { compact?: boolean }) {
  return (
    <div className="af-faq">
      {(compact ? commonQuestions.slice(0, 4) : commonQuestions).map(({ question, answer }) => (
        <details key={question}>
          <summary>{question}</summary>
          <p>{answer}</p>
        </details>
      ))}
    </div>
  )
}

export function StoreCallout() {
  return (
    <section className="af-callout af-shell" aria-labelledby="callout-heading">
      <div>
        <p className="af-eyebrow">Start with one good form</p>
        <h2 id="callout-heading">Build in Free. Grow with Pro.</h2>
        <p>Keep your forms, entries and workflow in WordPress.</p>
      </div>
      <div className="af-actions">
        <FreeLink />
        <ProLink />
      </div>
    </section>
  )
}
