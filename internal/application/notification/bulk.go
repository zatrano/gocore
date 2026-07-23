package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	appidempotency "github.com/zatrano/gocore/internal/application/idempotency"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainnotif "github.com/zatrano/gocore/internal/domain/notification"
	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/internal/infrastructure/logger"
)

// Recipient, tekil bir alıcıyı temsil eder. Kanal türüne göre ilgili alan
// kullanılır: inapp→Email, sms→Phone, email→Email. Locale, kullanıcının tercih
// dili veya CSV satırıdır; komut dili ile eşleşmeyenler atlanır.
type Recipient struct {
	UserID string
	Phone  string
	Email  string
	Locale string
}

// MessageContent, yönetici tarafından elle girilen (literal) mesaj içeriğidir.
// i18n anahtarı değil, doğrudan gönderilecek metindir.
type MessageContent struct {
	Title    string // sms için kullanılmaz
	Body     string
	HTMLBody string // yalnızca email; opsiyonel
}

// AsyncRunner, toplu gönderimi HTTP isteğinden ayırıp arka planda çalıştırma
// portudur. infrastructure/notification.AsyncRunner bu arayüzü karşılar.
type AsyncRunner interface {
	Go(ctx context.Context, fn func(context.Context) error) error
}

// BulkSendCommand, toplu (çok alıcılı) gönderim girdisidir.
type BulkSendCommand struct {
	Channel    Channel
	Content    MessageContent
	Recipients []Recipient
	Locale     string // hedef dil: yalnızca bu tercih dilindeki alıcılara gider
}

// InvalidRecipient, doğrulamayı geçemeyen bir alıcıyı ve nedenini kaydeder.
type InvalidRecipient struct {
	Index  int    `json:"index"`
	Reason string `json:"reason"`
}

// BulkResult, toplu gönderim özetidir. Accepted alıcılar arka planda gönderilir;
// gerçek teslim hataları loglanır (yanıtta yer almaz).
type BulkResult struct {
	Total    int                `json:"total"`
	Accepted int                `json:"accepted"`
	Invalid  []InvalidRecipient `json:"invalid"`
}

// BulkEnqueuer, toplu gönderim öğelerini kalıcı kuyruğa yazar.
type BulkEnqueuer interface {
	Enqueue(ctx context.Context, cmd SendCommand, idempotencyKey string) error
}

// ManualSender, yönetici kaynaklı elle ve toplu bildirim/SMS/e-posta gönderimini
// sağlar. Tekil gönderim senkron (anlık geri bildirim); toplu in-app senkron,
// SMS/e-posta kalıcı outbox üzerinden asenkron çalışır (BulkEnqueuer yoksa runner).
type ManualSender struct {
	dispatcher *Dispatcher
	runner     AsyncRunner
	enqueuer   BulkEnqueuer
	log        *slog.Logger
	users      UserDirectory
	resolver   RecipientResolver
	idem       *appidempotency.Service
	publisher  appshared.EventPublisher
}

// ManualSenderDeps, ManualSender bağımlılıklarını gruplar.
type ManualSenderDeps struct {
	Dispatcher *Dispatcher
	Runner     AsyncRunner
	Log        *slog.Logger
	Users      UserDirectory
	Resolver   RecipientResolver // nil ok
	Idem       *appidempotency.Service
	Enqueuer   BulkEnqueuer             // optional
	Publisher  appshared.EventPublisher // optional
}

// NewManualSender, ManualSender'ı kurar. users, "tüm kullanıcılara" gönderim için
// gerekir (nil ise yalnızca elle/liste gönderimi çalışır). resolver nil ise
// alıcı çözümlemesi yapılmaz (testler).
func NewManualSender(d ManualSenderDeps) *ManualSender {
	resolver := d.Resolver
	if resolver == nil {
		resolver = noopResolver{}
	}
	return &ManualSender{
		dispatcher: d.Dispatcher, runner: d.Runner, log: d.Log, users: d.Users,
		resolver: resolver, idem: d.Idem, enqueuer: d.Enqueuer, publisher: d.Publisher,
	}
}

func (s *ManualSender) publish(ctx context.Context, events ...shared.DomainEvent) {
	if s.publisher == nil || len(events) == 0 {
		return
	}
	_ = s.publisher.Publish(ctx, events...)
}

func notificationAggregateID(ctx context.Context) string {
	if id := logger.CorrelationID(ctx); id != "" {
		return id
	}
	return fmt.Sprintf("notif-%d", time.Now().UnixNano())
}

// SendOne, tek bir alıcıya senkron gönderim yapar; doğrulama/teslim hatasını
// doğrudan döner (çağırana anlık geri bildirim).
func (s *ManualSender) SendOne(ctx context.Context, channel Channel, r Recipient, content MessageContent) error {
	key := appidempotency.KeyFromContext(ctx)
	if s.idem == nil || key == "" {
		return s.sendOne(ctx, channel, r, content)
	}
	_, err := s.idem.Run(ctx, appidempotency.ScopeNotificationSend, key, logger.UserID(ctx), "", func() (any, error) {
		return true, s.sendOne(ctx, channel, r, content)
	})
	return err
}

