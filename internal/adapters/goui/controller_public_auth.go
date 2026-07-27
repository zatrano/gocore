package goui

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"

	appauth "github.com/zatrano/gocore/internal/application/auth"
	appcontact "github.com/zatrano/gocore/internal/application/contact"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	appuser "github.com/zatrano/gocore/internal/application/user"
	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/internal/infrastructure/security/turnstile"
	"github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/validation"
)

// publicAuthController, herkese açık ve auth ekranları için Controller üretir.
func publicAuthController(screen string) Controller {
	switch screen {
	case "home":
		return &homeController{}
	case "contact":
		return &contactController{}
	case "login":
		return &loginController{}
	case "register":
		return &registerController{}
	case "forgot":
		return &forgotController{}
	case "reset":
		return &resetController{}
	case "verify":
		return &verifyController{}
	default:
		return nil
	}
}

// --- home ---

type homeController struct{}

func (c *homeController) Mount(context.Context, *Page) error { return nil }

func (c *homeController) Render(p *Page) (string, error) {
	type feature struct{ Key, Title, Body string }
	return p.RenderView("pages.home", map[string]any{
		"LoggedIn":       p.Actor.UserID != "",
		"HeroTitle":      p.T("public.home.hero_title", "Kurumsal İş Yönetim Platformu"),
		"HeroSubtitle":   p.T("public.home.hero_subtitle", "Kullanıcı, yetkilendirme ve bildirim yönetimini tek yerden yönetin."),
		"DashboardLabel": p.T("public.nav.dashboard", "Panel"),
		"CTALabel":       p.T("public.home.cta_login", "Hemen Başla"),
		"ContactPath":    contactPath(p.Locale),
		"ContactLabel":   p.T("public.nav.contact", "İletişim"),
		"Features": []feature{
			{"users", p.T("public.home.feature1_title", "Kullanıcı Yönetimi"), p.T("public.home.feature1_body", "Kullanıcıları güvenle oluşturun, düzenleyin ve yönetin.")},
			{"rbac", p.T("public.home.feature2_title", "Dinamik RBAC"), p.T("public.home.feature2_body", "Rol ve izinleri veritabanı üzerinden esnekçe tanımlayın.")},
			{"notifications", p.T("public.home.feature3_title", "Bildirimler"), p.T("public.home.feature3_body", "Manuel, toplu ve dosyadan bildirim/e-posta gönderin.")},
		},
	})
}

func (c *homeController) HandleEvent(context.Context, *Page, string, map[string]any) error {
	return nil
}

// --- contact ---

type contactForm struct {
	Name    string `form:"name" validate:"required,min=2,max=100"`
	Email   string `form:"email" validate:"required,email" sanitize:"email"`
	Message string `form:"message" validate:"required,min=5,max=2000"`
}

type contactController struct {
	name, email, message string
	fieldErrors          map[string]string
	success              bool
}

func (c *contactController) Mount(context.Context, *Page) error { return nil }

func (c *contactController) Render(p *Page) (string, error) {
	return p.RenderView("pages.contact", map[string]any{
		"Title":            p.T("public.contact.title", "İletişim"),
		"Intro":            p.T("public.contact.intro", "Sorularınız için aşağıdaki formu doldurun veya bize doğrudan ulaşın."),
		"NameLabel":        p.T("public.contact.name", "Ad Soyad"),
		"EmailLabel":       p.T("public.contact.email", "E-posta"),
		"MessageLabel":     p.T("public.contact.message", "Mesaj"),
		"Name":             c.name,
		"Email":            c.email,
		"Message":          c.message,
		"ErrName":          c.fieldErrors["name"],
		"ErrEmail":         c.fieldErrors["email"],
		"ErrMessage":       c.fieldErrors["message"],
		"TurnstileSiteKey":  turnstileSiteKey(p),
		"TurnstileResetKey": turnstileResetKey(p),
		"SubmitLabel":      p.T("public.contact.submit", "Gönder"),
		"ReachTitle":       p.T("public.contact.reach_us", "Bize Ulaşın"),
		"PhoneLabel":       p.T("public.contact.phone", "Telefon"),
		"AddressLabel":     p.T("public.contact.address", "Adres"),
	})
}

