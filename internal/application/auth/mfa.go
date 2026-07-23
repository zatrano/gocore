package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"image/png"
	"strings"

	"github.com/pquerna/otp"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainauth "github.com/zatrano/gocore/internal/domain/auth"
	"github.com/zatrano/gocore/internal/domain/user"
)

const recoveryCodeCount = 8

// MFAHandler, iki adımlı doğrulama (TOTP) kurulum/etkinleştirme/kapatma
// use-case'lerini işler.
type MFAHandler struct {
	users    user.Repository
	mfaRepo  MFARepository
	totp     TOTP
	sessions *SessionManager
	pub      appshared.EventPublisher
	tx       appshared.TxManager
}

// MFADeps, MFAHandler bağımlılıklarını gruplar.
type MFADeps struct {
	Users    user.Repository
	MFARepo  MFARepository
	TOTP     TOTP
	Sessions *SessionManager
	Pub      appshared.EventPublisher
	Tx       appshared.TxManager
}

// NewMFAHandler, handler'ı kurar.
func NewMFAHandler(d MFADeps) *MFAHandler {
	return &MFAHandler{
		users: d.Users, mfaRepo: d.MFARepo, totp: d.TOTP,
		sessions: d.Sessions, pub: d.Pub, tx: d.Tx,
	}
}

// SetupResult, kurulum adımının çıktısıdır: gizli anahtar ve otpauth URI
// (authenticator uygulamasına QR olarak girilir).
type SetupResult struct {
	Secret    string `json:"secret"`
	URI       string `json:"uri"`
	QRDataURI string `json:"qr_data_uri"`
}

// Setup, yeni bir TOTP sırrı üretip kullanıcıya (henüz etkin olmadan) atar.
// Kullanıcı authenticator'a ekleyip Enable ile doğrular.
func (h *MFAHandler) Setup(ctx context.Context, userID string) (SetupResult, error) {
	id, err := user.ParseID(userID)
	if err != nil {
		return SetupResult{}, err
	}

	var res SetupResult
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		u, err := h.users.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if u.MFAEnabled() {
			return user.ErrMFAAlreadyEnabled
		}
		secret, uri, err := h.totp.Generate(u.Email().String())
		if err != nil {
			return err
		}
		if err := u.ConfigureMFA(secret); err != nil {
			return err
		}
		qrDataURI, err := qrDataURI(uri)
		if err != nil {
			return err
		}
		res = SetupResult{Secret: secret, URI: uri, QRDataURI: qrDataURI}
		return h.users.Update(ctx, u)
	})
	if err != nil {
		return SetupResult{}, err
	}
	return res, nil
}

// EnableCommand, MFA etkinleştirme (kod doğrulama) girdisidir.
type EnableCommand struct {
	UserID string
	Code   string
}

// EnableResult, yalnızca etkinleştirme anında gösterilecek kurtarma kodlarıdır.
type EnableResult struct {
	RecoveryCodes []string `json:"recovery_codes"`
}

// Enable, kurulumda üretilen sır için kodu doğrular ve MFA'yı etkinleştirir.
func (h *MFAHandler) Enable(ctx context.Context, cmd EnableCommand) (EnableResult, error) {
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return EnableResult{}, err
	}
	if h.mfaRepo == nil {
		return EnableResult{}, user.ErrMFANotConfigured
	}
	codes, hashes, err := generateRecoveryCodes()
	if err != nil {
		return EnableResult{}, err
	}
	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, err = h.users.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if u.MFASecret() == "" {
			return user.ErrMFANotConfigured
		}
		step, ok := h.totp.Validate(u.MFASecret(), cmd.Code)
		if !ok {
			return ErrInvalidMFACode
		}
		if err := u.EnableMFA(); err != nil {
			return err
		}
		if err := h.users.Update(ctx, u); err != nil {
			return err
		}
		if err := h.mfaRepo.ResetTOTPStep(ctx, cmd.UserID); err != nil {
			return err
		}
		consumed, err := h.mfaRepo.ConsumeTOTPStep(ctx, cmd.UserID, step)
		if err != nil {
			return err
		}
		if !consumed {
			return ErrInvalidMFACode
		}
		if err := h.mfaRepo.ReplaceRecoveryCodes(ctx, cmd.UserID, hashes); err != nil {
			return err
		}
		if h.pub != nil {
			return h.pub.Publish(ctx, u.PullEvents()...)
		}
		return nil
	})
	if err != nil {
		return EnableResult{}, err
	}
	if h.sessions != nil {
		if err := h.sessions.RevokeAll(ctx, cmd.UserID); err != nil {
			return EnableResult{}, err
		}
	}
	return EnableResult{RecoveryCodes: codes}, nil
}

// DisableCommand, MFA kapatma girdisidir (mevcut kodla doğrulanır).
type DisableCommand struct {
	UserID string
	Code   string
}

// Disable, geçerli bir kodla iki adımlı doğrulamayı kapatır.
func (h *MFAHandler) Disable(ctx context.Context, cmd DisableCommand) error {
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return err
	}
	if h.mfaRepo == nil {
		return user.ErrMFANotConfigured
	}
	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, err = h.users.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if !u.MFAEnabled() {
			return user.ErrMFANotEnabled
		}
		step, ok := h.totp.Validate(u.MFASecret(), cmd.Code)
		if !ok {
			return ErrInvalidMFACode
		}
		consumed, err := h.mfaRepo.ConsumeTOTPStep(ctx, cmd.UserID, step)
		if err != nil {
			return err
		}
		if !consumed {
			return ErrInvalidMFACode
		}
		if err := u.DisableMFA(); err != nil {
			return err
		}
		if err := h.users.Update(ctx, u); err != nil {
			return err
		}
		if err := h.mfaRepo.DeleteRecoveryCodes(ctx, cmd.UserID); err != nil {
			return err
		}
		if err := h.mfaRepo.ResetTOTPStep(ctx, cmd.UserID); err != nil {
			return err
		}
		if h.pub != nil {
			return h.pub.Publish(ctx, u.PullEvents()...)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if h.sessions != nil {
		return h.sessions.RevokeAll(ctx, cmd.UserID)
	}
	return nil
}

func generateRecoveryCodes() ([]string, []string, error) {
	codes := make([]string, 0, recoveryCodeCount)
	hashes := make([]string, 0, recoveryCodeCount)
	encoding := base32.StdEncoding.WithPadding(base32.NoPadding)
	for range recoveryCodeCount {
		raw := make([]byte, 10) // 80 bit; çevrimdışı SHA-256 brute-force için yeterli.
		if _, err := rand.Read(raw); err != nil {
			return nil, nil, err
		}
		encoded := encoding.EncodeToString(raw)
		code := encoded[:4] + "-" + encoded[4:8] + "-" + encoded[8:12] + "-" + encoded[12:]
		codes = append(codes, code)
		hashes = append(hashes, domainauth.HashToken(normalizeRecoveryCode(code)))
	}
	return codes, hashes, nil
}

func normalizeRecoveryCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
}

func qrDataURI(uri string) (string, error) {
	key, err := otp.NewKeyFromURL(uri)
	if err != nil {
		return "", err
	}
	img, err := key.Image(220, 220)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(out.Bytes()), nil
}
