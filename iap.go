// Package IAP provides an oauth2 token source for authenticating with
// Google Identity Aware Proxy.
package iap

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"golang.org/x/xerrors"
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
	tokenSource oauth2.TokenSource
}

// Options is passed to New for setting creation options
type Option func(*IAP) error

// New creates an IAP token source. If filename is empty, then attempt to read
// from environment varible, then wellknown file, then from compute metadata
func New(ctx context.Context, audience string, filename string) (*IAP, error) {
	s, err := getTokenSource(ctx, filename, audience)
	if err != nil {
		return nil, err
	}
	return &IAP{
		audience:    audience,
		tokenSource: s,
	}, nil
}

// Token returns a token to be used for authentication.
func (i *IAP) Token() (*oauth2.Token, error) {
	return i.tokenSource.Token()
}

const (
	envVar = "GOOGLE_APPLICATION_CREDENTIALS"
)

func getTokenSource(ctx context.Context, filename string, audience string) (oauth2.TokenSource, error) {
	if filename == "" {
		if f := os.Getenv(envVar); f != "" {
			filename = f
		}
	}

	if filename == "" {
		f := wellKnownFile()
		if _, err := os.Stat(f); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else {
			filename = f
		}
	}

	if filename != "" {
		cfg, err := readCredentialsFile(filename)
		if err != nil {
			return nil, err
		}
		cfg.UseIDToken = true
		cfg.PrivateClaims = map[string]interface{}{
			"target_audience": audience,
		}
		return cfg.TokenSource(ctx), nil
	}

	if metadata.OnGCE() {
		return newMetadataTokenSource(audience), nil
	}

	return nil, errors.New("unable to determine credentials source")
}

func wellKnownFile() string {
	const f = "application_default_credentials.json"
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "gcloud", f)
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "gcloud", f)
}

func readCredentialsFile(filename string) (*jwt.Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	c, err := google.JWTConfigFromJSON(data)
	if err != nil {
		return nil, xerrors.Errorf("failed to read service account from %s %w", filename, err)
	}
	return c, nil
}

func newMetadataTokenSource(audience string) oauth2.TokenSource {
	m := metadataTokenSource{
		audience: audience,
	}
	return oauth2.ReuseTokenSource(nil, &m)
}

type metadataTokenSource struct {
	audience string
}

// see https://cloud.google.com/run/docs/authenticating/service-to-service
func (m *metadataTokenSource) Token() (*oauth2.Token, error) {
	data, err := metadata.Get("instance/service-accounts/default/identity?audience=" + m.audience)
	if err != nil {
		return nil, xerrors.Errorf("failed to get token from metadata service: %w", err)
	}

	return &oauth2.Token{
		AccessToken: data,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Minute * 30),
	}, nil
}
