package moka

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zatrano/gocore/internal/infrastructure/config"
	"github.com/zatrano/gocore/pkg/retry"
)

const (
	pathDoDirectPaymentThreeD = "/PaymentDealer/DoDirectPaymentThreeD"
	defaultTestBaseURL        = "https://service.refmokaunited.com"
)

// Client, Moka United REST API istemcisidir.
type Client struct {
	baseURL    string
	dealerCode string
	username   string
	password   string
	software   string
	httpClient *http.Client
}

// NewClient, yapılandırmadan istemci oluşturur.
func NewClient(cfg config.Payment) *Client {
	base := cfg.MokaBaseURL
	if base == "" {
		base = defaultTestBaseURL
	}
	software := cfg.MokaSoftware
	if software == "" {
		software = "zatrano"
	}
	return &Client{
		baseURL:    base,
		dealerCode: cfg.MokaDealerCode,
		username:   cfg.MokaUsername,
		password:   cfg.MokaPassword.Value(),
		software:   software,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIResponse, Moka yanıt zarfıdır.
type APIResponse struct {
	Data          any    `json:"Data"`
	ResultCode    string `json:"ResultCode"`
	ResultMessage string `json:"ResultMessage"`
	Exception     any    `json:"Exception"`
}

func (r APIResponse) apiError() error {
	if r.ResultCode == "Success" {
		return nil
	}
	msg := strings.TrimSpace(r.ResultMessage)
	if msg == "" {
		msg = r.ResultCode
	}
	if msg == "" {
		msg = "unknown error"
	}
	return fmt.Errorf("moka: %s", msg)
}

func (c *Client) auth() map[string]string {
	return map[string]string{
		"DealerCode": c.dealerCode,
		"Username":   c.username,
		"Password":   c.password,
		"CheckKey":   CheckKey(c.dealerCode, c.username, c.password),
	}
}

func (c *Client) post(ctx context.Context, uriPath string, body any, dataTarget any) error {
	return c.postBody(ctx, uriPath, map[string]any{
		"PaymentDealerRequest": body,
	}, dataTarget)
}

func (c *Client) postBody(ctx context.Context, uriPath string, body map[string]any, dataTarget any) error {
	payload := map[string]any{"PaymentDealerAuthentication": c.auth()}
	for k, v := range body {
		payload[k] = v
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("moka: istek kodlama: %w", err)
	}
	return retry.Do(ctx, retry.DefaultOptions(), func() error {
		return c.doPostBody(ctx, uriPath, raw, dataTarget)
	})
}

func (c *Client) doPostBody(ctx context.Context, uriPath string, raw []byte, dataTarget any) error {
	url := c.baseURL + uriPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("moka: istek oluşturma: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return retry.MarkRetryable(fmt.Errorf("moka: http: %w", err))
	}
	defer res.Body.Close()

	respRaw, err := io.ReadAll(res.Body)
	if err != nil {
		return retry.MarkRetryable(fmt.Errorf("moka: yanıt okuma: %w", err))
	}
	if res.StatusCode >= 400 {
		err := fmt.Errorf("moka: http %d: %s", res.StatusCode, string(respRaw))
		if retry.HTTPStatusRetryable(res.StatusCode) {
			return retry.MarkRetryable(err)
		}
		return err
	}

	var envelope struct {
		APIResponse
		Data json.RawMessage `json:"Data"`
	}
	if err := json.Unmarshal(respRaw, &envelope); err != nil {
		return fmt.Errorf("moka: yanıt çözme: %w", err)
	}
	if err := envelope.apiError(); err != nil {
		return err
	}
	if dataTarget != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, dataTarget); err != nil {
			return fmt.Errorf("moka: data çözme: %w", err)
		}
	}
	return nil
}
