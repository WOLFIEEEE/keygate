package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/tabloy/keygate/internal/model"
)

const maxWordPressZIP = 50 * 1024 * 1024

func (s *ReleaseService) inspectWordPressArtifact(ctx context.Context, key string, rel *model.Release, digest string) (*model.WordPressMetadata, error) {
	body, err := s.storage.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, maxWordPressZIP+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxWordPressZIP {
		return nil, fmt.Errorf("WordPress ZIP must be at most 50 MiB")
	}
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != digest {
		return nil, fmt.Errorf("uploaded package changed during finalization; upload again")
	}
	prod, err := s.store.FindProductByID(ctx, rel.ProductID)
	if err != nil {
		return nil, err
	}
	return inspectWordPressZIP(data, prod.Slug, rel.Version)
}

// Bound the compressed ZIP, expanded bytes, entry count and main PHP file.
// Reject traversal, symlinks, duplicates and extra roots before signing it.
func inspectWordPressZIP(data []byte, slug, version string) (*model.WordPressMetadata, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("package must be a valid ZIP")
	}
	if len(zr.File) > 10000 {
		return nil, fmt.Errorf("ZIP has too many entries")
	}
	main := slug + "/" + slug + ".php"
	seen := map[string]bool{}
	var expanded uint64
	var php, readme string
	for _, f := range zr.File {
		name := strings.TrimSuffix(f.Name, "/")
		if seen[name] || name == "" || strings.ContainsAny(name, "\\\x00") || path.Clean(name) != name ||
			(name != slug && !strings.HasPrefix(name, slug+"/")) || (!f.Mode().IsRegular() && !f.FileInfo().IsDir()) {
			return nil, fmt.Errorf("ZIP contains an unsafe, duplicate or unexpected path")
		}
		seen[name] = true
		if f.UncompressedSize64 > 100*1024*1024-expanded {
			return nil, fmt.Errorf("expanded ZIP must be at most 100 MiB")
		}
		expanded += f.UncompressedSize64
		if name != main && name != slug+"/readme.txt" {
			continue
		}
		if f.UncompressedSize64 > 128*1024 {
			return nil, fmt.Errorf("plugin header or readme is too large")
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(r, 128*1024+1))
		r.Close()
		if err != nil || len(b) > 128*1024 {
			return nil, fmt.Errorf("cannot read plugin headers")
		}
		if name == main {
			php = string(b)
		} else {
			readme = string(b)
		}
	}
	header := func(label string) string {
		m := regexp.MustCompile(`(?mi)^\s*\*?\s*` + regexp.QuoteMeta(label) + `:\s*([^\r\n]+)`).FindStringSubmatch(php)
		if len(m) != 2 {
			return ""
		}
		return strings.TrimSpace(m[1])
	}
	if header("Plugin Name") == "" || header("Version") != version {
		return nil, fmt.Errorf("plugin name/version must match this release; expected %s in %s", version, main)
	}
	wp, phpVersion := header("Requires at least"), header("Requires PHP")
	versionPattern := regexp.MustCompile(`^\d+\.\d+(?:\.\d+)?$`)
	if !versionPattern.MatchString(wp) || !versionPattern.MatchString(phpVersion) {
		return nil, fmt.Errorf("Requires at least and Requires PHP headers are required")
	}
	tested := ""
	if m := regexp.MustCompile(`(?mi)^Tested up to:\s*([\d.]+)\s*$`).FindStringSubmatch(readme); len(m) == 2 {
		tested = m[1]
	}
	api := 0
	if m := regexp.MustCompile(`(?m)^const MIN_API = (\d+);`).FindStringSubmatch(php); len(m) == 2 {
		api, _ = strconv.Atoi(m[1])
	}
	return &model.WordPressMetadata{PluginFile: main, Requires: wp, RequiresPHP: phpVersion, Tested: tested, MinimumAPI: api}, nil
}
