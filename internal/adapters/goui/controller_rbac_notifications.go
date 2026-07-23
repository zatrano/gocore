package goui

// rbacNotificationsController, RBAC / bildirim / yükleme ekranları için Controller üretir.
func rbacNotificationsController(screen string) Controller {
	switch screen {
	case "roles":
		return &rolesController{}
	case "role-new":
		return &roleNewController{}
	case "role-show":
		return &roleShowController{}
	case "permissions":
		return &permissionsController{}
	case "notification-send":
		return &notificationSendController{}
	case "notification-bulk":
		return &notificationBulkController{}
	case "notification-upload":
		return &notificationUploadController{}
	case "uploads":
		return &uploadsController{}
	default:
		return nil
	}
}
