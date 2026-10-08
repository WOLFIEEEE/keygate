import { MutationCache, QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { type ComponentType, lazy, StrictMode, Suspense } from "react"
import { createRoot } from "react-dom/client"
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { ErrorBoundary } from "@/components/error-boundary"
import { AdminLayout, PortalLayout } from "@/components/layout"
import { StoreLayout } from "@/components/store-layout"
import { showToast, ToastBridge, ToastProvider } from "@/components/toast"
import { AuthProvider } from "@/hooks/use-auth"
import { SiteConfigProvider } from "@/hooks/use-site-config"
import { I18nProvider } from "@/i18n"
import AcceptInvitePage from "@/pages/accept-invite"
import CheckoutSuccessPage from "@/pages/checkout-success"
import LoginPage from "@/pages/login"
import SetupPage from "@/pages/setup"
import StorePage from "@/pages/store"
import "./index.css"

// Product visitors do not need the admin tables, billing dialogs or charts.
// Keep the existing layout and attribution while the requested page loads.
function deferredPage(load: () => Promise<{ default: ComponentType }>) {
  const Page = lazy(load)
  return function DeferredPage() {
    return (
      <Suspense
        fallback={
          <p role="status" className="p-6">
            Loading page…
          </p>
        }
      >
        <Page />
      </Suspense>
    )
  }
}
const AddonsPage = deferredPage(() => import("@/pages/admin/addons"))
const AnalyticsPage = deferredPage(() => import("@/pages/admin/analytics"))
const APIKeysPage = deferredPage(() => import("@/pages/admin/api-keys"))
const AuditPage = deferredPage(() => import("@/pages/admin/audit"))
const CustomersPage = deferredPage(() => import("@/pages/admin/customers"))
const DashboardPage = deferredPage(() => import("@/pages/admin/dashboard"))
const LicensesPage = deferredPage(() => import("@/pages/admin/licenses"))
const PlansPage = deferredPage(() => import("@/pages/admin/plans"))
const ProductsPage = deferredPage(() => import("@/pages/admin/products"))
const ReleasesPage = deferredPage(() => import("@/pages/admin/releases"))
const SettingsPage = deferredPage(() => import("@/pages/admin/settings"))
const WebhooksPage = deferredPage(() => import("@/pages/admin/webhooks"))
const PortalAccountPage = deferredPage(() => import("@/pages/portal/account"))
const PortalLicensesPage = deferredPage(() => import("@/pages/portal/licenses"))

const queryClient = new QueryClient({
  mutationCache: new MutationCache({
    onError: (error) => {
      showToast(error instanceof Error ? error.message : "An error occurred")
    },
  }),
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false, staleTime: 30_000 },
  },
})

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <I18nProvider>
          <ToastProvider>
            <ToastBridge />
            <SiteConfigProvider>
              <AuthProvider>
                <ErrorBoundary>
                  <Routes>
                    <Route path="/setup" element={<SetupPage />} />
                    <Route path="/login" element={<LoginPage />} />
                    <Route element={<StoreLayout />}>
                      <Route path="/" element={<StorePage />} />
                      {/* Keep old plugin links and bookmarks working without separate public pages. */}
                      <Route path="/products/accessible-forms" element={<Navigate to="/" replace />} />
                      <Route path="/products/accessible-forms-pro" element={<Navigate to="/#comparison" replace />} />
                      <Route path="/pricing" element={<Navigate to="/#store-plans" replace />} />
                      <Route path="/guide" element={<Navigate to="/#installation" replace />} />
                    </Route>
                    <Route path="/checkout/success" element={<CheckoutSuccessPage />} />
                    <Route path="/accept-invite" element={<AcceptInvitePage />} />

                    {/* Admin */}
                    <Route path="/admin" element={<AdminLayout />}>
                      <Route index element={<DashboardPage />} />
                      <Route path="products" element={<ProductsPage />} />
                      <Route path="plans" element={<PlansPage />} />
                      <Route path="releases" element={<ReleasesPage />} />
                      <Route path="licenses" element={<LicensesPage />} />
                      <Route path="api-keys" element={<APIKeysPage />} />
                      <Route path="webhooks" element={<WebhooksPage />} />
                      <Route path="addons" element={<AddonsPage />} />
                      <Route path="analytics" element={<AnalyticsPage />} />
                      <Route path="audit" element={<AuditPage />} />
                      <Route path="customers" element={<CustomersPage />} />
                      <Route path="settings" element={<SettingsPage />} />
                    </Route>

                    {/* Portal */}
                    <Route path="/portal" element={<PortalLayout />}>
                      <Route index element={<PortalLicensesPage />} />
                      <Route path="account" element={<PortalAccountPage />} />
                    </Route>

                    <Route path="*" element={<Navigate to="/login" replace />} />
                  </Routes>
                </ErrorBoundary>
              </AuthProvider>
            </SiteConfigProvider>
          </ToastProvider>
        </I18nProvider>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
)
