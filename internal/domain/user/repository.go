package user

import (
	"context"

	"github.com/zatrano/gocore/pkg/pagination"
)

// Repository, User aggregate'i için persistence portudur (Dependency Inversion).
// Domain katmanı yalnızca bu arayüzü bilir; PostgreSQL implementasyonu adapters
// katmanındadır. Bu sayede domain, altyapıdan tamamen bağımsız kalır.
//
// Tasarım notları:
//   - N+1 önleme: koleksiyon dönen metodlar tek sorguda tüm veriyi çeker;
//     ilişki gerektiğinde FindByIDs gibi batch metodlar kullanılır.
//   - Tüm metodlar context alır (timeout / cancellation / trace propagation).
type Repository interface {
	// Save, yeni bir kullanıcıyı kalıcılaştırır. E-posta benzersizliği ihlalinde
	// ErrEmailAlreadyExists döner.
	Save(ctx context.Context, u *User) error

	// Update, mevcut kullanıcıyı optimistic locking ile günceller.
	Update(ctx context.Context, u *User) error

	// FindByID, canlı (silinmemiş) kullanıcıyı kimliğine göre getirir. Yoksa ErrNotFound.
	FindByID(ctx context.Context, id ID) (*User, error)

	// FindByIDIncludeDeleted, silinmiş dahil kullanıcıyı getirir (yönetim/detay için).
	FindByIDIncludeDeleted(ctx context.Context, id ID) (*User, error)

	// FindByEmail, e-postaya göre kullanıcı getirir. Yoksa ErrNotFound.
	FindByEmail(ctx context.Context, email Email) (*User, error)

	// FindByIDs, birden fazla kullanıcıyı TEK sorguda getirir (batch). N+1
	// sorgu problemini önlemek için ilişkisel yüklemelerde kullanılır.
	FindByIDs(ctx context.Context, ids []ID) ([]*User, error)

	// List, sayfa tabanlı (offset) veya keyset (Cursor) sayfalama ile kullanıcıları listeler.
	List(ctx context.Context, filter ListFilter, page pagination.Request) (pagination.Page[*User], error)

	// ExistsByEmail, e-posta kayıtlı mı diye kontrol eder (hafif sorgu).
	ExistsByEmail(ctx context.Context, email Email) (bool, error)

	// Delete, kullanıcıyı YAZILIMSAL olarak siler (soft delete): kayıt fiziksel
	// olarak korunur, deleted_at damgalanır ve canlı sorgularda görünmez.
	// Kayıt yoksa veya zaten silinmişse ErrNotFound döner.
	Delete(ctx context.Context, id ID) error

	// Restore, yazılımsal silmeyi geri alır. Silinmemiş/olmayan kayıtta ErrNotFound.
	Restore(ctx context.Context, id ID) error

	// HardDelete, yalnızca zaten soft-delete edilmiş bir kaydı KALICI olarak
	// (fiziksel) siler. Uygun kayıt yoksa ErrNotFound.
	HardDelete(ctx context.Context, id ID) error

	// CountActiveByRole, canlı (silinmemiş) kullanıcıları role göre sayar.
	// Son admin koruması gibi iş kurallarında kullanılır.
	CountActiveByRole(ctx context.Context, role Role) (int64, error)
}

// ListFilter, kullanıcı listeleme için filtreleme kriterleridir. Sıfır değerli
// alanlar "filtre yok" anlamına gelir (immutable, opsiyonel alanlar).
type ListFilter struct {
	// Role, belirli bir role göre filtreler (nil = tümü).
	Role *Role
	// Active, aktiflik durumuna göre filtreler (nil = tümü).
	Active *bool
	// Search, ad/e-posta üzerinde arama terimi (boş = arama yok).
	Search string
	// Deleted, silinme kapsamı: "" = yalnızca canlı, "only" = yalnızca silinenler, "all" = hepsi.
	Deleted string
}
