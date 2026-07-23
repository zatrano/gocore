package bootstrap

import (
	"strings"

	appauth "github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/infrastructure/oauth"
)

func (g *graph) wireAuth() {
	g.loginH = appauth.NewLoginHandler(appauth.LoginDeps{
		Users: g.userRepo, Hasher: g.hasher, Sessions: g.sessions, Guard: g.guard, TOTP: g.totp,
		MFARepo: g.mfaRepo, Cache: g.memCache, Tx: g.txManager, Publisher: g.publisher,
	})
	g.changePwdH = appauth.NewChangePasswordHandler(appauth.ChangePasswordDeps{
		Users: g.userRepo, Hasher: g.hasher, Sessions: g.sessions, Notifier: g.authNotifier,
		Pub: g.publisher, Tx: g.txManager,
	})
	g.forgotH = appauth.NewForgotPasswordHandler(g.userRepo, g.authTokenRepo, g.authNotifier, g.cfg.Auth.ResetTokenTTL)
	g.resetH = appauth.NewResetPasswordHandler(appauth.ResetPasswordDeps{
		Users: g.userRepo, Tokens: g.authTokenRepo, Hasher: g.hasher, Sessions: g.sessions,
		Pub: g.publisher, Tx: g.txManager,
	})
	g.emailVerifier = appauth.NewEmailVerifier(g.userRepo, g.authTokenRepo, g.authNotifier, g.publisher, g.txManager, g.cfg.Auth.VerifyTokenTTL)
	g.mfaH = appauth.NewMFAHandler(appauth.MFADeps{
		Users: g.userRepo, MFARepo: g.mfaRepo, TOTP: g.totp, Sessions: g.sessions, Pub: g.publisher, Tx: g.txManager,
	})
	oauthAPI := oauth.New(g.cfg.OAuth, g.cfg.OAuth.CallbackBaseURL)
	webOAuthBase := strings.TrimSuffix(g.cfg.Auth.EmailLinkBaseURL, "/") + "/auth/oauth"
	oauthWeb := oauth.New(g.cfg.OAuth, webOAuthBase)
	g.oauthH = appauth.NewOAuthHandler(appauth.OAuthDeps{
		Providers: oauthAPI, Users: g.userRepo, Hasher: g.hasher, Sessions: g.sessions,
		Pub: g.publisher, Tx: g.txManager, DefaultLocale: g.cfg.I18n.DefaultLocale,
	})
	g.oauthWebH = appauth.NewOAuthHandler(appauth.OAuthDeps{
		Providers: oauthWeb, Users: g.userRepo, Hasher: g.hasher, Sessions: g.sessions,
		Pub: g.publisher, Tx: g.txManager, DefaultLocale: g.cfg.I18n.DefaultLocale,
	})
	authBase := appauth.ServiceDeps{
		Login: g.loginH, Sessions: g.sessions, ChangePwd: g.changePwdH,
		Forgot: g.forgotH, Reset: g.resetH, Verifier: g.emailVerifier, MFA: g.mfaH,
	}
	g.authService = appauth.NewService(appauth.ServiceDeps{
		Login: authBase.Login, Sessions: authBase.Sessions, ChangePwd: authBase.ChangePwd,
		Forgot: authBase.Forgot, Reset: authBase.Reset, Verifier: authBase.Verifier,
		MFA: authBase.MFA, OAuth: g.oauthH,
	})
	g.authServiceWeb = appauth.NewService(appauth.ServiceDeps{
		Login: authBase.Login, Sessions: authBase.Sessions, ChangePwd: authBase.ChangePwd,
		Forgot: authBase.Forgot, Reset: authBase.Reset, Verifier: authBase.Verifier,
		MFA: authBase.MFA, OAuth: g.oauthWebH,
	})
}
