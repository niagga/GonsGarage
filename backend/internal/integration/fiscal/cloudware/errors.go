package cloudware

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
)

var secretPattern = regexp.MustCompile(`(?i)(bearer\s+\S+|refresh_token[=:]\s*\S+|access_token[=:]\s*\S+|authorization[=:]\s*\S+|token[=:]\s*\S+|[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)`)

// RedactDiagnostics strips credential-bearing fragments from diagnostic text.
func RedactDiagnostics(s string) string {
	if s == "" {
		return s
	}
	return secretPattern.ReplaceAllString(s, "[REDACTED]")
}

// NormalizeHTTPError translates Cloudware HTTP outcomes into provider-neutral errors.
func NormalizeHTTPError(status int, body []byte) *ports.FiscalProviderError {
	bodyStr := strings.TrimSpace(string(body))
	lower := strings.ToLower(bodyStr)

	switch {
	case status == http.StatusUnauthorized || strings.Contains(lower, "invalid_grant") || strings.Contains(lower, "refresh"):
		return ports.NewFiscalProviderError(ports.FiscalErrorClassUnauthorized, "refresh_failed", "credential refresh required", nil)
	case status == http.StatusForbidden && (strings.Contains(lower, "license") || strings.Contains(lower, "active_license")):
		return ports.NewFiscalProviderError(ports.FiscalErrorClassAuthorization, "active_license_required", "active GC license required for non-GET operations", nil)
	case status == http.StatusTooManyRequests:
		return ports.NewFiscalProviderError(ports.FiscalErrorClassRateLimit, "rate_limited", "provider rate limited the request", nil)
	case status >= 500:
		return ports.NewFiscalProviderError(ports.FiscalErrorClassTransient, "provider_unavailable", "provider temporarily unavailable", nil)
	case status >= 400:
		code := "provider_client_error"
		msg := "provider rejected the request"
		var parsed map[string]any
		if json.Unmarshal(body, &parsed) == nil {
			if c, ok := parsed["code"].(string); ok && strings.TrimSpace(c) != "" {
				code = strings.TrimSpace(c)
			}
			if m, ok := parsed["message"].(string); ok && strings.TrimSpace(m) != "" {
				msg = RedactDiagnostics(strings.TrimSpace(m))
			}
		}
		if strings.Contains(lower, "license") {
			return ports.NewFiscalProviderError(ports.FiscalErrorClassAuthorization, "active_license_required", "active GC license required for non-GET operations", nil)
		}
		return ports.NewFiscalProviderError(ports.FiscalErrorClassValidation, code, msg, nil)
	}

	// Successful status with unrecognized mutation payload defaults to ambiguous (no blind retry).
	if looksLikeUnknownMutation(body) {
		err := ports.NewFiscalProviderError(ports.FiscalErrorClassAmbiguous, "unknown_mutation_response", "provider response was not recognized", nil)
		err.Retryable = false
		err.DefinitiveNonAcceptance = false
		return err
	}
	err := ports.NewFiscalProviderError(ports.FiscalErrorClassAmbiguous, "unknown_mutation_response", "provider response was not recognized", nil)
	err.Retryable = false
	err.DefinitiveNonAcceptance = false
	return err
}

func looksLikeUnknownMutation(body []byte) bool {
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return true
	}
	if _, ok := parsed["id"]; ok {
		return false
	}
	if _, ok := parsed["document_id"]; ok {
		return false
	}
	return true
}
