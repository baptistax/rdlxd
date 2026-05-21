package reddit

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/baptistax/rdlxd/internal/config"
)

const DefaultTokenURL = "https://www.reddit.com/api/v1/access_token"

type OAuthConfig struct {
	ClientID    string
	RedirectURI string
	Scopes      []string
	UserAgent   string
	TokenURL    string
	HTTPClient  *http.Client
}

type InstalledAppFlow struct {
	Config OAuthConfig
}

func (f InstalledAppFlow) AuthorizationURL(state string) (string, error) {
	if f.Config.ClientID == "" {
		return "", fmt.Errorf("client id is required")
	}
	if f.Config.RedirectURI == "" {
		return "", fmt.Errorf("redirect uri is required")
	}
	values := url.Values{}
	values.Set("client_id", f.Config.ClientID)
	values.Set("response_type", "code")
	values.Set("state", state)
	values.Set("redirect_uri", f.Config.RedirectURI)
	values.Set("duration", "permanent")
	values.Set("scope", strings.Join(f.Config.Scopes, " "))
	return "https://www.reddit.com/api/v1/authorize?" + values.Encode(), nil
}

func (f InstalledAppFlow) ExchangeCode(ctx context.Context, code string) (*config.OAuthToken, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("authorization code is required")
	}
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", strings.TrimSpace(code))
	values.Set("redirect_uri", f.Config.RedirectURI)
	return f.postToken(ctx, values)
}

func (f InstalledAppFlow) Refresh(ctx context.Context, refreshToken string) (*config.OAuthToken, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("refresh token is required")
	}
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", strings.TrimSpace(refreshToken))
	return f.postToken(ctx, values)
}

func NewOAuthState() (string, error) {
	data := make([]byte, 24)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	ErrorValue   string `json:"error"`
	Description  string `json:"error_description"`
}

func (f InstalledAppFlow) postToken(ctx context.Context, values url.Values) (*config.OAuthToken, error) {
	if f.Config.ClientID == "" {
		return nil, fmt.Errorf("client id is required")
	}
	tokenURL := f.Config.TokenURL
	if tokenURL == "" {
		tokenURL = DefaultTokenURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(f.Config.ClientID, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if f.Config.UserAgent != "" {
		req.Header.Set("User-Agent", f.Config.UserAgent)
	}
	resp, err := f.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var decoded tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || decoded.ErrorValue != "" {
		if decoded.Description != "" {
			return nil, fmt.Errorf("oauth token request failed: %s", decoded.Description)
		}
		if decoded.ErrorValue != "" {
			return nil, fmt.Errorf("oauth token request failed: %s", decoded.ErrorValue)
		}
		return nil, fmt.Errorf("oauth token request failed with status %d", resp.StatusCode)
	}
	if decoded.AccessToken == "" {
		return nil, fmt.Errorf("oauth token response did not include an access token")
	}
	expiresIn := decoded.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	token := &config.OAuthToken{
		AccessToken:  decoded.AccessToken,
		RefreshToken: decoded.RefreshToken,
		TokenType:    decoded.TokenType,
		ExpiresAt:    time.Now().UTC().Add(time.Duration(expiresIn) * time.Second),
		Scopes:       strings.Fields(decoded.Scope),
	}
	if token.TokenType == "" {
		token.TokenType = "bearer"
	}
	return token, nil
}

func (f InstalledAppFlow) httpClient() *http.Client {
	if f.Config.HTTPClient != nil {
		return f.Config.HTTPClient
	}
	return http.DefaultClient
}