func (c *contactController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	switch event {
	case "contact.name", "contact.email", "contact.message":
		c.applyInput(event, payloadString(payload, "value"))
		return nil
	case "contact.submit":
		return c.submit(ctx, p, payload)
	default:
		return nil
	}
}

func (c *contactController) applyInput(event, value string) {
	switch event {
	case "contact.name":
		c.name = value
	case "contact.email":
		c.email = value
	case "contact.message":
		c.message = value
	}
}

func (c *contactController) submit(ctx context.Context, p *Page, payload map[string]any) error {
	c.success = false
	c.fieldErrors = nil
	req := contactForm{
		Name:    payloadString(payload, "name"),
		Email:   payloadString(payload, "email"),
		Message: payloadString(payload, "message"),
	}
	c.name, c.email, c.message = req.Name, req.Email, req.Message
	if err := verifyTurnstileToken(ctx, p.Deps.Turnstile, turnstileToken(payload)); err != nil {
		return displayErr(ctx, err)
	}
	if err := checkForm(ctx, p, &req, &c.fieldErrors); err != nil {
		return err
	}
	c.name, c.email, c.message = req.Name, req.Email, req.Message
	if p.Deps.Contacts == nil {
		return errors.New("iletişim servisi yapılandırılmamış")
	}
	if _, err := p.Deps.Contacts.Submit(ctx, appcontact.SubmitCommand{
		Name: req.Name, Email: req.Email, Message: req.Message,
		Locale: string(i18n.LocaleFromContext(ctx)),
		IP:     p.ClientIP, UserAgent: p.UserAgent,
	}); err != nil {
		return displayErr(ctx, err)
	}
	c.name, c.email, c.message = "", "", ""
	c.success = true
	p.Notice = i18n.T(ctx, "public.contact.success", "Mesajınız alındı, en kısa sürede dönüş yapacağız.")
	return nil
}

// --- login ---

type loginForm struct {
	Email    string `form:"email" validate:"required,email" sanitize:"email"`
	Password string `form:"password" validate:"required"`
	Next     string `form:"next"`
}

type loginMFAForm struct {
	Code string `form:"mfa_code" validate:"required"`
}

type loginController struct {
	email, mfaCode, mfaChallenge, next string
	mfaRequired                        bool
	fieldErrors                        map[string]string
	providers                          []string
}

func (c *loginController) Mount(_ context.Context, p *Page) error {
	if p.Query != nil {
		c.next = p.Query["next"]
	}
	if p.Deps.Auth != nil {
		c.providers = p.Deps.Auth.OAuthProviders()
	}
	return nil
}

func (c *loginController) Render(p *Page) (string, error) {
	return p.RenderView("pages.login", map[string]any{
		"MFARequired":      c.mfaRequired,
		"MFATitle":         p.T("public.auth.mfa.title", "İki Adımlı Doğrulama"),
		"MFAIntro":         p.T("public.auth.mfa.intro", "İki adımlı doğrulama kodunuzu girin."),
		"MFACodeLabel":     p.T("public.auth.mfa.code", "Authenticator veya kurtarma kodu"),
		"MFACode":          c.mfaCode,
		"MFAChallenge":     c.mfaChallenge,
		"MFASubmitLabel":   p.T("public.auth.mfa.submit", "Doğrula"),
		"ErrMFACode":       c.fieldErrors["mfa_code"],
		"MFALinks":         []ViewLink{{Href: "/auth/login", Label: p.T("public.auth.mfa.back", "Giriş sayfasına dön")}},
		"Title":            p.T("public.auth.login.title", "Giriş"),
		"OAuthProviders":   buildOAuthProviders(p, c.providers, "login", c.next),
		"OAuthDivider":     p.T("public.auth.or", "veya"),
		"Next":             c.next,
		"EmailLabel":       p.T("public.auth.field.email", "E-posta"),
		"PasswordLabel":    p.T("public.auth.field.password", "Parola"),
		"Email":            c.email,
		"ErrEmail":         c.fieldErrors["email"],
		"ErrPassword":      c.fieldErrors["password"],
		"TurnstileSiteKey":  turnstileSiteKey(p),
		"TurnstileResetKey": turnstileResetKey(p),
		"SubmitLabel":      p.T("public.auth.login.submit", "Giriş Yap"),
		"Links": []ViewLink{
			{Href: "/auth/forgot-password", Label: p.T("public.auth.login.forgot", "Şifremi unuttum")},
			{Href: "/auth/register", Label: p.T("public.nav.register", "Kayıt")},
		},
	})
}

