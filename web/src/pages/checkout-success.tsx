import { CheckCircle, Loader2 } from "lucide-react"
import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useSiteConfig } from "@/hooks/use-site-config"
import { useI18n } from "@/i18n"
import { freeDownload } from "@/lib/accessible-forms"
import { checkout } from "@/lib/api"

export default function CheckoutSuccessPage() {
  const { t } = useI18n()
  const { site_name, logo_url, attribution_text, attribution_url } = useSiteConfig()
  const [status, setStatus] = useState<"loading" | "ok" | "error">("loading")
  const [email, setEmail] = useState("")
  const [renewal, setRenewal] = useState(false)

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const sessionId = params.get("session_id")
    if (!sessionId) {
      setStatus("error")
      return
    }

    checkout
      .verify(sessionId)
      .then((r) => {
        setStatus(r.status === "ok" ? "ok" : "loading")
        if (r.email) setEmail(r.email)
        setRenewal(r.kind === "renewal")
      })
      .catch(() => setStatus("error"))
  }, [])

  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-muted/30 px-4 py-8">
      <main className="w-full max-w-md">
        <h1 className="sr-only">Purchase confirmation</h1>
        <Card className="w-full max-w-md text-center">
          <CardHeader>
            <div className="flex justify-center mb-2">
              <img src={logo_url || "/logo.svg"} alt={site_name} className="h-12 w-12" />
            </div>
            <CardTitle className="text-2xl">{site_name}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {status === "loading" && (
              <div className="flex flex-col items-center gap-3 py-4">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                <p className="text-muted-foreground">{t("checkout.verifying")}</p>
                <a href="/portal" className="text-primary underline min-h-11 inline-flex items-center">
                  Check your account for licenses
                </a>
              </div>
            )}
            {status === "ok" && (
              <div className="flex flex-col items-center gap-3 py-4">
                <CheckCircle className="h-12 w-12 text-green-500" />
                <h2 className="text-xl font-semibold">{t("checkout.success")}</h2>
                {renewal ? (
                  <>
                    <p className="text-muted-foreground">{t("checkout.renewalSuccess")}</p>
                    <Button className="mt-4" asChild>
                      <a href="/portal">{t("checkout.backToPortal")}</a>
                    </Button>
                  </>
                ) : (
                  <>
                    <p className="text-muted-foreground">
                      {email ? t("checkout.licenseSentTo", { email }) : t("checkout.licenseCreated")}
                    </p>
                    <Button className="mt-4" asChild>
                      <a href="/login">{t("checkout.viewLicense")}</a>
                    </Button>
                    <div className="text-sm text-left space-y-3 border-t pt-4 mt-2">
                      <p className="font-medium">Your next steps</p>
                      <ol className="list-decimal pl-5 space-y-2">
                        <li>Download your Pro installer and license key from your account.</li>
                        <li>
                          Install and activate{" "}
                          <a href={freeDownload.url} download className="text-primary underline">
                            Accessible Forms Free
                          </a>
                          , then install Pro.
                        </li>
                        <li>Connect the key in Accessible Forms → Pro license.</li>
                      </ol>
                      <a href="/#installation" className="text-primary underline inline-flex min-h-11 items-center">
                        Installation instructions
                      </a>
                    </div>
                  </>
                )}
              </div>
            )}
            {status === "error" && (
              <div className="flex flex-col items-center gap-3 py-4">
                <p className="text-muted-foreground">{t("checkout.error")}</p>
                <Button asChild variant="outline">
                  <a href="/portal">Open your account</a>
                </Button>
                <a href="/" className="text-primary underline min-h-11 inline-flex items-center">
                  Back to product and plans
                </a>
              </div>
            )}
          </CardContent>
        </Card>
      </main>
      <footer>
        <a
          href={attribution_url}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-4 text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          {attribution_text}
        </a>
      </footer>
    </div>
  )
}
