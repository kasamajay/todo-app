package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	googleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
)

// GoogleUserInfo is the subset of Google's OpenID Connect userinfo response
// this app cares about.
type GoogleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
}

// RandomState returns a URL-safe random value suitable for an OAuth CSRF
// state parameter. It carries no claims, so unlike Mint/Verify it is not
// HMAC-signed - its own unguessability plus a short cookie TTL is enough.
func RandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GoogleAuthURL builds the URL to redirect the browser to in order to start
// Google's OAuth 2.0 Authorization Code flow.
func GoogleAuthURL(clientID, redirectURI, state string) string {
	q := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
	}
	return googleAuthURL + "?" + q.Encode()
}

// ExchangeGoogleCode exchanges an authorization code for an access token by
// calling Google's token endpoint directly over stdlib net/http - no JWT or
// JWKS parsing is needed since Google's servers do that verification.
func ExchangeGoogleCode(clientID, clientSecret, redirectURI, code string) (string, error) {
	resp, err := http.PostForm(googleTokenURL, url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tr googleTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK || tr.AccessToken == "" {
		if tr.Error != "" {
			return "", fmt.Errorf("google token exchange failed: %s", tr.Error)
		}
		return "", fmt.Errorf("google token exchange failed: status %d", resp.StatusCode)
	}
	return tr.AccessToken, nil
}

// FetchGoogleUserInfo fetches the authenticated user's profile using an
// access token obtained from ExchangeGoogleCode.
func FetchGoogleUserInfo(accessToken string) (GoogleUserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return GoogleUserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return GoogleUserInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GoogleUserInfo{}, errors.New("google userinfo request failed")
	}

	var info GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return GoogleUserInfo{}, err
	}
	if info.Sub == "" {
		return GoogleUserInfo{}, errors.New("google userinfo response missing sub")
	}
	return info, nil
}