func (c *loginController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	switch event {
	case "login.email", "login.password", "login.mfa_code", "login.next":
		c.applyInput(event, payloadString(payload, "value"))
		return nil
	case "login.submit":
		return c.submit(ctx, p, payload)
	default:
		return nil
	}
}

func (c *loginController) applyInput(event, value string) {
	switch event {
	case "login.email":
		c.email = value
	case "login.mfa_code":
		c.mfaCode = value
	case "login.next":
		c.next = value
	}
}

func (c *loginController) submit(ctx context.Context, p *Page, payload map[string]any) error {
	c.fieldErrors = nil
	if c.mfaRequired {
		req := loginMFAForm{Code: payloadString(payload, "mfa_code")}
		if err := checkForm(ctx, p, &req, &c.fieldErrors); err != nil {
			return err
		}
		tokens, err := p.Deps.Auth.Login(ctx, appauth.LoginCommand{
			MFACode: req.Code, MFAChallenge: c.mfaChallenge,
			ClientKey: appshared.ActorFromContext(ctx).IP,
		})
		if err != nil {
			return displayErr(ctx, err)
		}
		c.mfaRequired = false
		c.mfaCode, c.mfaChallenge = "", ""
		return redirectWithAuthExchange(ctx, p, tokens, c.next)
	}

	req := loginForm{
		Email:    payloadString(payload, "email"),
		Password: payloadString(payload, "password"),
		Next:     payloadString(payload, "next"),
	}
	if req.Next == "" {
		req.Next = c.next
	}
	c.email, c.next = req.Email, req.Next
	if err := verifyTurnstileToken(ctx, p.Deps.Turnstile, turnstileToken(payload)); err != nil {
		return displayErr(ctx, err)
	}
	if err := checkForm(ctx, p, &req, &c.fieldErrors); err != nil {
		return err
	}
	c.email = req.Email
	if p.Deps.Auth == nil {
		return errors.New("giriş servisi yapılandırılmamış")
	}
	tokens, err := p.Deps.Auth.Login(ctx, appauth.LoginCommand{
		Email: req.Email, Password: req.Password,
		ClientKey: appshared.ActorFromContext(ctx).IP,
	})
	if err != nil {
		if challenge, ok := appauth.MFAChallengeFrom(err); ok {
			c.mfaRequired = true
			c.mfaChallenge = challenge
			p.Error = userMessage(ctx, err)
			return nil
		}
		return displayErr(ctx, err)
	}
	c.mfaRequired = false
	c.mfaCode = ""
	return redirectWithAuthExchange(ctx, p, tokens, req.Next)
}

// --- register ---

type registerForm struct {
	Email    string `form:"email" validate:"required,email" sanitize:"email"`
	Name     string `form:"name" validate:"required,min=2,max=100"`
	Password string `form:"password" validate:"required,min=8,max=128"`
	Phone    string `form:"phone" validate:"omitempty,phone" sanitize:"phone"`
}

type registerController struct {
	email, name, phone string
	fieldErrors        map[string]string
	providers          []string
}

func (c *registerController) Mount(_ context.Context, p *Page) error {
	if p.Deps.Auth != nil {
		c.providers = p.Deps.Auth.OAuthProviders()
	}
	return nil
}

