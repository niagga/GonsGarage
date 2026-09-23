package cloudware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClientConfig bounds Cloudware transport behavior.
type HTTPClientConfig struct {
	BaseURL          string
	AllowedHosts     []string
	Timeout          time.Duration
	MaxResponseBytes int64
}

// HTTPClient is a bounded TLS-capable client with host allowlisting and no mutation retries.
type HTTPClient struct {
	base             *url.URL
	allowed          map[string]struct{}
	client           *http.Client
	maxResponseBytes int64
}

// NewHTTPClient constructs a fail-closed HTTP client for Cloudware.
func NewHTTPClient(cfg HTTPClientConfig) (*HTTPClient, error) {
	baseRaw := strings.TrimSpace(cfg.BaseURL)
	if baseRaw == "" {
		return nil, fmt.Errorf("%w: base URL required", ErrBaseURLNotAllowed)
	}
	base, err := url.Parse(baseRaw)
	if err != nil || base.Host == "" {
		return nil, fmt.Errorf("%w: invalid base URL", ErrBaseURLNotAllowed)
	}
	if base.Scheme != "https" && base.Scheme != "http" {
		return nil, fmt.Errorf("%w: unsupported scheme", ErrBaseURLNotAllowed)
	}
	allowed := make(map[string]struct{}, len(cfg.AllowedHosts)+1)
	allowed[strings.ToLower(base.Host)] = struct{}{}
	for _, host := range cfg.AllowedHosts {
		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" {
			allowed[host] = struct{}{}
		}
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	maxBytes := cfg.MaxResponseBytes
	if maxBytes <= 0 {
		maxBytes = 2 << 20
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          4,
		IdleConnTimeout:       30 * time.Second,
		DisableCompression:    false,
	}
	return &HTTPClient{
		base:    base,
		allowed: allowed,
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		maxResponseBytes: maxBytes,
	}, nil
}

// DoJSON performs a single request without transport-level retries.
func (c *HTTPClient) DoJSON(ctx context.Context, method, pathOrURL string, headers map[string]string, body any) ([]byte, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("cloudware http client is nil")
	}
	target, err := c.resolveURL(pathOrURL)
	if err != nil {
		return nil, err
	}
	var reader io.Reader
	if body != nil {
		raw, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return nil, marshalErr
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, c.maxResponseBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > c.maxResponseBytes {
		return nil, ErrResponseTooLarge
	}
	if resp.StatusCode >= 300 {
		return raw, NormalizeHTTPError(resp.StatusCode, raw)
	}
	return raw, nil
}

func (c *HTTPClient) resolveURL(pathOrURL string) (*url.URL, error) {
	raw := strings.TrimSpace(pathOrURL)
	if raw == "" {
		return c.base, nil
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return nil, err
		}
		if _, ok := c.allowed[strings.ToLower(parsed.Host)]; !ok {
			return nil, ErrBaseURLNotAllowed
		}
		return parsed, nil
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	resolved := c.base.ResolveReference(ref)
	if _, ok := c.allowed[strings.ToLower(resolved.Host)]; !ok {
		return nil, ErrBaseURLNotAllowed
	}
	return resolved, nil
}
