package notification

import (
	"context"

	"github.com/zatrano/gocore/pkg/pagination"
)

// Service, bildirim use-case'lerinin okuma/yazma yüzeyi (facade).
// Transport katmanı (HTTP/GoUI) buna bakar; CQRS handler'lar içeride kalır.
type Service struct {
	list        *ListHandler
	markRead    *MarkReadHandler
	markAllRead *MarkAllReadHandler
	deleteOne   *DeleteHandler
	deleteAll   *DeleteAllHandler
	unread      *UnreadCountHandler
	sender      *ManualSender
}

// ServiceDeps, Service bağımlılıklarını gruplar.
type ServiceDeps struct {
	List        *ListHandler
	MarkRead    *MarkReadHandler
	MarkAllRead *MarkAllReadHandler
	DeleteOne   *DeleteHandler
	DeleteAll   *DeleteAllHandler
	Unread      *UnreadCountHandler
	Sender      *ManualSender
}

// NewService, bildirim facade'ini kurar.
func NewService(d ServiceDeps) *Service {
	return &Service{
		list: d.List, markRead: d.MarkRead, markAllRead: d.MarkAllRead,
		deleteOne: d.DeleteOne, deleteAll: d.DeleteAll, unread: d.Unread,
		sender: d.Sender,
	}
}

// Sender, elle/toplu gönderim bileşenini döner.
func (s *Service) Sender() *ManualSender { return s.sender }

func (s *Service) List(ctx context.Context, q ListMyQuery) (pagination.Page[View], error) {
	return s.list.Handle(ctx, q)
}

func (s *Service) MarkRead(ctx context.Context, cmd MarkReadCommand) error {
	return s.markRead.Handle(ctx, cmd)
}

func (s *Service) MarkAllRead(ctx context.Context, cmd MarkAllReadCommand) (int64, error) {
	return s.markAllRead.Handle(ctx, cmd)
}

func (s *Service) Delete(ctx context.Context, cmd DeleteCommand) error {
	return s.deleteOne.Handle(ctx, cmd)
}

func (s *Service) DeleteAll(ctx context.Context, cmd DeleteAllCommand) (int64, error) {
	return s.deleteAll.Handle(ctx, cmd)
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.unread.Handle(ctx, userID)
}

func (s *Service) SendOne(ctx context.Context, channel Channel, r Recipient, content MessageContent) error {
	return s.sender.SendOne(ctx, channel, r, content)
}

func (s *Service) SendToAllUsers(ctx context.Context, channel Channel, content MessageContent, locale string) (BulkResult, error) {
	return s.sender.SendToAllUsers(ctx, channel, content, locale)
}

func (s *Service) SendBulk(ctx context.Context, cmd BulkSendCommand) (BulkResult, error) {
	return s.sender.SendBulk(ctx, cmd)
}
