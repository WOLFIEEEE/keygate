package service

import (
	"archive/zip"
	"bytes"
	"testing"
)

func wordpressZIP(t *testing.T, main string, extra string) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	f, _ := writer.Create("accessible-forms-pro/accessible-forms-pro.php")
	f.Write([]byte(main))
	if extra != "" {
		f, _ = writer.Create(extra)
		f.Write([]byte("extra"))
	}
	writer.Close()
	return buf.Bytes()
}
func TestInspectWordPressZIP(t *testing.T) {
	php := "<?php\n/**\n * Plugin Name: Accessible Forms Pro\n * Version: 1.0.1\n * Requires at least: 6.5\n * Requires PHP: 8.1\n */\nconst MIN_API = 5;\n"
	metadata, err := inspectWordPressZIP(wordpressZIP(t, php, ""), "accessible-forms-pro", "1.0.1")
	if err != nil || metadata.Requires != "6.5" || metadata.RequiresPHP != "8.1" || metadata.MinimumAPI != 5 {
		t.Fatalf("metadata %+v: %v", metadata, err)
	}
	for _, test := range []struct{ name, main, extra, version string }{
		{"version mismatch", php, "", "1.0.2"},
		{"missing header", "<?php", "", "1.0.1"},
		{"traversal", php, "accessible-forms-pro/../escape.php", "1.0.1"},
		{"extra root", php, "other/plugin.php", "1.0.1"},
		{"duplicate", php, "accessible-forms-pro/accessible-forms-pro.php", "1.0.1"},
		{"backslash", php, "accessible-forms-pro/dir\\evil.php", "1.0.1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := inspectWordPressZIP(wordpressZIP(t, test.main, test.extra), "accessible-forms-pro", test.version); err == nil {
				t.Fatal("invalid package accepted")
			}
		})
	}
}
