// Package turnstile, Cloudflare Turnstile siteverify entegrasyonunu sağlar.
package turnstile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zatrano/gocore/internal/domain/shared"
)

const defaultVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// Verifier, Turnstile token doğrulaması yapar.
type Verifier interface {
	Enabled() bool
	Verify(ctx context.Context, token, remoteIP string) error
}

// Client, Cloudflare Turnstile siteverify API istemcisidir.
type Client struct {
	secret    string
	verifyURL string
	http      *http.Client
}

// NewClient, yapılandırmadan Turnstile doğrulayıcısı kurar. Secret boşsa
// devre dışı (Enabled=false) bir istemci döner.
func NewClient(siteKey, secret string) *Client {
	_ = siteKey // site key yalnızca şablon/widget tarafında kullanılır
	return &Client{
		secret:    secret,
		verifyURL: defaultVerifyURL,
		http:      &http.Client{Timeout: 10 * time.Second},
	}
}

// Enabled, secret anahtarı yapılandırıldıysa true döner.
func (c *Client) Enabled() bool {
	return c != nil && c.secret != ""
}

// Verify, Turnstile token'ını Cloudflare siteverify ile doğrular.
func (c *Client) Verify(ctx context.Context, token, remoteIP string) error {
	if !c.Enabled() {
		return nil
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrRequired
	}

	form := url.Values{}
	form.Set("secret", c.secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("turnstile: istek oluşturulamadı: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return ErrFailed.WithCause(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ErrFailed.WithCause(err)
	}

	var out verifyResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return ErrFailed.WithCause(err)
	}
	if !out.Success {
		return ErrFailed
	}
	return nil
}

type verifyResponse struct {
	Success bool `json:"success"`
}

// ErrRequired, Turnstile token'ının eksik olduğunu belirtir.
var ErrRequired = shared.NewDomainError(shared.KindValidation, "turnstile.required", "güvenlik doğrulamasını tamamlayın")

// ErrFailed, Turnstile doğrulamasının başarısız olduğunu belirtir.
var ErrFailed = shared.NewDomainError(shared.KindValidation, "turnstile.failed", "güvenlik doğrulaması başarısız")
