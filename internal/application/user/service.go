package user

import (
	"context"

	"github.com/zatrano/gocore/pkg/pagination"
)

// Service, kullanıcı use-case'lerinin okuma yüzeyi (facade).
// Transport katmanı (HTTP/GoUI) buna bakar; CQRS handler'lar içeride kalır.
type Service struct {
	register     *RegisterHandler
	activate     *ActivateHandler
	changeEmail  *ChangeEmailHandler
	changePhone  *ChangePhoneHandler
	changeName   *ChangeNameHandler
	changeRole   *ChangeRoleHandler
	changeLocale *ChangeLocaleHandler
	del          *DeleteHandler
	restore      *RestoreHandler
	get          *GetHandler
	list         *ListHandler
	access       Access
}

// ServiceDeps, Service bağımlılıklarını gruplar.
type ServiceDeps struct {
	Register     *RegisterHandler
	Activate     *ActivateHandler
	ChangeEmail  *ChangeEmailHandler
	ChangePhone  *ChangePhoneHandler
	ChangeName   *ChangeNameHandler
	ChangeRole   *ChangeRoleHandler
	ChangeLocale *ChangeLocaleHandler
	Delete       *DeleteHandler
	Restore      *RestoreHandler
	Get          *GetHandler
	List         *ListHandler
	Access       Access
}

// NewService, kullanıcı facade'ini kurar. Handler alanları nil olabilir; yalnızca
// Access() kullanan testlerde Access dolu yeterlidir.
func NewService(d ServiceDeps) *Service {
	return &Service{
		register: d.Register, activate: d.Activate, changeEmail: d.ChangeEmail,
		changePhone: d.ChangePhone, changeName: d.ChangeName, changeRole: d.ChangeRole,
		changeLocale: d.ChangeLocale, del: d.Delete, restore: d.Restore,
		get: d.Get, list: d.List, access: d.Access,
	}
}

// Access, rol tabanlı yetki kontrollerini döner.
func (s *Service) Access() Access { return s.access }

func (s *Service) Register(ctx context.Context, cmd RegisterCommand) (View, error) {
	return s.register.Handle(ctx, cmd)
}

func (s *Service) Activate(ctx context.Context, cmd ActivateCommand) error {
	return s.activate.Handle(ctx, cmd)
}

func (s *Service) ChangeEmail(ctx context.Context, cmd ChangeEmailCommand) (View, error) {
	return s.changeEmail.Handle(ctx, cmd)
}

func (s *Service) ChangePhone(ctx context.Context, cmd ChangePhoneCommand) (View, error) {
	return s.changePhone.Handle(ctx, cmd)
}

func (s *Service) ChangeName(ctx context.Context, cmd ChangeNameCommand) (View, error) {
	return s.changeName.Handle(ctx, cmd)
}

func (s *Service) ChangeRole(ctx context.Context, cmd ChangeRoleCommand) (View, error) {
	return s.changeRole.Handle(ctx, cmd)
}

func (s *Service) ChangeLocale(ctx context.Context, cmd ChangeLocaleCommand) (View, error) {
	return s.changeLocale.Handle(ctx, cmd)
}

func (s *Service) Delete(ctx context.Context, cmd DeleteCommand) error {
	return s.del.Handle(ctx, cmd)
}

func (s *Service) Restore(ctx context.Context, cmd RestoreCommand) error {
	return s.restore.Handle(ctx, cmd)
}

func (s *Service) Get(ctx context.Context, q GetQuery) (View, error) {
	return s.get.Handle(ctx, q)
}

func (s *Service) List(ctx context.Context, q ListQuery) (pagination.Page[View], error) {
	return s.list.Handle(ctx, q)
}
