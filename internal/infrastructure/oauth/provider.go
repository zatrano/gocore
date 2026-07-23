// Package oauth, auth.OAuthProvider portunun golang.org/x/oauth2 tabanlı
// implementasyonlarını (google, github) ve config'ten sağlayıcı üreten bir
// factory sağlar. Kimlik bilgileri boşsa ilgili sağlayıcı devre dışı kalır.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	appauth "github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/infrastructure/config"
)

// userInfoFetcher, exchange sonrası kullanıcı bilgisini sağlayıcıya özgü şekilde
// çeker ve normalize eder.
type userInfoFetcher func(ctx context.Context, client *http.Client) (appauth.OAuthUserInfo, error)

// provider, tek bir OAuth sağlayıcısının genel implementasyonudur.
type provider struct {
	name    string
	cfg     *oauth2.Config
	fetchFn userInfoFetcher
}

func (p *provider) Name() string { return p.name }

func (p *provider) AuthCodeURL(state string) string {
	return p.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *provider) Exchange(ctx context.Context, code string) (appauth.OAuthUserInfo, error) {
	tok, err := p.cfg.Exchange(ctx, code)
	if err != nil {
		return appauth.OAuthUserInfo{}, fmt.Errorf("oauth: exchange: %w", err)
	}
	client := p.cfg.Client(ctx, tok)
	return p.fetchFn(ctx, client)
}

// New, config'te tanımlı ve kimlik bilgisi dolu olan sağlayıcıları üretir.
// callbackBase boşsa cfg.CallbackBaseURL kullanılır (API varsayılanı).
func New(cfg config.OAuth, callbackBase string) []appauth.OAuthProvider {
	if callbackBase == "" {
		callbackBase = cfg.CallbackBaseURL
	}
	callbackBase = strings.TrimSuffix(callbackBase, "/")
	var providers []appauth.OAuthProvider

	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret.Value() != "" {
		providers = append(providers, &provider{
			name: "google",
			cfg: &oauth2.Config{
				ClientID:     cfg.GoogleClientID,
				ClientSecret: cfg.GoogleClientSecret.Value(),
				RedirectURL:  callbackURL(callbackBase, "google"),
				Scopes:       []string{"openid", "email", "profile"},
				Endpoint:     endpoints.Google,
			},
			fetchFn: fetchGoogleUser,
		})
	}

	if cfg.GithubClientID != "" && cfg.GithubClientSecret.Value() != "" {
		providers = append(providers, &provider{
			name: "github",
			cfg: &oauth2.Config{
				ClientID:     cfg.GithubClientID,
				ClientSecret: cfg.GithubClientSecret.Value(),
				RedirectURL:  callbackURL(callbackBase, "github"),
				Scopes:       []string{"read:user", "user:email"},
				Endpoint:     endpoints.GitHub,
			},
			fetchFn: fetchGithubUser,
		})
	}

	return providers
}

func callbackURL(base, provider string) string {
	return base + "/" + provider + "/callback"
}

// fetchGoogleUser, Google OIDC userinfo ucundan bilgi çeker.
func fetchGoogleUser(ctx context.Context, client *http.Client) (appauth.OAuthUserInfo, error) {
	var body struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := getJSON(ctx, client, "https://openidconnect.googleapis.com/v1/userinfo", &body); err != nil {
		return appauth.OAuthUserInfo{}, err
	}
	return appauth.OAuthUserInfo{Email: body.Email, Name: body.Name, ProviderUserID: body.Sub}, nil
}

// fetchGithubUser, GitHub API'den kullanıcı ve birincil e-postayı çeker.
func fetchGithubUser(ctx context.Context, client *http.Client) (appauth.OAuthUserInfo, error) {
	var user struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := getJSON(ctx, client, "https://api.github.com/user", &user); err != nil {
		return appauth.OAuthUserInfo{}, err
	}
	email := user.Email
	if email == "" {
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := getJSON(ctx, client, "https://api.github.com/user/emails", &emails); err == nil {
			for _, e := range emails {
				if e.Primary && e.Verified {
					email = e.Email
					break
				}
			}
		}
	}
	name := user.Name
	if name == "" {
		name = user.Login
	}
	return appauth.OAuthUserInfo{Email: email, Name: name, ProviderUserID: fmt.Sprintf("%d", user.ID)}, nil
}

// getJSON, verilen URL'e GET yapar ve gövdeyi out'a çözer.
func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("oauth: userinfo %d: %s", resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
