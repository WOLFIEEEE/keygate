package main

import (
	"os"
	"strings"
	"testing"
)

func TestStorefrontMetadata(t *testing.T) {
	source, err := os.ReadFile("../../web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	for path, page := range storefrontPages {
		out := string(storefrontHTML(source, path, "https://license.example.test"))
		if !strings.Contains(out, "<title>"+page.title+" · Accessible.org</title>") {
			t.Fatalf("%s title missing", path)
		}
		if !strings.Contains(out, `rel="canonical" href="https://license.example.test`+path+`"`) {
			t.Fatalf("%s canonical missing", path)
		}
		if !strings.Contains(out, `name="generator" content="Keygate (https://keygate.app)"`) {
			t.Fatal("attribution changed")
		}
	}
	// A configuration string cannot break the attribute or become replacement
	// syntax, even when it contains quotes, ampersands or dollar signs.
	out := string(storefrontHTML(source, "/pricing", `https://example.test/"&$1`))
	if !strings.Contains(out, `https://example.test/&#34;&amp;$1/pricing`) {
		t.Fatal("base URL was not escaped literally")
	}
	if string(storefrontHTML(source, "/portal", "https://example.test")) != string(source) {
		t.Fatal("private shell changed")
	}
}
