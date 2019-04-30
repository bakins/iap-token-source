package iap

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/jws"
)

func TestTokenSource(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	enc := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	require.NoError(t, err)

	serviceAccountKey := struct {
		Type         string `json:"type"`
		ClientEmail  string `json:"client_email"`
		PrivateKeyID string `json:"private_key_id"`
		PrivateKey   string `json:"private_key"`
		TokenURL     string `json:"token_uri"`
		ProjectID    string `json:"project_id"`
	}{
		Type:        "service_account",
		ClientEmail: "hello-world@example.com",
		PrivateKey:  string(enc),
	}

	data, err := json.Marshal(&serviceAccountKey)
	require.NoError(t, err)

	audience := "test@example.com"

	i, err := New(audience, WithServiceAccount(data), WithPostFormer(&testPortFormer{t}))
	require.NoError(t, err)

	tok, err := i.Token()
	require.NoError(t, err)
	require.Equal(t, "Bearer", tok.TokenType)
	require.Equal(t, "hello-world", tok.AccessToken)
}

type testPortFormer struct {
	t *testing.T
}

func (t *testPortFormer) PostForm(url string, values url.Values) (resp *http.Response, err error) {
	cs, err := jws.Decode(values.Get("assertion"))
	require.NoError(t.t, err)
	require.Equal(t.t, TokenURI, cs.Aud)
	require.Equal(t.t, "hello-world@example.com", cs.Iss)

	tok := struct {
		IDToken string `json:"id_token"`
	}{
		IDToken: "hello-world",
	}

	data, err := json.Marshal(&tok)
	require.NoError(t.t, err)

	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBuffer(data)),
	}, nil
}
