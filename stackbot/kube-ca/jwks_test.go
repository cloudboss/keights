// Copyright 2026 Joseph Wright <joseph@cloudboss.co>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateJWKS_ECDSA_P256(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	jwksStr, err := generateJWKS(key.Public())
	require.NoError(t, err)

	var jwks map[string]any
	err = json.Unmarshal([]byte(jwksStr), &jwks)
	require.NoError(t, err)

	keys, ok := jwks["keys"].([]any)
	require.True(t, ok)
	require.Len(t, keys, 1)

	jwk, ok := keys[0].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "EC", jwk["kty"])
	assert.Equal(t, "P-256", jwk["crv"])
	assert.Equal(t, "ES256", jwk["alg"])
	assert.Equal(t, "sig", jwk["use"])
	assert.NotEmpty(t, jwk["x"])
	assert.NotEmpty(t, jwk["y"])
	assert.NotEmpty(t, jwk["kid"])
}

func TestGenerateJWKS_ECDSA_P384(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)

	jwksStr, err := generateJWKS(key.Public())
	require.NoError(t, err)

	var jwks map[string]any
	err = json.Unmarshal([]byte(jwksStr), &jwks)
	require.NoError(t, err)

	keys := jwks["keys"].([]any)
	jwk := keys[0].(map[string]any)

	assert.Equal(t, "EC", jwk["kty"])
	assert.Equal(t, "P-384", jwk["crv"])
	assert.Equal(t, "ES384", jwk["alg"])
}

func TestGenerateJWKS_RSA(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)

	jwksStr, err := generateJWKS(key.Public())
	require.NoError(t, err)

	var jwks map[string]any
	err = json.Unmarshal([]byte(jwksStr), &jwks)
	require.NoError(t, err)

	keys, ok := jwks["keys"].([]any)
	require.True(t, ok)
	require.Len(t, keys, 1)

	jwk, ok := keys[0].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "RSA", jwk["kty"])
	assert.Equal(t, "RS256", jwk["alg"])
	assert.Equal(t, "sig", jwk["use"])
	assert.NotEmpty(t, jwk["n"])
	assert.NotEmpty(t, jwk["e"])
	assert.NotEmpty(t, jwk["kid"])
}

// TestGenerateJWKS_KIDMatchesKubernetes verifies that the kid matches the
// algorithm used by Kubernetes: base64url(sha256(PKIX DER public key)).
func TestGenerateJWKS_KIDMatchesKubernetes(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	jwksStr, err := generateJWKS(key.Public())
	require.NoError(t, err)

	var jwks map[string]any
	err = json.Unmarshal([]byte(jwksStr), &jwks)
	require.NoError(t, err)

	jwk := jwks["keys"].([]any)[0].(map[string]any)
	kid := jwk["kid"].(string)

	// Compute expected kid the way Kubernetes does.
	der, err := x509.MarshalPKIXPublicKey(key.Public())
	require.NoError(t, err)
	hash := sha256.Sum256(der)
	expectedKID := base64.RawURLEncoding.EncodeToString(hash[:])

	assert.Equal(t, expectedKID, kid)
}

func TestGenerateJWKS_Deterministic(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	jwks1, err := generateJWKS(key.Public())
	require.NoError(t, err)

	jwks2, err := generateJWKS(key.Public())
	require.NoError(t, err)

	assert.Equal(t, jwks1, jwks2)
}
