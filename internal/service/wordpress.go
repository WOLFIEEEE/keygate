package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/mod/semver"
	"golang.org/x/net/idna"

	"github.com/tabloy/keygate/internal/model"
	"github.com/tabloy/keygate/internal/store"
	"github.com/tabloy/keygate/pkg/apperr"
)

// WordPressPlatform stores the plugin ZIP alongside the other release artifacts.
const WordPressPlatform = "wordpress"

// WordPressService adapts the existing activation and release services to a
// WordPress site's identity. Licenses grant updates; installed plugin features
// are never switched off by this API.
type WordPressService struct {
	BaseURL         string
	DownloadOrigins []string
	store           *store.Store
	licenses        *LicenseService
	releases        *ReleaseService
}

func NewWordPressService(st *store.Store, licenses *LicenseService, releases *ReleaseService) *WordPressService {
	return &WordPressService{store: st, licenses: licenses, releases: releases}
}

type WordPressInput struct {
	ProductSlug string
	LicenseKey  string
	SiteURL     string
	IPAddress   string
	// Version is the installed version for Update, the requested version for Download.
	Version string
}

type WordPressUpdateResult struct {
	UpdateAvailable bool                   `json:"update_available"`
	Update          *WordPressPluginUpdate `json:"update"`
	Meta            map[string]any         `json:"meta"`
}

// WordPressPluginUpdate carries the fields used by update_plugins_{$hostname}.
// Package is deliberately short-lived: the client must request Download again
// immediately before an upgrade, rather than reuse WordPress's cached URL.
type WordPressPluginUpdate struct {
	Name             string                   `json:"name"`
	Slug             string                   `json:"slug"`
	Version          string                   `json:"version"`
	URL              string                   `json:"url"`
	Package          string                   `json:"package"`
	PackageExpiresAt time.Time                `json:"package_expires_at"`
	SHA256           string                   `json:"sha256"`
	ReleaseNotes     string                   `json:"release_notes"`
	WordPress        *model.WordPressMetadata `json:"wordpress_metadata,omitempty"`
}

func (s *WordPressService) Activate(ctx context.Context, in WordPressInput) (*ActivateResult, error) {
	prod, site, err := s.request(ctx, in)
	if err != nil {
		return nil, err
	}
	return s.licenses.Activate(ctx, ActivateInput{
		LicenseKey: strings.TrimSpace(in.LicenseKey), ProductID: prod.ID,
		Identifier: site.identifier, IdentifierType: "device", Label: site.label,
		IPAddress: in.IPAddress,
	})
}

func (s *WordPressService) Verify(ctx context.Context, in WordPressInput) (*VerifyResult, error) {
	prod, site, err := s.request(ctx, in)
	if err != nil {
		return nil, err
	}
	return s.verify(ctx, in, prod, site)
}

func (s *WordPressService) Deactivate(ctx context.Context, in WordPressInput) error {
	prod, site, err := s.request(ctx, in)
	if err != nil {
		return err
	}
	return s.licenses.Deactivate(ctx, DeactivateInput{
		LicenseKey: strings.TrimSpace(in.LicenseKey), ProductID: prod.ID,
		Identifier: site.identifier, IPAddress: in.IPAddress,
	})
}