func (s *ManualSender) sendOne(ctx context.Context, channel Channel, r Recipient, content MessageContent) error {
	resolved, err := s.resolver.Resolve(ctx, channel, r)
	if err != nil {
		return err
	}
	if err := s.dispatcher.Send(ctx, s.build(channel, resolved, content, resolved.Locale)); err != nil {
		return err
	}
	s.publish(ctx, domainnotif.NewManualSendAcceptedEvent(notificationAggregateID(ctx), string(channel)))
	return nil
}

// SendToAllUsers, aktif tüm kullanıcılara toplu gönderir.
// SMS için telefonu olmayan kullanıcılar geçersiz alıcı olarak sayılır.
func (s *ManualSender) SendToAllUsers(ctx context.Context, channel Channel, content MessageContent, locale string) (BulkResult, error) {
	switch channel {
	case ChannelInApp, ChannelEmail, ChannelSMS:
	default:
		return BulkResult{}, ErrAudienceUnsupported
	}
	if s.users == nil {
		return BulkResult{}, ErrUserDirectoryRequired
	}
	recipients, err := s.loadAllActiveRecipients(ctx, channel, locale)
	if err != nil {
		return BulkResult{}, err
	}
	return s.SendBulk(ctx, BulkSendCommand{
		Channel:    channel,
		Content:    content,
		Recipients: recipients,
		Locale:     locale,
	})
}

func (s *ManualSender) loadAllActiveRecipients(ctx context.Context, channel Channel, targetLocale string) ([]Recipient, error) {
	const pageSize = 100
	target := normalizeLocale(targetLocale)
	out := make([]Recipient, 0, pageSize)
	pageNum := 1
	for {
		items, hasMore, err := s.users.ListActiveContacts(ctx, pageNum, pageSize)
		if err != nil {
			return nil, err
		}
		for _, u := range items {
			userLoc := normalizeLocale(u.Locale)
			// Seçilen dil varsa yalnızca o tercih dilindeki kullanıcılara gider.
			if target != "" && userLoc != target {
				continue
			}
			r := Recipient{Locale: userLoc}
			switch channel {
			case ChannelInApp:
				r.UserID = u.ID
				r.Email = u.Email
			case ChannelEmail:
				r.Email = u.Email
			case ChannelSMS:
				r.Phone = u.Phone
			}
			out = append(out, r)
		}
		if !hasMore {
			break
		}
		pageNum++
	}
	return out, nil
}

func normalizeLocale(locale string) string {
	return strings.ToLower(strings.TrimSpace(locale))
}

// SendBulk, tüm alıcıları senkron doğrular, geçerli olanları tek bir arka plan
// görevinde sırayla gönderir ve anında bir özet döner.
func (s *ManualSender) SendBulk(ctx context.Context, cmd BulkSendCommand) (BulkResult, error) {
	key := appidempotency.KeyFromContext(ctx)
	if s.idem != nil && key != "" {
		raw, err := s.idem.Run(ctx, appidempotency.ScopeNotificationBulk, key, logger.UserID(ctx), "", func() (any, error) {
			return s.sendBulk(ctx, cmd)
		})
		if err != nil {
			return BulkResult{}, err
		}
		return decodeBulkResult(raw)
	}
	return s.sendBulk(ctx, cmd)
}

