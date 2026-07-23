package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/zatrano/gocore/internal/infrastructure/config"
)

const netgsmSendURL = "https://api.netgsm.com.tr/sms/rest/v2/send"

// NetgsmProvider, Netgsm SMS REST v2 API adaptörüdür.
type NetgsmProvider struct {
	user      string
	password  string
	from      string
	encoding  string
	iysFilter string
	appName   string
	client    *http.Client
	log       *slog.Logger
}

// NewNetgsm, Netgsm sağlayıcısını yapılandırmayla kurar.
func NewNetgsm(cfg config.Notify, log *slog.Logger) *NetgsmProvider {
	encoding := cfg.NetgsmEncoding
	if encoding == "" {
		encoding = "TR"
	}
	iys := cfg.NetgsmIYSFilter
	if iys == "" {
		iys = "0"
	}
	return &NetgsmProvider{
		user:      cfg.NetgsmUser,
		password:  cfg.NetgsmPassword.Value(),
		from:      cfg.SMSFrom,
		encoding:  encoding,
		iysFilter: iys,
		appName:   cfg.NetgsmAppName,
		client:    &http.Client{Timeout: 30 * time.Second},
		log:       log,
	}
}

func (p *NetgsmProvider) Name() string { return "netgsm" }

type netgsmMessage struct {
	Msg string `json:"msg"`
	No  string `json:"no"`
}

type netgsmSendRequest struct {
	MsgHeader string          `json:"msgheader"`
	Messages  []netgsmMessage `json:"messages"`
	Encoding  string          `json:"encoding"`
	IYSFilter string          `json:"iysfilter"`
	AppName   string          `json:"appname,omitempty"`
}

type netgsmSendResponse struct {
	Code        string  `json:"code"`
	JobID       *string `json:"jobid"`
	Description string  `json:"description"`
}

// Send, tek alıcıya SMS gönderir.
func (p *NetgsmProvider) Send(ctx context.Context, to, body string) error {
	if p.user == "" || p.password == "" || p.from == "" {
		return &ProviderError{
			Provider: "netgsm",
			Code:     "config",
			Message:  "Netgsm kimlik bilgileri veya mesaj başlığı yapılandırılmamış",
		}
	}

	no := normalizeTRMobile(to)
	if no == "" {
		return &ProviderError{Provider: "netgsm", Code: "70", Message: "geçersiz telefon numarası"}
	}

	payload := netgsmSendRequest{
		MsgHeader: p.from,
		Messages:  []netgsmMessage{{Msg: body, No: no}},
		Encoding:  p.encoding,
		IYSFilter: p.iysFilter,
		AppName:   p.appName,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, netgsmSendURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.SetBasicAuth(p.user, p.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("netgsm: istek: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return &ProviderError{
			Provider: "netgsm",
			Code:     fmt.Sprintf("http_%d", resp.StatusCode),
			Message:  strings.TrimSpace(string(respBody)),
		}
	}

	var out netgsmSendResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("netgsm: yanıt çözümleme: %w", err)
	}
	if out.Code == "00" || out.Code == "01" || out.Code == "02" {
		jobID := ""
		if out.JobID != nil {
			jobID = *out.JobID
		}
		p.log.InfoContext(ctx, "netgsm sms kuyruğa alındı",
			slog.String("code", out.Code),
			slog.String("jobid", jobID),
			slog.String("to", no),
		)
		return nil
	}
	return &ProviderError{
		Provider: "netgsm",
		Code:     out.Code,
		Message:  out.Description,
	}
}
