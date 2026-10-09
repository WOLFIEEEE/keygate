import { useQuery } from "@tanstack/react-query"

export interface PublicPlan {
  id: string
  name: string
  license_type: string
  billing_interval: string
  max_sites: number
  checkout_id: string
  price: number | null
  currency: string | null
}

export function usePublicPlans() {
  return useQuery({
    queryKey: ["public-wordpress-plans"],
    queryFn: async ({ signal }) => {
      const response = await fetch("/api/v1/products/accessible-forms-pro/plans", {
        signal: AbortSignal.any([signal, AbortSignal.timeout(15000)]),
      })
      const body = await response.json()
      if (!response.ok || body.success !== true || !Array.isArray(body.data?.plans))
        throw new Error("Plans unavailable")
      return body.data.plans as PublicPlan[]
    },
  })
}

export function useStoreReady() {
  return useQuery({
    queryKey: ["store-ready"],
    queryFn: async ({ signal }) => {
      const response = await fetch("/ready", {
        signal: AbortSignal.any([signal, AbortSignal.timeout(10000)]),
      })
      const body = await response.json()
      return response.ok && body.ready === true
    },
  })
}
