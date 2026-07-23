package goui

// settingsPaymentsAuditController, ayar / ödeme / audit ekranları için Controller fabrikasıdır.
func settingsPaymentsAuditController(screen string) Controller {
	switch screen {
	case "sms-settings":
		return &smsSettingsCtrl{}
	case "sms-provider":
		return &smsProviderCtrl{}
	case "payment-settings":
		return &paymentSettingsCtrl{}
	case "payment-provider":
		return &paymentProviderCtrl{}
	case "checkout":
		return &checkoutCtrl{}
	case "payments":
		return &paymentsListCtrl{}
	case "payment-show":
		return &paymentShowCtrl{}
	case "audit":
		return &auditListCtrl{}
	case "audit-show":
		return &auditShowCtrl{}
	default:
		return nil
	}
}