func (c *registerController) Render(p *Page) (string, error) {
	return p.RenderView("pages.register", map[string]any{
		"Title":            p.T("public.auth.register.title", "Kayıt Ol"),
		"OAuthProviders":   buildOAuthProviders(p, c.providers, "register", ""),
		"OAuthDivider":     p.T("public.auth.or", "veya"),
		"NameLabel":        p.T("public.auth.field.name", "Ad Soyad"),
		"EmailLabel":       p.T("public.auth.field.email", "E-posta"),
		"PhoneLabel":       p.T("public.auth.field.phone", "Telefon"),
		"PasswordLabel":    p.T("public.auth.field.password", "Parola"),
		"Name":             c.name,
		"Email":            c.email,
		"Phone":            c.phone,
		"ErrName":          c.fieldErrors["name"],
		"ErrEmail":         c.fieldErrors["email"],
		"ErrPhone":         c.fieldErrors["phone"],
		"ErrPassword":      c.fieldErrors["password"],
		"TurnstileSiteKey":  turnstileSiteKey(p),
		"TurnstileResetKey": turnstileResetKey(p),
		"SubmitLabel":      p.T("public.auth.register.submit", "Kayıt Ol"),
		"Links":            []ViewLink{{Href: "/auth/login", Label: p.T("public.auth.register.login_link", "Giriş yap")}},
	})
}

func (c *registerController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	switch event {
	case "register.name", "register.email", "register.phone", "register.password":
		c.applyInput(event, payloadString(payload, "value"))
		return nil
	case "register.submit":
		return c.submit(ctx, p, payload)
	default:
		return nil
	}
}

func (c *registerController) applyInput(event, value string) {
	switch event {
	case "register.name":
		c.name = value
	case "register.email":
		c.email = value
	case "register.phone":
		c.phone = value
	}
}

func (c *registerController) submit(ctx context.Context, p *Page, payload map[string]any) error {
	c.fieldErrors = nil
	req := registerForm{
		Email:    payloadString(payload, "email"),
		Name:     payloadString(payload, "name"),
		Password: payloadString(payload, "password"),
		Phone:    payloadString(payload, "phone"),
	}
	c.email, c.name, c.phone = req.Email, req.Name, req.Phone
	if err := verifyTurnstileToken(ctx, p.Deps.Turnstile, turnstileToken(payload)); err != nil {
		return displayErr(ctx, err)
	}
	if err := checkForm(ctx, p, &req, &c.fieldErrors); err != nil {
		return err
	}
	c.email, c.name, c.phone = req.Email, req.Name, req.Phone
	if p.Deps.Users == nil {
		return errors.New("kayıt servisi yapılandırılmamış")
	}
	if _, err := p.Deps.Users.Register(ctx, appuser.RegisterCommand{
		Email: req.Email, Name: req.Name, Password: req.Password, Phone: req.Phone,
		PreferredLocale:     string(i18n.LocaleFromContext(ctx)),
		AllowPrivilegedRole: false,
	}); err != nil {
		return displayErr(ctx, err)
	}
	p.Notice = i18n.T(ctx, "success.user.registered", "kullanıcı başarıyla kaydedildi")
	p.Redirect = "/auth/login"
	return nil
}

// --- forgot ---

type forgotForm struct {
	Email string `form:"email" validate:"required,email" sanitize:"email"`
}

type forgotController struct {
	email       string
	fieldErrors map[string]string
}

func (c *forgotController) Mount(_ context.Context, p *Page) error {
	if p.Query != nil {
		c.email = p.Query["email"]
	}
	return nil
}

func (c *forgotController) Render(p *Page) (string, error) {
	return p.RenderView("pages.forgot", map[string]any{
		"Title":            p.T("public.auth.forgot.title", "Şifremi Unuttum"),
		"EmailLabel":       p.T("public.auth.field.email", "E-posta"),
		"Email":            c.email,
		"ErrEmail":         c.fieldErrors["email"],
		"TurnstileSiteKey":  turnstileSiteKey(p),
		"TurnstileResetKey": turnstileResetKey(p),
		"SubmitLabel":      p.T("public.auth.forgot.submit", "Sıfırlama Bağlantısı Gönder"),
		"Links":            []ViewLink{{Href: "/auth/login", Label: p.T("public.auth.forgot.back", "Girişe dön")}},
	})
}

func (c *forgotController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	switch event {
	case "forgot.email":
		c.email = payloadString(payload, "value")
		return nil
	case "forgot.submit":
		return c.submit(ctx, p, payload)
	default:
		return nil
	}
}

