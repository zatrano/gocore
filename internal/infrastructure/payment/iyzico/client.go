package iyzico

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zatrano/gocore/internal/infrastructure/config"
	"github.com/zatrano/gocore/pkg/retry"
)

const (
	pathBinCheck      = "/payment/bin/check"
	pathInit3DS       = "/payment/3dsecure/initialize"
	pathAuth3DS       = "/payment/3dsecure/auth"
	pathPaymentDetail = "/payment/detail"
	defaultSandboxURL = "https://sandbox-api.iyzipay.com"
)

// Client, Iyzico REST API istemcisidir.
type Client struct {
	baseURL    string
	apiKey     string
	secretKey  string
	httpClient *http.Client
}

// NewClient, yapılandırmadan istemci oluşturur.
func NewClient(cfg config.Payment) *Client {
	base := cfg.IyzicoBaseURL
	if base == "" {
		base = defaultSandboxURL
	}
	return &Client{
		baseURL:   base,
		apiKey:    cfg.IyzicoAPIKey,
		secretKey: cfg.IyzicoSecretKey.Value(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIResponse, Iyzico yanıtlarındaki ortak alanları taşır.
type APIResponse struct {
	Status       string `json:"status"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	Locale       string `json:"locale"`
	SystemTime   int64  `json:"systemTime"`
}

func (r APIResponse) apiError() error {
	if r.Status == "success" {
		return nil
	}
	msg := r.ErrorMessage
	if msg == "" {
		msg = r.Status
	}
	return fmt.Errorf("iyzico: %s (%s)", msg, r.ErrorCode)
}

func (c *Client) post(ctx context.Context, uriPath string, reqBody, respBody any) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("iyzico: istek kodlama: %w", err)
	}
	return retry.Do(ctx, retry.DefaultOptions(), func() error {
		return c.doPost(ctx, uriPath, body, respBody)
	})
}

func (c *Client) doPost(ctx context.Context, uriPath string, body []byte, respBody any) error {
	auth, rnd := authorizationHeader(c.apiKey, c.secretKey, uriPath, body)
	url := c.baseURL + uriPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("iyzico: istek oluşturma: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", auth)
	req.Header.Set("x-iyzi-rnd", rnd)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return retry.MarkRetryable(fmt.Errorf("iyzico: http: %w", err))
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return retry.MarkRetryable(fmt.Errorf("iyzico: yanıt okuma: %w", err))
	}
	if res.StatusCode >= 400 {
		err := fmt.Errorf("iyzico: http %d: %s", res.StatusCode, string(raw))
		if retry.HTTPStatusRetryable(res.StatusCode) {
			return retry.MarkRetryable(err)
		}
		return err
	}
	if err := json.Unmarshal(raw, respBody); err != nil {
		return fmt.Errorf("iyzico: yanıt çözme: %w", err)
	}
	return nil
}
