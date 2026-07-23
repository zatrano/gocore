package bootstrap

import (
	appaudit "github.com/zatrano/gocore/internal/application/audit"
	appcontact "github.com/zatrano/gocore/internal/application/contact"
	appnotif "github.com/zatrano/gocore/internal/application/notification"
	appuser "github.com/zatrano/gocore/internal/application/user"
)

func (g *graph) wireUser() {
	g.localePolicy = appuser.NewLocalePolicy(g.cfg.I18n.DefaultLocale, g.cfg.I18n.Supported)
	g.registerH = appuser.NewRegisterHandler(g.userRepo, g.hasher, g.publisher, g.txManager, g.localePolicy, g.roleChecker)
	g.activateH = appuser.NewActivateHandler(g.userRepo, g.publisher, g.txManager)
	g.changeEmailH = appuser.NewChangeEmailHandler(g.userRepo, g.publisher, g.txManager, g.userAccess)
	g.changePhoneH = appuser.NewChangePhoneHandler(g.userRepo, g.publisher, g.txManager, g.userAccess)
	g.changeLocaleH = appuser.NewChangeLocaleHandler(g.userRepo, g.publisher, g.txManager, g.localePolicy)
	g.changeNameH = appuser.NewChangeNameHandler(g.userRepo, g.publisher, g.txManager, g.userAccess)
	g.changeRoleH = appuser.NewChangeRoleHandler(g.userRepo, g.publisher, g.sessions, g.txManager, g.roleChecker)
	g.deleteH = appuser.NewDeleteHandler(g.userRepo, g.publisher, g.txManager)
	g.restoreH = appuser.NewRestoreHandler(g.userRepo, g.publisher, g.txManager)
	g.getH = appuser.NewGetHandler(g.userRepo, g.userAccess)
	g.listH = appuser.NewListHandler(g.userRepo, g.userAccess)
	g.userService = appuser.NewService(appuser.ServiceDeps{
		Register: g.registerH, Activate: g.activateH, ChangeEmail: g.changeEmailH,
		ChangePhone: g.changePhoneH, ChangeName: g.changeNameH, ChangeRole: g.changeRoleH,
		ChangeLocale: g.changeLocaleH, Delete: g.deleteH, Restore: g.restoreH,
		Get: g.getH, List: g.listH, Access: g.userAccess,
	})
	g.auditListH = appaudit.NewListHandler(g.auditor)
	g.auditGetH = appaudit.NewGetHandler(g.auditor)
	g.auditService = appaudit.NewService(appaudit.ServiceDeps{List: g.auditListH, Get: g.auditGetH})
	g.contactSubmitH = appcontact.NewSubmitHandler(g.contactRepo, g.outboxRepo, g.publisher, g.txManager, g.cfg.Contact.RecipientEmail)
	g.contactListH = appcontact.NewListHandler(g.contactRepo)
	g.contactGetH = appcontact.NewGetHandler(g.contactRepo)
	g.contactMarkReadH = appcontact.NewMarkReadHandler(g.contactRepo)
	g.contactService = appcontact.NewService(appcontact.ServiceDeps{
		Submit: g.contactSubmitH, List: g.contactListH, Get: g.contactGetH, MarkRead: g.contactMarkReadH,
	})

	g.manualSender = appnotif.NewManualSender(appnotif.ManualSenderDeps{
		Dispatcher: g.dispatcher, Runner: g.asyncRunner, Log: g.log,
		Users:    appnotif.UserRepoDirectory{Repo: g.userRepo},
		Resolver: appnotif.UserRepoResolver{Users: g.userRepo}, Idem: g.idemSvc,
		Enqueuer: g.outboxDispatch, Publisher: g.publisher,
	})
	g.notifService = appnotif.NewService(appnotif.ServiceDeps{
		List: g.notifListH, MarkRead: g.notifMarkReadH, MarkAllRead: g.notifMarkAllReadH,
		DeleteOne: g.notifDeleteH, DeleteAll: g.notifDeleteAllH, Unread: g.notifUnreadH,
		Sender: g.manualSender,
	})
}
