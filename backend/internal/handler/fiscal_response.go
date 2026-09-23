package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

// DecimalString rejects JSON numbers for fiscal decimal fields.
type DecimalString string

func (d *DecimalString) UnmarshalJSON(b []byte) error {
	if d == nil {
		return errors.New("decimal string is nil")
	}
	if len(b) == 0 || b[0] != '"' {
		return fmt.Errorf("decimal must be a JSON string")
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*d = DecimalString(s)
	return nil
}

func writeFiscalFieldError(c *gin.Context, field, message string) {
	c.JSON(http.StatusUnprocessableEntity, gin.H{
		"error":  "validation_failed",
		"fields": []gin.H{{"field": field, "code": "invalid", "message": message}},
	})
}

func writeFiscalizationError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, domain.ErrUnauthorizedAccess), errors.Is(err, domain.ErrPermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case errors.Is(err, ports.ErrFiscalArtifactDenied):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, domain.ErrInvoiceNotFound), errors.Is(err, ports.ErrFiscalDocumentNotFound),
		errors.Is(err, ports.ErrFiscalArtifactNotFound), errors.Is(err, domain.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, ports.ErrFiscalVersionConflict), errors.Is(err, ports.ErrFiscalIntentConflict),
		errors.Is(err, ports.ErrFiscalHistoryProtected), errors.Is(err, ports.ErrFiscalActionNotAllowed):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ports.ErrFiscalNotReady), errors.Is(err, ports.ErrFiscalInvoiceNotEligible):
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":  "not_ready",
			"fields": []gin.H{{"field": "readiness", "code": "not_ready", "message": err.Error()}},
		})
	case errors.Is(err, ports.ErrFiscalArtifactUnavailable), errors.Is(err, ports.ErrFiscalArtifactCompromised):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "artifact temporarily unavailable"})
	default:
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	}
}

func sanitizeFiscalizationProjection(p *ports.FiscalizationProjection) ports.FiscalizationProjection {
	if p == nil {
		return ports.FiscalizationProjection{Status: string(domain.FiscalPresentationStateUnavailable), AllowedActions: []string{}}
	}
	out := *p
	if out.AllowedActions == nil {
		out.AllowedActions = []string{}
	}
	// Defense in depth: never serialize known-sensitive keys even if a caller mutates the struct.
	out.LastErrorSafe = stripSensitiveFragments(out.LastErrorSafe)
	return out
}

func sanitizeFiscalizationSummary(s ports.FiscalizationSummary) ports.FiscalizationSummary {
	if s.AllowedActions == nil {
		s.AllowedActions = []string{}
	}
	return s
}

func stripSensitiveFragments(s string) string {
	lower := strings.ToLower(s)
	banned := []string{"bearer ", "token=", "credential", "storagekey", "storage_key", "https://", "http://", "s3://"}
	for _, b := range banned {
		if strings.Contains(lower, b) {
			return "[redacted]"
		}
	}
	return s
}

func projectionJSONOmitsSecrets(raw []byte) bool {
	lower := strings.ToLower(string(raw))
	for _, banned := range []string{
		"credentialciphertext", "credentialnonce", "storagekey", "rawprovider",
		"authorization:", "bearer ", "s3://", `"https://`,
	} {
		if strings.Contains(lower, banned) {
			return false
		}
	}
	return true
}