func (c *forgotController) submit(ctx context.Context, p *Page, payload map[string]any) error {
	c.fieldErrors = nil
	req := forgotForm{Email: payloadString(payload, "email")}
	c.email = req.Email
	if err := verifyTurnstileToken(ctx, p.Deps.Turnstile, turnstileToken(payload)); err != nil {
		return displayErr(ctx, err)
	}
	if err := checkForm(ctx, p, &req, &c.fieldErrors); err != nil {
		return err
	}
	c.email = req.Email
	if p.Deps.Auth != nil {
		if err := p.Deps.Auth.ForgotPassword(ctx, appauth.ForgotPasswordCommand{Email: req.Email}); err != nil {
			return displayErr(ctx, err)
		}
	}
	p.Notice = i18n.T(ctx, "success.auth.reset_sent",
		"eğer bu e-posta kayıtlıysa, sıfırlama bağlantısı gönderildi")
	return nil
}

// --- reset ---

type resetForm struct {
	Token       string `form:"token" validate:"required"`
	NewPassword string `form:"new_password" validate:"required,min=8"`
}

type resetController struct {
	token       string
	fieldErrors map[string]string
}

func (c *resetController) Mount(_ context.Context, p *Page) error {
	if p.Query != nil {
		c.token = p.Query["token"]
	}
	return nil
}

func (c *resetController) Render(p *Page) (string, error) {
	return p.RenderView("pages.reset", map[string]any{
		"Title":            p.T("public.auth.reset.title", "Şifre Sıfırla"),
		"Token":            c.token,
		"PasswordLabel":    p.T("public.auth.field.new_password", "Yeni Parola"),
		"ErrNewPassword":   c.fieldErrors["new_password"],
		"TurnstileSiteKey":  turnstileSiteKey(p),
		"TurnstileResetKey": turnstileResetKey(p),
		"SubmitLabel":      p.T("public.auth.reset.submit", "Parolayı Sıfırla"),
		"Links":            []ViewLink{{Href: "/auth/login", Label: p.T("public.auth.reset.back", "Giriş sayfasına dön")}},
	})
}

func (c *resetController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	switch event {
	case "reset.submit":
		return c.submit(ctx, p, payload)
	default:
		return nil
	}
}

func (c *resetController) submit(ctx context.Context, p *Page, payload map[string]any) error {
	c.fieldErrors = nil
	req := resetForm{
		Token:       payloadString(payload, "token"),
		NewPassword: payloadString(payload, "new_password"),
	}
	if req.Token == "" {
		req.Token = c.token
	}
	c.token = req.Token
	if err := verifyTurnstileToken(ctx, p.Deps.Turnstile, turnstileToken(payload)); err != nil {
		return displayErr(ctx, err)
	}
	if err := checkForm(ctx, p, &req, &c.fieldErrors); err != nil {
		return err
	}
	if p.Deps.Auth == nil {
		return errors.New("şifre sıfırlama servisi yapılandırılmamış")
	}
	if err := p.Deps.Auth.ResetPassword(ctx, appauth.ResetPasswordCommand{
		Token: req.Token, NewPassword: req.NewPassword,
	}); err != nil {
		return displayErr(ctx, err)
	}
	p.Notice = i18n.T(ctx, "success.auth.password_reset", "şifreniz sıfırlandı")
	p.Redirect = "/auth/login"
	return nil
}

// --- verify ---

type resendForm struct {
	Email string `form:"email" validate:"required,email" sanitize:"email"`
}

type verifyController struct {
	email       string
	fieldErrors map[string]string
	done        bool
}

func (c *verifyController) Mount(ctx context.Context, p *Page) error {
	if p.Query != nil {
		c.email = p.Query["email"]
		token := p.Query["token"]
		if token != "" && p.Deps.Auth != nil {
			if err := p.Deps.Auth.VerifyEmail(ctx, appauth.VerifyCommand{Token: token}); err != nil {
				p.Error = userMessage(ctx, err)
				return nil
			}
			p.Notice = i18n.T(ctx, "success.auth.email_verified", "e-posta adresiniz doğrulandı")
			p.Redirect = "/auth/login"
			c.done = true
		}
	}
	return nil
}