func (s *ManualSender) sendBulk(ctx context.Context, cmd BulkSendCommand) (BulkResult, error) {
	if strings.TrimSpace(cmd.Content.Body) == "" {
		return BulkResult{}, ErrContentRequired
	}
	if _, ok := s.dispatcher.senders[cmd.Channel]; !ok {
		return BulkResult{}, ErrUnsupportedChannel
	}

	result := BulkResult{Total: len(cmd.Recipients), Invalid: []InvalidRecipient{}}
	valid := make([]SendCommand, 0, len(cmd.Recipients))
	targetLocale := normalizeLocale(cmd.Locale)
	for i, r := range cmd.Recipients {
		resolved, err := s.resolver.Resolve(ctx, cmd.Channel, r)
		if err != nil {
			result.Invalid = append(result.Invalid, InvalidRecipient{Index: i, Reason: reasonCode(err)})
			continue
		}
		locale := normalizeLocale(resolved.Locale)
		if targetLocale != "" {
			if locale != "" && locale != targetLocale {
				result.Invalid = append(result.Invalid, InvalidRecipient{Index: i, Reason: "locale_mismatch"})
				continue
			}
			if locale == "" {
				locale = targetLocale
			}
		}
		sc := s.build(cmd.Channel, resolved, cmd.Content, locale)
		if err := sc.validate(); err != nil {
			result.Invalid = append(result.Invalid, InvalidRecipient{Index: i, Reason: reasonCode(err)})
			continue
		}
		valid = append(valid, sc)
	}
	result.Accepted = len(valid)

	if len(valid) > 0 {
		// In-app kalıcı DB yazımıdır; outbox gecikmesi/idempotency tuzağı olmadan
		// anında kaydedilir. SMS/e-posta dış sağlayıcı çağrıları outbox'ta kalır.
		if cmd.Channel == ChannelInApp {
			sent := 0
			for i, sc := range valid {
				if err := s.dispatcher.Send(ctx, sc); err != nil {
					if s.log != nil {
						s.log.ErrorContext(ctx, "in-app toplu gönderim öğesi başarısız",
							slog.String("error", err.Error()),
						)
					}
					result.Invalid = append(result.Invalid, InvalidRecipient{
						Index: i, Reason: reasonCode(err),
					})
					continue
				}
				sent++
			}
			result.Accepted = sent
			if s.log != nil {
				s.log.InfoContext(ctx, "in-app toplu gönderim tamamlandı",
					slog.Int("accepted", sent),
					slog.Int("invalid", len(result.Invalid)),
				)
			}
		} else if s.enqueuer != nil {
			batchID := uuid.NewString()
			enqueued := 0
			for i, sc := range valid {
				key := fmt.Sprintf("notif-bulk:%s:%s:%s", batchID, string(cmd.Channel), recipientKey(sc))
				if err := s.enqueuer.Enqueue(ctx, sc, key); err != nil {
					if s.log != nil {
						s.log.ErrorContext(ctx, "toplu gönderim outbox enqueue başarısız",
							slog.String("channel", string(sc.Channel)),
							slog.String("error", err.Error()),
						)
					}
					result.Invalid = append(result.Invalid, InvalidRecipient{
						Index: i, Reason: "notification.enqueue_failed",
					})
					continue
				}
				enqueued++
			}
			result.Accepted = enqueued
			if s.log != nil {
				s.log.InfoContext(ctx, "toplu gönderim outbox'a alındı",
					slog.String("channel", string(cmd.Channel)),
					slog.String("batch_id", batchID),
					slog.Int("accepted", enqueued),
				)
			}
		} else if s.runner != nil {
			_ = s.runner.Go(ctx, func(ctx context.Context) error {
				var failed int
				for _, sc := range valid {
					if err := s.dispatcher.Send(ctx, sc); err != nil {
						failed++
						if s.log != nil {
							s.log.ErrorContext(ctx, "toplu gönderim öğesi başarısız",
								slog.String("channel", string(sc.Channel)),
								slog.String("error", err.Error()),
							)
						}
					}
				}
				if s.log != nil {
					s.log.InfoContext(ctx, "toplu gönderim tamamlandı",
						slog.String("channel", string(cmd.Channel)),
						slog.Int("accepted", len(valid)),
						slog.Int("failed", failed),
					)
				}
				return nil
			})
		} else {
			// Senkron fallback (testler): doğrudan gönder.
			sent := 0
			for i, sc := range valid {
				if err := s.dispatcher.Send(ctx, sc); err != nil {
					result.Invalid = append(result.Invalid, InvalidRecipient{
						Index: i, Reason: reasonCode(err),
					})
					continue
				}
				sent++
			}
			result.Accepted = sent
		}
	}
	if result.Accepted > 0 {
		s.publish(ctx, domainnotif.NewBulkSendAcceptedEvent(
			notificationAggregateID(ctx),
			string(cmd.Channel),
			result.Total,
			result.Accepted,
			len(result.Invalid),
		))
	}
	return result, nil
}

func recipientKey(sc SendCommand) string {
	switch {
	case sc.UserID != "":
		return sc.UserID
	case sc.Email != "":
		return sc.Email
	case sc.Phone != "":
		return sc.Phone
	default:
		return "unknown"
	}
}

func decodeBulkResult(raw any) (BulkResult, error) {
	if res, ok := raw.(BulkResult); ok {
		return res, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return BulkResult{}, err
	}
	var res BulkResult
	if err := json.Unmarshal(b, &res); err != nil {
		return BulkResult{}, err
	}
	return res, nil
}

// build, bir alıcı + içeriği literal (fallback) alanlarla SendCommand'a çevirir.
// i18n anahtarları boş bırakılır; Dispatcher fallback metni kullanır.
func (s *ManualSender) build(channel Channel, r Recipient, content MessageContent, locale string) SendCommand {
	return SendCommand{
		Channel:          channel,
		UserID:           r.UserID,
		Phone:            r.Phone,
		Email:            r.Email,
		TitleFallback:    content.Title,
		BodyFallback:     content.Body,
		HTMLBodyFallback: content.HTMLBody,
		Locale:           locale,
	}
}

// reasonCode, doğrulama hatasından makine-okur bir kod çıkarır (yanıtta alıcı
// başına neden göstermek için).
func reasonCode(err error) string {
	if de, ok := shared.AsDomainError(err); ok {
		return de.Code
	}
	return err.Error()
}
