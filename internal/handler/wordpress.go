package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tabloy/keygate/internal/service"
	"github.com/tabloy/keygate/pkg/response"
)

type WordPressHandler struct {
	svc *service.WordPressService
}

func NewWordPressHandler(svc *service.WordPressService) *WordPressHandler {
	return &WordPressHandler{svc: svc}
}

type wordpressRequest struct {
	LicenseKey string `json:"license_key" binding:"required"`
	SiteURL    string `json:"site_url" binding:"required"`
	Version    string `json:"version"`
}

func bindWordPressRequest(c *gin.Context) (service.WordPressInput, bool) {
	// License-bearing responses and failures must never be shared by a cache.
	c.Header("Cache-Control", "private, no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	var req wordpressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "a JSON body with license_key and site_url is required (maximum 16 KiB)")
		return service.WordPressInput{}, false
	}
	return service.WordPressInput{
		ProductSlug: c.Param("product_slug"), LicenseKey: req.LicenseKey,
		SiteURL: req.SiteURL, Version: req.Version, IPAddress: c.ClientIP(),
	}, true
}

func (h *WordPressHandler) Activate(c *gin.Context) {
	in, ok := bindWordPressRequest(c)
	if !ok {
		return
	}
	out, err := h.svc.Activate(c.Request.Context(), in)
	if err != nil {
		writeAppErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *WordPressHandler) Verify(c *gin.Context) {
	in, ok := bindWordPressRequest(c)
	if !ok {
		return
	}
	out, err := h.svc.Verify(c.Request.Context(), in)
	if err != nil {
		writeAppErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *WordPressHandler) Deactivate(c *gin.Context) {
	in, ok := bindWordPressRequest(c)
	if !ok {
		return
	}
	if err := h.svc.Deactivate(c.Request.Context(), in); err != nil {
		writeAppErr(c, err)
		return
	}
	response.OK(c, gin.H{"status": "deactivated"})
}

func (h *WordPressHandler) Update(c *gin.Context) {
	in, ok := bindWordPressRequest(c)
	if !ok {
		return
	}
	out, err := h.svc.Update(c.Request.Context(), in)
	if err != nil {
		writeAppErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *WordPressHandler) Download(c *gin.Context) {
	in, ok := bindWordPressRequest(c)
	if !ok {
		return
	}
	out, err := h.svc.Download(c.Request.Context(), in)
	if err != nil {
		writeAppErr(c, err)
		return
	}
	response.OK(c, out)
}
