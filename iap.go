// Package IAP provides an oauth2 token source for authenticating with
// Google Identity Aware Proxy.
package iap

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jws"
	"golang.org/x/oauth2/jwt"
)

const (
	// TokenURI is the base uri of google oauth API
	TokenURI = "https://www.googleapis.com/oauth2/v4/token"
)

// PostFormer issues a POST to the specified URL, with data's keys and values URL-encoded as the request body.
// See https://golang.org/pkg/net/http/#Client.PostForm
type PostFormer interface {
	PostForm(url string, data url.Values) (resp *http.Response, err error)
}

// IAP is an Oauth2 token source for using a Google service account
// to access services protected by Identity Aware Proxy
type IAP struct {
	audience    string
	jwt         *jwt.Config
	postFormer  PostFormer
	tokenSource oauth2.TokenSource
}

// Options is passed to New for setting creation options
type Option func(*IAP) error

// New creates an IAP token source.
func New(audience string, opts ...Option) (*IAP, error) {
	i := &IAP{
		audience:   audience,
		postFormer: &http.Client{},
	}
	for _, o := range opts {
		if err := o(i); err != nil {
			return nil, errors.Wrap(err, "option failed")
		}
	}

	if i.jwt == nil {
		filename := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		if filename == "" {
			return nil, errors.New("no ServiceAccount option set and GOOGLE_APPLICATION_CREDENTIALS is not set")
		}
		if err := WithServiceAccountFile(filename)(i); err != nil {
			return nil, err
		}
	}

	key, err := parseKey(i.jwt.PrivateKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse private key")
	}

	ts := &tokenSource{
		iap: i,
		key: key,
	}

	tok, err := ts.Token()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get token")
	}

	i.tokenSource = oauth2.ReuseTokenSource(tok, ts)
	return i, nil
}

// Token returns a token to be used for authentication.
func (i *IAP) Token() (*oauth2.Token, error) {
	return i.tokenSource.Token()
}

// WithPostFormer sets the PostFormer to use when requesting the token.
func WithPostFormer(p PostFormer) Option {
	return func(i *IAP) error {
		i.postFormer = p
		return nil
	}
}

// WithServiceAccountFile sets the Google Service Account Key file to use for requesting a token.
// By default, the file specified in the environment variable GOOGLE_APPLICATION_CREDENTIALS is used.
func WithServiceAccountFile(filename string) Option {
	return func(i *IAP) error {
		json, err := ioutil.ReadFile(filename)
		if err != nil {
			return errors.Wrapf(err, "failed to read %s", filename)
		}
		if err := WithServiceAccount(json)(i); err != nil {
			return errors.Wrapf(err, "failed to create JWT config from %s", filename)
		}
		return nil
	}
}

// WithServiceAccount sets the  Google Service Account Key to use for requesting a token.
// By default, the contents of the file specified in the environment variable GOOGLE_APPLICATION_CREDENTIALS is used.
func WithServiceAccount(json []byte) Option {
	return func(i *IAP) error {
		j, err := google.JWTConfigFromJSON(json, i.audience)
		if err != nil {
			return errors.Wrap(err, "failed to create JWT config")
		}
		i.jwt = j
		return nil
	}
}

type tokenSource struct {
	key *rsa.PrivateKey
	iap *IAP
}

func (s *tokenSource) Token() (*oauth2.Token, error) {
	// based on https://github.com/b4b4r07/iap_curl

	iat := time.Now()
	exp := iat.Add(time.Hour)
	jwt := &jws.ClaimSet{
		Iss: s.iap.jwt.Email,
		Aud: TokenURI,
		Iat: iat.Unix(),
		Exp: exp.Unix(),
		PrivateClaims: map[string]interface{}{
			"target_audience": s.iap.audience,
		},
	}

	jwsHeader := &jws.Header{
		Algorithm: "RS256",
		Typ:       "JWT",
	}

	msg, err := jws.Encode(jwsHeader, jwt, s.key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode request")
	}

	v := url.Values{}
	v.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	v.Set("assertion", msg)

	resp, err := s.iap.postFormer.PostForm(TokenURI, v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to POST token request")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var t struct {
		TokenType string `json:"token_type"`
		IDToken   string `json:"id_token"`
		ExpiresIn int64  `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &t); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal request body")
	}

	tokenType := t.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	var expiry = exp
	if t.ExpiresIn != 0 {
		expiry = iat.Add(time.Duration(t.ExpiresIn) * time.Second)
	}

	o := &oauth2.Token{
		AccessToken: t.IDToken,
		TokenType:   tokenType,
		Expiry:      expiry,
	}
	return o, nil

}

// based on  https://github.com/golang/oauth2/blob/9f3314589c9a9136388751d9adae6b0ed400978a/internal/oauth2.go
// and https://github.com/b4b4r07/iap_curl/blob/96d6908f23531ce339be0c7f67d462800a80a3fa/iap.go
func parseKey(key []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("invalid private key data")
	}
	key = block.Bytes

	parsedKey, err := x509.ParsePKCS8PrivateKey(key)
	if err != nil {
		parsedKey, err = x509.ParsePKCS1PrivateKey(key)
		if err != nil {
			return nil, errors.Wrap(err, "private key should be a PEM or plain PKCS1 or PKCS8")
		}
	}
	parsed, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is invalid")
	}
	return parsed, nil
}