func (c *verifyController) Render(p *Page) (string, error) {
	return p.RenderView("pages.verify", map[string]any{
		"Done":             c.done,
		"RedirectingText":  p.T("public.auth.verify.redirecting", "Yönlendiriliyorsunuz…"),
		"Title":            p.T("public.auth.verify.title", "E-posta Doğrulama"),
		"Intro":            p.T("public.auth.verify.intro", "Doğrulama e-postasını tekrar göndermek için aşağıdaki düğmeye tıklayın."),
		"EmailLabel":       p.T("public.auth.field.email", "E-posta"),
		"Email":            c.email,
		"ErrEmail":         c.fieldErrors["email"],
		"TurnstileSiteKey":  turnstileSiteKey(p),
		"TurnstileResetKey": turnstileResetKey(p),
		"SubmitLabel":      p.T("public.auth.verify.submit", "Doğrulama E-postası Gönder"),
		"Links":            []ViewLink{{Href: "/auth/login", Label: p.T("public.auth.verify.back", "Giriş sayfasına dön")}},
	})
}

func (c *verifyController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	switch event {
	case "verify.email":
		c.email = payloadString(payload, "value")
		return nil
	case "verify.resend":
		return c.resend(ctx, p, payload)
	default:
		return nil
	}
}

func (c *verifyController) resend(ctx context.Context, p *Page, payload map[string]any) error {
	c.fieldErrors = nil
	req := resendForm{Email: payloadString(payload, "email")}
	if req.Email == "" {
		req.Email = c.email
	}
	c.email = req.Email
	if err := verifyTurnstileToken(ctx, p.Deps.Turnstile, turnstileToken(payload)); err != nil {
		return displayErr(ctx, err)
	}
	if err := checkForm(ctx, p, &req, &c.fieldErrors); err != nil {
		return err
	}
	c.email = req.Email
	if p.Deps.Auth != nil {
		if err := p.Deps.Auth.ResendVerification(ctx, appauth.ResendCommand{Email: req.Email}); err != nil {
			return displayErr(ctx, err)
		}
	}
	p.Notice = i18n.T(ctx, "success.auth.verification_sent",
		"eğer bu e-posta kayıtlıysa, doğrulama bağlantısı gönderildi")
	return nil
}

// --- shared helpers ---

type viewOAuthProvider struct {
	Key   string
	Href  string
	Label string
	Icon  string
}

func buildOAuthProviders(p *Page, providers []string, from, next string) []viewOAuthProvider {
	if len(providers) == 0 {
		return nil
	}
	out := make([]viewOAuthProvider, 0, len(providers))
	for _, provider := range providers {
		href := "/auth/oauth/" + url.PathEscape(provider) + "?from=" + url.QueryEscape(from)
		if next != "" {
			href += "&next=" + url.QueryEscape(next)
		}
		out = append(out, viewOAuthProvider{
			Key:   provider,
			Href:  href,
			Label: oauthButtonLabel(p, provider, from),
			Icon:  oauthIcon(provider),
		})
	}
	return out
}

func turnstileSiteKey(p *Page) string {
	if !viewTurnstileEnabled(p) {
		return ""
	}
	return p.Deps.TurnstileSiteKey
}

func turnstileResetKey(p *Page) string {
	if p == nil || !viewTurnstileEnabled(p) {
		return ""
	}
	return p.EventNonce
}

func turnstileToken(payload map[string]any) string {
	if v := payloadString(payload, "cf-turnstile-response"); v != "" {
		return v
	}
	return payloadString(payload, "turnstile_token")
}

func checkForm(ctx context.Context, p *Page, req any, fieldErrors *map[string]string) error {
	if p.Deps.Validate == nil {
		return errors.New("doğrulama servisi yapılandırılmamış")
	}
	if err := validation.Check(p.Deps.Validate, req); err != nil {
		*fieldErrors = toFieldErrors(ctx, err)
		return displayErr(ctx, err)
	}
	return nil
}

