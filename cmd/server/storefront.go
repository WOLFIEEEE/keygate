package main

import (
	"html"
	"regexp"
)

type storefrontPage struct{ title, description string }

var storefrontPages = map[string]storefrontPage{
	"/":                              {"Accessible Forms for WordPress", "Build WordPress forms with visible labels, helpful validation and a clear submission workflow. Start free or add advanced tools with Pro."},
	"/products/accessible-forms":     {"Accessible Forms", "A free WordPress form builder with unlimited forms, 13 field types, four themes, clear validation and entry management."},
	"/products/accessible-forms-pro": {"Accessible Forms Pro", "Add conditional questions, multi-step forms, private uploads, custom styling, delivery rules and team workflows to Accessible Forms."},
	"/pricing":                       {"Accessible Forms Pro plans and pricing", "Compare Free and Pro. Choose a Pro license by site allowance and billing period, with the same advanced features in every plan."},
	"/guide":                         {"Accessible Forms installation and help", "Install Free and Pro, connect your license, publish your first form and understand updates, delivery and data handling."},
}

var storefrontTags = map[string]*regexp.Regexp{
	"title":               regexp.MustCompile(`<title>[^<]*</title>`),
	"description":         regexp.MustCompile(`<meta name="description" content="[^"]*"\s*/?>`),
	"og:title":            regexp.MustCompile(`<meta property="og:title" content="[^"]*"\s*/?>`),
	"og:description":      regexp.MustCompile(`<meta property="og:description" content="[^"]*"\s*/?>`),
	"og:url":              regexp.MustCompile(`<meta property="og:url" content="[^"]*"\s*/?>`),
	"og:image":            regexp.MustCompile(`<meta property="og:image" content="[^"]*"\s*/?>`),
	"twitter:title":       regexp.MustCompile(`<meta name="twitter:title" content="[^"]*"\s*/?>`),
	"twitter:description": regexp.MustCompile(`<meta name="twitter:description" content="[^"]*"\s*/?>`),
	"twitter:image":       regexp.MustCompile(`<meta name="twitter:image" content="[^"]*"\s*/?>`),
	"canonical":           regexp.MustCompile(`<link rel="canonical" href="[^"]*"\s*/?>`),
}

// Link previews and non-JavaScript crawlers receive the product's metadata.
// Only known public store routes are changed; API and authenticated pages keep
// their existing shell and attribution. Content and URLs are escaped as HTML.
func storefrontHTML(source []byte, path, baseURL string) []byte {
	page, ok := storefrontPages[path]
	if !ok {
		return source
	}
	title := html.EscapeString(page.title + " · Accessible.org")
	description := html.EscapeString(page.description)
	pageURL := html.EscapeString(baseURL + path)
	imageURL := html.EscapeString(baseURL + "/accessible-forms-icon.png")
	values := map[string]string{
		"title":               "<title>" + title + "</title>",
		"description":         `<meta name="description" content="` + description + `" />`,
		"og:title":            `<meta property="og:title" content="` + title + `" />`,
		"og:description":      `<meta property="og:description" content="` + description + `" />`,
		"og:url":              `<meta property="og:url" content="` + pageURL + `" />`,
		"og:image":            `<meta property="og:image" content="` + imageURL + `" />`,
		"twitter:title":       `<meta name="twitter:title" content="` + title + `" />`,
		"twitter:description": `<meta name="twitter:description" content="` + description + `" />`,
		"twitter:image":       `<meta name="twitter:image" content="` + imageURL + `" />`,
		"canonical":           `<link rel="canonical" href="` + pageURL + `" />`,
	}
	for key, value := range values {
		// ReplaceAllFunc treats '$' in a configured URL as literal content.
		source = storefrontTags[key].ReplaceAllFunc(source, func([]byte) []byte { return []byte(value) })
	}
	return source
}
