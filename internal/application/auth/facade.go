package auth

import (
	"context"

	"github.com/zatrano/gocore/internal/domain/user"
)

// Service, kimlik doğrulama use-case'lerinin yüzeyi (facade).
// Transport katmanı (HTTP/GoUI) buna bakar; CQRS handler'lar içeride kalır.
type Service struct {
	login     *LoginHandler
	sessions  *SessionManager
	changePwd *ChangePasswordHandler
	forgot    *ForgotPasswordHandler
	reset     *ResetPasswordHandler
	verifier  *EmailVerifier
	mfa       *MFAHandler
	oauth     *OAuthHandler
}

// ServiceDeps, Service bağımlılıklarını gruplar.
type ServiceDeps struct {
	Login     *LoginHandler
	Sessions  *SessionManager
	ChangePwd *ChangePasswordHandler
	Forgot    *ForgotPasswordHandler
	Reset     *ResetPasswordHandler
	Verifier  *EmailVerifier
	MFA       *MFAHandler
	OAuth     *OAuthHandler
}

// NewService, kimlik doğrulama facade'ini kurar.
func NewService(d ServiceDeps) *Service {
	return &Service{
		login: d.Login, sessions: d.Sessions, changePwd: d.ChangePwd,
		forgot: d.Forgot, reset: d.Reset, verifier: d.Verifier,
		mfa: d.MFA, oauth: d.OAuth,
	}
}

// Sessions, oturum yöneticisini döner (middleware ve özel çağrılar için).
func (s *Service) Sessions() *SessionManager { return s.sessions }

// MFAHandler, MFA handler'ını döner.
func (s *Service) MFAHandler() *MFAHandler { return s.mfa }

// OAuthHandler, OAuth handler'ını döner.
func (s *Service) OAuthHandler() *OAuthHandler { return s.oauth }

// Verifier, e-posta doğrulama servisini döner.
func (s *Service) Verifier() *EmailVerifier { return s.verifier }

func (s *Service) Login(ctx context.Context, cmd LoginCommand) (TokenPair, error) {
	return s.login.Handle(ctx, cmd)
}

func (s *Service) ChangePassword(ctx context.Context, cmd ChangePasswordCommand) error {
	return s.changePwd.Handle(ctx, cmd)
}

func (s *Service) AdminSetPassword(ctx context.Context, cmd AdminSetPasswordCommand) error {
	return s.changePwd.AdminSet(ctx, cmd)
}

func (s *Service) ForgotPassword(ctx context.Context, cmd ForgotPasswordCommand) error {
	return s.forgot.Handle(ctx, cmd)
}

func (s *Service) ResetPassword(ctx context.Context, cmd ResetPasswordCommand) error {
	return s.reset.Handle(ctx, cmd)
}

func (s *Service) Issue(ctx context.Context, c Claims) (TokenPair, error) {
	return s.sessions.Issue(ctx, c)
}

func (s *Service) Verify(ctx context.Context, token string) (Claims, error) {
	return s.sessions.Verify(ctx, token)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	return s.sessions.Refresh(ctx, refreshToken)
}

func (s *Service) Logout(ctx context.Context, accessToken, refreshToken string) error {
	return s.sessions.Logout(ctx, accessToken, refreshToken)
}

func (s *Service) RevokeAll(ctx context.Context, userID string) error {
	return s.sessions.RevokeAll(ctx, userID)
}

func (s *Service) SendVerification(ctx context.Context, u *user.User) error {
	return s.verifier.Send(ctx, u)
}

func (s *Service) ResendVerification(ctx context.Context, cmd ResendCommand) error {
	return s.verifier.Resend(ctx, cmd)
}

func (s *Service) VerifyEmail(ctx context.Context, cmd VerifyCommand) error {
	return s.verifier.Verify(ctx, cmd)
}

func (s *Service) MFASetup(ctx context.Context, userID string) (SetupResult, error) {
	return s.mfa.Setup(ctx, userID)
}

func (s *Service) MFAEnable(ctx context.Context, cmd EnableCommand) (EnableResult, error) {
	return s.mfa.Enable(ctx, cmd)
}

func (s *Service) MFADisable(ctx context.Context, cmd DisableCommand) error {
	return s.mfa.Disable(ctx, cmd)
}

func (s *Service) OAuthProviders() []string {
	if s.oauth == nil {
		return nil
	}
	return s.oauth.Providers()
}

func (s *Service) OAuthAuthCodeURL(provider, state string) (string, error) {
	return s.oauth.AuthCodeURL(provider, state)
}

func (s *Service) OAuthCallback(ctx context.Context, provider, code string) (TokenPair, error) {
	return s.oauth.Callback(ctx, provider, code)
}
