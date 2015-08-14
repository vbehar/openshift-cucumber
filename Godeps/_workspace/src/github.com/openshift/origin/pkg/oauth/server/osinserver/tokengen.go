package osinserver

import (
	"encoding/base64"
	"io"
	"strings"

	"crypto/rand"

	"github.com/RangelReale/osin"
)

func randomBytes(len int) []byte {
	b := make([]byte, len)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		// rand.Reader should never fail
		panic(err.Error())
	}
	return b
}
func randomToken() string {
	// 32 bytes (256 bits) = 43 base64-encoded characters
	b := randomBytes(32)
	// Use URLEncoding to ensure we don't get / characters
	s := base64.URLEncoding.EncodeToString(b)
	// Strip trailing ='s... they're ugly
	return strings.TrimRight(s, "=")
}

// TokenGen is an authorization and access token generator
type TokenGen struct {
}

// GenerateAuthorizeToken generates a random UUID code
func (TokenGen) GenerateAuthorizeToken(data *osin.AuthorizeData) (ret string, err error) {
	return randomToken(), nil
}

// GenerateAccessToken generates random UUID access and refresh tokens
func (TokenGen) GenerateAccessToken(data *osin.AccessData, generaterefresh bool) (string, string, error) {
	accesstoken := randomToken()

	refreshtoken := ""
	if generaterefresh {
		refreshtoken = randomToken()
	}
	return accesstoken, refreshtoken, nil
}
