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
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

func b64EncodeURL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func bigIntBytes(n *big.Int, length int) []byte {
	b := n.Bytes()
	if len(b) >= length {
		return b
	}
	padded := make([]byte, length)
	copy(padded[length-len(b):], b)
	return padded
}

func publicKeyToJWK(pub crypto.PublicKey) (map[string]string, error) {
	switch key := pub.(type) {
	case *ecdsa.PublicKey:
		return ecdsaPublicKeyToJWK(key)
	case *rsa.PublicKey:
		return rsaPublicKeyToJWK(key)
	default:
		return nil, fmt.Errorf("unsupported key type: %T", pub)
	}
}

func ecdsaPublicKeyToJWK(pub *ecdsa.PublicKey) (map[string]string, error) {
	var crv, alg string
	var size int
	switch pub.Curve {
	case elliptic.P256():
		crv = "P-256"
		alg = "ES256"
		size = 32
	case elliptic.P384():
		crv = "P-384"
		alg = "ES384"
		size = 48
	default:
		return nil, fmt.Errorf("unsupported ECDSA curve: %v", pub.Curve.Params().Name)
	}

	kid, err := keyIDFromPublicKey(pub)
	if err != nil {
		return nil, err
	}
	jwk := map[string]string{
		"kty": "EC",
		"crv": crv,
		"alg": alg,
		"use": "sig",
		"kid": kid,
		"x":   b64EncodeURL(bigIntBytes(pub.X, size)),
		"y":   b64EncodeURL(bigIntBytes(pub.Y, size)),
	}
	return jwk, nil
}

func rsaPublicKeyToJWK(pub *rsa.PublicKey) (map[string]string, error) {
	e := big.NewInt(int64(pub.E))

	kid, err := keyIDFromPublicKey(pub)
	if err != nil {
		return nil, err
	}
	jwk := map[string]string{
		"kty": "RSA",
		"alg": "RS256",
		"use": "sig",
		"kid": kid,
		"n":   b64EncodeURL(bigIntBytes(pub.N, (pub.N.BitLen()+7)/8)),
		"e":   b64EncodeURL(e.Bytes()),
	}
	return jwk, nil
}

// keyIDFromPublicKey computes the key ID the same way Kubernetes does:
// base64url(sha256(PKIX DER encoding of the public key)).
func keyIDFromPublicKey(pub crypto.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("failed to serialize public key to DER format: %w", err)
	}
	hash := sha256.Sum256(der)
	return b64EncodeURL(hash[:]), nil
}

func generateJWKS(pub crypto.PublicKey) (string, error) {
	jwk, err := publicKeyToJWK(pub)
	if err != nil {
		return "", err
	}
	jwks := map[string]any{
		"keys": []any{jwk},
	}
	b, err := json.Marshal(jwks)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
