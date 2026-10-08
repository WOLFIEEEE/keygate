import { ArrowDownToLine, ArrowUpRight } from "lucide-react"
import { useEffect } from "react"
import { Link, Outlet, useLocation } from "react-router-dom"
import { useSiteConfig } from "@/hooks/use-site-config"
import { featureGroups, freeDownload } from "@/lib/accessible-forms"
import "@/store.css"

export function StoreLayout() {
  const { attribution_text, attribution_url } = useSiteConfig()
  const { hash } = useLocation()
  useEffect(() => {
    if (!hash) return
    const frame = requestAnimationFrame(() => {
      let targetID = hash.slice(1)
      try {
        targetID = decodeURIComponent(targetID)
      } catch {
        /* A malformed fragment must not break the purchase page. */
      }
      const target = document.getElementById(targetID)
      if (target instanceof HTMLDetailsElement) target.open = true
      target?.scrollIntoView()
      target?.focus({ preventScroll: true })
    })
    return () => cancelAnimationFrame(frame)
  }, [hash])
  return (
    <div className="af-store">
      <a href="#store-main" className="af-skip">
        Skip to content
      </a>
      <header className="af-header">
        <div className="af-shell af-header-inner">
          <Link to="/" className="af-brand">
            <img src="/accessible-forms-icon.png" alt="" width="32" height="32" />
            <span>Accessible.org</span>
          </Link>
          <Link to="/portal" className="af-account">
            My account <ArrowUpRight aria-hidden="true" size={16} />
          </Link>
        </div>
      </header>
      <main id="store-main" tabIndex={-1}>
        <Outlet />
      </main>
      <footer className="af-footer af-shell">
        <p>Payments &amp; licenses</p>
        <a href={attribution_url} target="_blank" rel="noreferrer">
          {attribution_text}
        </a>
      </footer>
    </div>
  )
}

export function FreeLink() {
  return (
    <a className="af-button af-button-secondary" href={freeDownload.url} download>
      <ArrowDownToLine aria-hidden="true" size={17} /> Download Free
    </a>
  )
}

export function FeatureComparison() {
  return (
    <table className="af-comparison">
      <caption className="af-sr-only">Accessible Forms Free features and the features Pro adds</caption>
      <thead>
        <tr>
          <th scope="col">Feature</th>
          <th scope="col">Free</th>
          <th scope="col">Pro adds</th>
        </tr>
      </thead>
      <tbody>
        {featureGroups.map((group) => (
          <tr key={group.id}>
            <th scope="row">{group.title}</th>
            <td>{group.free}</td>
            <td>{group.pro}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
