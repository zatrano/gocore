package sms

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zatrano/gocore/internal/infrastructure/config"
)

const iletimerkeziSendURL = "https://api.iletimerkezi.com/v1/send-sms/get/"

// IletimerkeziProvider, İleti Merkezi SMS GET API adaptörüdür.
// Dokümantasyon: https://www.toplusmsapi.com/sms/gonder/get
type IletimerkeziProvider struct {
	key     string
	hash    string
	from    string
	iys     string
	iysList string
	client  *http.Client
	log     *slog.Logger
}

// NewIletimerkezi, İleti Merkezi sağlayıcısını yapılandırmayla kurar.
func NewIletimerkezi(cfg config.Notify, log *slog.Logger) *IletimerkeziProvider {
	iys := cfg.IletimerkeziIYS
	if iys == "" {
		iys = "0"
	}
	iysList := cfg.IletimerkeziIYSList
	if iysList == "" {
		iysList = "BIREYSEL"
	}
	return &IletimerkeziProvider{
		key:     cfg.IletimerkeziKey,
		hash:    cfg.IletimerkeziHash.Value(),
		from:    cfg.SMSFrom,
		iys:     iys,
		iysList: iysList,
		client:  &http.Client{Timeout: 30 * time.Second},
		log:     log,
	}
}

func (p *IletimerkeziProvider) Name() string { return "iletimerkezi" }

type iletimerkeziResponse struct {
	XMLName xml.Name `xml:"response"`
	Status  struct {
		Code    string `xml:"code"`
		Message string `xml:"message"`
	} `xml:"status"`
	Order struct {
		ID string `xml:"id"`
	} `xml:"order"`
}

// Send, tek alıcıya SMS gönderir.
func (p *IletimerkeziProvider) Send(ctx context.Context, to, body string) error {
	if p.key == "" || p.hash == "" || p.from == "" {
		return &ProviderError{
			Provider: "iletimerkezi",
			Code:     "config",
			Message:  "İleti Merkezi kimlik bilgileri veya mesaj başlığı yapılandırılmamış",
		}
	}

	no := normalizeTRMobile(to)
	if no == "" {
		return &ProviderError{Provider: "iletimerkezi", Code: "452", Message: "geçersiz telefon numarası"}
	}

	params := url.Values{}
	params.Set("key", p.key)
	params.Set("hash", p.hash)
	params.Set("text", body)
	params.Set("receipents", no) // API parametre adı (yazım hatası dokümanda böyle)
	params.Set("sender", p.from)
	params.Set("iys", p.iys)
	if p.iys == "1" {
		params.Set("iysList", p.iysList)
	}

	reqURL := iletimerkeziSendURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/xml")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("iletimerkezi: istek: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return err
	}

	var out iletimerkeziResponse
	if err := xml.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("iletimerkezi: yanıt çözümleme: %w", err)
	}

	code := strings.TrimSpace(out.Status.Code)
	if code == "200" {
		p.log.InfoContext(ctx, "iletimerkezi sms kuyruğa alındı",
			slog.String("code", code),
			slog.String("order_id", out.Order.ID),
			slog.String("to", no),
		)
		return nil
	}
	msg := strings.TrimSpace(out.Status.Message)
	if msg == "" {
		msg = strings.TrimSpace(string(respBody))
	}
	return &ProviderError{
		Provider: "iletimerkezi",
		Code:     code,
		Message:  msg,
	}
}
