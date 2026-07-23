package settings

import "context"

// Repository, uygulama ayarları için persistence portudur.
type Repository interface {
	Get(ctx context.Context, key SettingKey) (string, error)
	Set(ctx context.Context, key SettingKey, value string) error
}