func verifyTurnstileToken(ctx context.Context, v turnstile.Verifier, token string) error {
	if v == nil || !v.Enabled() {
		return nil
	}
	return v.Verify(ctx, token, appshared.ActorFromContext(ctx).IP)
}

func displayErr(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	return errors.New(userMessage(ctx, err))
}

func userMessage(ctx context.Context, err error) string {
	if err == nil {
		return ""
	}
	var inv validation.InvalidFields
	if errors.As(err, &inv) {
		return i18n.T(ctx, "validation.detail", "Bir veya daha fazla alan geçersiz")
	}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		return i18n.T(ctx, "validation.detail", "Bir veya daha fazla alan geçersiz")
	}
	if de, ok := shared.AsDomainError(err); ok {
		return i18n.T(ctx, de.Code, de.Message)
	}
	if errors.Is(err, errAuthCacheUnavailable) {
		return errAuthCacheUnavailable.Error()
	}
	return i18n.T(ctx, "internal_error", "Beklenmeyen bir hata oluştu")
}

func toFieldErrors(ctx context.Context, err error) map[string]string {
	var inv validation.InvalidFields
	if errors.As(err, &inv) {
		return inv.FieldMap(ctx)
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return nil
	}
	out := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		out[formFieldName(fe.Field())] = validation.FieldMessage(ctx, fe)
	}
	return out
}

func formFieldName(structField string) string {
	switch structField {
	case "Email":
		return "email"
	case "Password":
		return "password"
	case "Name":
		return "name"
	case "Phone":
		return "phone"
	case "Message":
		return "message"
	case "MFACode":
		return "mfa_code"
	case "NewPassword":
		return "new_password"
	case "Token":
		return "token"
	default:
		return strings.ToLower(structField)
	}
}

func oauthButtonLabel(p *Page, provider, from string) string {
	register := from == "register"
	switch strings.ToLower(provider) {
	case "google":
		if register {
			return p.T("public.auth.oauth_google_register", "Google ile Kayıt Ol")
		}
		return p.T("public.auth.oauth_google", "Google ile Giriş")
	case "github":
		if register {
			return p.T("public.auth.oauth_github_register", "GitHub ile Kayıt Ol")
		}
		return p.T("public.auth.oauth_github", "GitHub ile Giriş")
	default:
		return oauthLabel(provider) + " ile Giriş"
	}
}

func oauthIcon(provider string) string {
	switch strings.ToLower(provider) {
	case "google":
		return `<svg class="oauth-btn__icon" viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/><path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/><path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/></svg>`
	case "github":
		return `<svg class="oauth-btn__icon" viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path fill="currentColor" d="M12 .5C5.65.5.5 5.65.5 12a11.5 11.5 0 0 0 7.86 10.93c.58.11.79-.25.79-.56 0-.28-.01-1.02-.02-2-3.2.7-3.88-1.54-3.88-1.54-.53-1.35-1.3-1.71-1.3-1.71-1.06-.73.08-.72.08-.72 1.17.08 1.79 1.2 1.79 1.2 1.04 1.78 2.73 1.27 3.4.97.11-.76.41-1.27.74-1.56-2.55-.29-5.24-1.28-5.24-5.7 0-1.26.45-2.29 1.18-3.1-.12-.29-.51-1.47.11-3.06 0 0 .96-.31 3.15 1.18a10.9 10.9 0 0 1 2.88-.39c.98 0 1.96.13 2.88.39 2.19-1.49 3.15-1.18 3.15-1.18.62 1.59.23 2.77.11 3.06.73.81 1.18 1.84 1.18 3.1 0 4.43-2.7 5.41-5.27 5.69.42.36.8 1.08.8 2.18 0 1.58-.01 2.85-.01 3.24 0 .31.21.68.8.56A11.5 11.5 0 0 0 23.5 12C23.5 5.65 18.35.5 12 .5z"/></svg>`
	default:
		return ""
	}
}

func oauthLabel(provider string) string {
	switch strings.ToLower(provider) {
	case "google":
		return "Google"
	case "github":
		return "GitHub"
	default:
		return provider
	}
}
