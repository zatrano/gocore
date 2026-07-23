package idempotency

// İşlem kapsamları — scope + key + actor_id benzersiz çifti tekrarı engeller.
const (
	ScopePaymentInit      = "payment.init"
	ScopeNotificationSend = "notification.send"
	ScopeNotificationBulk = "notification.bulk"
	ScopeFormPost         = "form.post"
	ScopeAPI              = "api"
)
