package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tabloy/keygate/internal/service"
	"github.com/tabloy/keygate/internal/store"
	"github.com/tabloy/keygate/pkg/response"
)

// A first installer has no WordPress site activation yet. Authenticate the
// owner through their portal session, then apply the normal release entitlement.
type PortalDownloadsHandler struct {
	Store    *store.Store
	Releases *service.ReleaseService
}

func (h *PortalDownloadsHandler) Download(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	email, _ := c.Get("email")
	owner, _ := email.(string)
	if owner == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var req struct {
		LicenseID string `json:"license_id" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		response.BadRequest(c, "license_id is required")
		return
	}
	lic, err := h.Store.FindLicenseByID(c.Request.Context(), req.LicenseID)
	if err != nil || !strings.EqualFold(owner, lic.Email) || lic.Product == nil || lic.Product.Slug != "accessible-forms-pro" {
		response.NotFound(c, "license not found")
		return
	}
	out, err := h.Releases.GenerateDownload(c.Request.Context(), service.DownloadInput{LicenseKey: h.Store.DecryptLicenseKey(lic), ProductID: lic.ProductID, Platform: service.WordPressPlatform})
	if err != nil {
		writeAppErr(c, err)
		return
	}
	response.OK(c, out)
}
