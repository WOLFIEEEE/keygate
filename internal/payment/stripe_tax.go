package payment

import "github.com/stripe/stripe-go/v82"

// Stripe Tax is optional and requires the merchant's registrations. Both the
// API checkout and the public /pay link use this same policy.
func (h *StripeHandler) applyAutomaticTax(params *stripe.CheckoutSessionParams) {
	if !h.AutomaticTax {
		return
	}
	params.AutomaticTax = &stripe.CheckoutSessionAutomaticTaxParams{Enabled: stripe.Bool(true)}
	params.TaxIDCollection = &stripe.CheckoutSessionTaxIDCollectionParams{Enabled: stripe.Bool(true)}
	params.BillingAddressCollection = stripe.String("required")
	if params.Customer != nil {
		params.CustomerUpdate = &stripe.CheckoutSessionCustomerUpdateParams{Address: stripe.String("auto")}
	}
}