func (s *WordPressService) Update(ctx context.Context, in WordPressInput) (*WordPressUpdateResult, error) {
	if err := validateVersion(in.Version); err != nil {
		return nil, apperr.BadRequest("version must be the installed plugin's semantic version (for example 1.0.0)")
	}
	prod, site, err := s.request(ctx, in)
	if err != nil {
		return nil, err
	}
	verified, err := s.verify(ctx, in, prod, site)
	if err != nil {
		return nil, err
	}
	out := &WordPressUpdateResult{Meta: responseMeta()}
	rel, err := s.releases.findLatestPublished(ctx, prod.ID, model.ReleaseChannelStable, WordPressPlatform, verified.UpdatesUntil)
	if errors.Is(err, ErrReleaseNoneAvailable) {
		return out, nil
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if semver.Compare(semverNormalize(rel.Version), semverNormalize(in.Version)) <= 0 {
		return out, nil
	}

	download, err := s.releases.GenerateDownload(ctx, DownloadInput{
		LicenseKey: strings.TrimSpace(in.LicenseKey), ProductID: prod.ID,
		Version: rel.Version, Platform: WordPressPlatform, Channel: model.ReleaseChannelStable,
	})
	if err != nil {
		return nil, err
	}
	out.UpdateAvailable = true
	out.Update = &WordPressPluginUpdate{
		Name: prod.Name, Slug: prod.Slug, Version: rel.Version, URL: prod.DownloadURL,
		Package: download.URL, PackageExpiresAt: download.ExpiresAt,
		SHA256: download.SHA256, ReleaseNotes: rel.ReleaseNotes,
		WordPress: download.WordPress,
	}
	return out, nil
}

// Download rechecks activation, license lifecycle, publication and maintenance
// entitlement before producing a fresh URL for the exact version being installed.
func (s *WordPressService) Download(ctx context.Context, in WordPressInput) (*DownloadResult, error) {
	if err := validateVersion(in.Version); err != nil {
		return nil, apperr.BadRequest("version must be the exact plugin version to download (for example 1.0.0)")
	}
	prod, site, err := s.request(ctx, in)
	if err != nil {
		return nil, err
	}
	if _, err := s.verify(ctx, in, prod, site); err != nil {
		return nil, err
	}
	return s.releases.GenerateDownload(ctx, DownloadInput{
		LicenseKey: strings.TrimSpace(in.LicenseKey), ProductID: prod.ID,
		Version: in.Version, Platform: WordPressPlatform, Channel: model.ReleaseChannelStable,
	})
}

func (s *WordPressService) verify(ctx context.Context, in WordPressInput, prod *model.Product, site wordpressSite) (*VerifyResult, error) {
	verified, err := s.licenses.Verify(ctx, VerifyInput{
		LicenseKey: strings.TrimSpace(in.LicenseKey), ProductID: prod.ID,
		Identifier: site.identifier, IPAddress: in.IPAddress,
	})
	if err != nil {
		return nil, err
	}
	lic, err := s.store.FindLicenseByKey(ctx, strings.TrimSpace(in.LicenseKey))
	if err != nil {
		return nil, apperr.Internal(err)
	}
	verified.ActiveSites, err = s.store.CountActivations(ctx, lic.ID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	verified.MaxSites = s.licenses.maxActivations(lic)
	if lic.Plan != nil {
		verified.LicenseType = lic.Plan.LicenseType
	}
	return verified, nil
}

// Info announces releases without revealing storage keys or download links.
func (s *WordPressService) Info(ctx context.Context, slug string) (*WordPressPluginUpdate, error) {
	prod, err := s.store.FindProductBySlug(ctx, slug)
	if err != nil || !model.ProductSupports(prod.Type, model.CapReleases) {
		return nil, licenseNotFound()
	}
	rel, err := s.releases.findLatestPublished(ctx, prod.ID, model.ReleaseChannelStable, WordPressPlatform, nil)
	if errors.Is(err, ErrReleaseNoneAvailable) {
		return nil, apperr.New(404, "NO_RELEASE", "no WordPress release is published")
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	out := &WordPressPluginUpdate{Name: prod.Name, Slug: prod.Slug, Version: rel.Version, URL: prod.DownloadURL, ReleaseNotes: rel.ReleaseNotes}
	for _, a := range rel.Artifacts {
		if a.Platform == WordPressPlatform {
			out.WordPress = a.WordPress
		}
	}
	return out, nil
}

func (s *WordPressService) PublicConfig(ctx context.Context, slug string) (map[string]any, error) {
	prod, err := s.store.FindProductBySlug(ctx, slug)
	if err != nil {
		return nil, licenseNotFound()
	}
	keys, err := s.store.ListSigningKeys(ctx, prod.ID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	public := map[string]string{}
	for _, k := range keys {
		public[k.ID] = k.PublicKey
	}
	return map[string]any{"base_url": s.BaseURL, "download_origins": s.DownloadOrigins, "product_id": prod.ID, "product_slug": prod.Slug, "public_keys": public, "meta": responseMeta()}, nil
}

func (s *WordPressService) request(ctx context.Context, in WordPressInput) (*model.Product, wordpressSite, error) {
	site, err := normalizeWordPressSite(in.SiteURL)
	if err != nil {
		return nil, wordpressSite{}, apperr.BadRequest(err.Error())
	}
	if strings.TrimSpace(in.LicenseKey) == "" || len(in.LicenseKey) > 512 {
		return nil, wordpressSite{}, apperr.BadRequest("license_key is required and must be at most 512 bytes")
	}
	prod, err := s.store.FindProductBySlug(ctx, strings.ToLower(strings.TrimSpace(in.ProductSlug)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, wordpressSite{}, licenseNotFound()
	}
	if err != nil {
		return nil, wordpressSite{}, apperr.Internal(err)
	}
	if !model.ProductSupports(prod.Type, model.CapActivations) || !model.ProductSupports(prod.Type, model.CapReleases) {
		return nil, wordpressSite{}, licenseNotFound()
	}
	return prod, site, nil
}

type wordpressSite struct {
	identifier string
	label      string
}

// normalizeWordPressSite never visits the URL. HTTP-to-HTTPS migrations, default
// ports, host casing and trailing slashes retain the same slot. Subdirectories,
// www, staging subdomains and nonstandard ports remain separate installations.
func normalizeWordPressSite(raw string) (wordpressSite, error) {
	raw = strings.TrimSpace(raw)
	invalid := errors.New("site_url must be an absolute HTTP(S) URL without credentials, query parameters or a fragment")
	if raw == "" || len(raw) > 2048 || strings.ContainsAny(raw, "\\#\x00\r\n\t") {
		return wordpressSite{}, invalid
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" ||
		u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return wordpressSite{}, invalid
	}
	if strings.ContainsAny(u.Path, "\\") || strings.ContainsFunc(u.Path, unicode.IsControl) ||
		strings.Contains(strings.ToLower(u.RawPath), "%2f") {
		return wordpressSite{}, invalid
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if ip := net.ParseIP(host); ip != nil {
		host = ip.String()
	} else {
		host, err = idna.Lookup.ToASCII(host)
		if err != nil || host == "" || strings.ContainsAny(host, "%:/ ") {
			return wordpressSite{}, invalid
		}
		for _, label := range strings.Split(host, ".") {
			if label == "" || len(label) > 63 {
				return wordpressSite{}, invalid
			}
		}
	}
	port := u.Port()
	if port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return wordpressSite{}, invalid
		}
		port = strconv.Itoa(n)
		if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
			port = ""
		}
	} else if strings.HasSuffix(u.Host, ":") {
		return wordpressSite{}, invalid
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	cleanPath := path.Clean("/" + u.Path)
	if cleanPath == "/" {
		cleanPath = ""
	}
	canonical := (&url.URL{Host: host, Path: cleanPath}).String()
	digest := sha256.Sum256([]byte(canonical))
	return wordpressSite{
		identifier: "wp:" + hex.EncodeToString(digest[:]),
		label:      u.Scheme + ":" + canonical,
	}, nil
}
