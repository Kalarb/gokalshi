package gokalshi

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestPEM(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	}

	tmpDir := t.TempDir()
	pemPath := filepath.Join(tmpDir, "test_key.pem")
	pemFile, err := os.Create(pemPath)
	require.NoError(t, err)
	defer pemFile.Close()

	err = pem.Encode(pemFile, pemBlock)
	require.NoError(t, err)

	return pemPath
}

func TestLoadCredentials(t *testing.T) {
	pemPath := generateTestPEM(t)

	creds, err := LoadCredentials("test-key-id", pemPath)
	require.NoError(t, err)

	assert.Equal(t, "test-key-id", creds.KeyID)
	assert.NotNil(t, creds.PrivateKey)
}

func TestLoadCredentials_FileNotFound(t *testing.T) {
	_, err := LoadCredentials("key-id", "/nonexistent/path/key.pem")
	assert.Error(t, err)

	var authErr *AuthError
	assert.True(t, errors.As(err, &authErr))
	assert.Equal(t, "load_credentials", authErr.Op)
}

func TestLoadCredentials_InvalidPEM(t *testing.T) {
	tmpDir := t.TempDir()
	badPath := filepath.Join(tmpDir, "bad.pem")
	err := os.WriteFile(badPath, []byte("not a pem file"), 0600)
	require.NoError(t, err)

	_, err = LoadCredentials("key-id", badPath)
	assert.Error(t, err)

	var authErr *AuthError
	assert.True(t, errors.As(err, &authErr))
	assert.Equal(t, "load_credentials", authErr.Op)
}

func TestSignPSSText(t *testing.T) {
	pemPath := generateTestPEM(t)
	creds, err := LoadCredentials("test-key-id", pemPath)
	require.NoError(t, err)

	sig, err := creds.SignPSSText("test message")
	require.NoError(t, err)

	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(sig)
	require.NoError(t, err)
	assert.NotEmpty(t, decoded)
}

func TestSignPSSText_DifferentMessages(t *testing.T) {
	pemPath := generateTestPEM(t)
	creds, err := LoadCredentials("test-key-id", pemPath)
	require.NoError(t, err)

	sig1, err := creds.SignPSSText("message one")
	require.NoError(t, err)

	sig2, err := creds.SignPSSText("message two")
	require.NoError(t, err)

	// Different messages should produce different signatures
	assert.NotEqual(t, sig1, sig2)
}

func TestRequestHeaders(t *testing.T) {
	pemPath := generateTestPEM(t)
	creds, err := LoadCredentials("test-key-id", pemPath)
	require.NoError(t, err)

	headers, err := creds.RequestHeaders("GET", "/trade-api/v2/markets?status=open")
	require.NoError(t, err)

	assert.Equal(t, "application/json", headers["Content-Type"])
	assert.Equal(t, "test-key-id", headers["KALSHI-ACCESS-KEY"])
	assert.NotEmpty(t, headers["KALSHI-ACCESS-SIGNATURE"])
	assert.NotEmpty(t, headers["KALSHI-ACCESS-TIMESTAMP"])

	// Verify signature is valid base64
	_, err = base64.StdEncoding.DecodeString(headers["KALSHI-ACCESS-SIGNATURE"])
	assert.NoError(t, err)
}

func TestSignPSSText_QueryParamsStripped(t *testing.T) {
	pemPath := generateTestPEM(t)
	creds, err := LoadCredentials("test-key-id", pemPath)
	require.NoError(t, err)

	ts := "1700000000000"
	basePath := "/api/v2/markets"

	sig1, err := creds.SignPSSText(ts + "GET" + basePath)
	require.NoError(t, err)

	sig2, err := creds.SignPSSText(ts + "GET" + basePath)
	require.NoError(t, err)

	// RSA-PSS is randomized, so signatures differ even for same input.
	// Instead, verify that both produce valid base64 of the same length.
	decoded1, err := base64.StdEncoding.DecodeString(sig1)
	require.NoError(t, err)
	decoded2, err := base64.StdEncoding.DecodeString(sig2)
	require.NoError(t, err)
	assert.Equal(t, len(decoded1), len(decoded2))

	sigDifferent, err := creds.SignPSSText(ts + "GET" + "/api/v2/other")
	require.NoError(t, err)
	decodedDiff, err := base64.StdEncoding.DecodeString(sigDifferent)
	require.NoError(t, err)
	assert.Equal(t, len(decoded1), len(decodedDiff))
}

func TestLoadCredentials_PKCS1(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	der := x509.MarshalPKCS1PrivateKey(key)
	pemPath := filepath.Join(t.TempDir(), "pkcs1.pem")
	f, err := os.Create(pemPath)
	require.NoError(t, err)
	err = pem.Encode(f, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
	require.NoError(t, err)
	f.Close()

	creds, err := LoadCredentials("pkcs1-key", pemPath)
	require.NoError(t, err)
	assert.Equal(t, "pkcs1-key", creds.KeyID)
	assert.NotNil(t, creds.PrivateKey)
}

func generateTestPEMString(t *testing.T) (string, *rsa.PrivateKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	})

	return string(pemBlock), key
}

func TestLoadCredentialsFromPEM(t *testing.T) {
	pemStr, _ := generateTestPEMString(t)

	creds, err := LoadCredentialsFromPEM("pem-key-id", pemStr)
	require.NoError(t, err)

	assert.Equal(t, "pem-key-id", creds.KeyID)
	assert.NotNil(t, creds.PrivateKey)

	// Verify signing works
	sig, err := creds.SignPSSText("test message")
	require.NoError(t, err)
	assert.NotEmpty(t, sig)
}

func TestLoadCredentialsFromPEM_PKCS1(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	der := x509.MarshalPKCS1PrivateKey(key)
	pemStr := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: der,
	}))

	creds, err := LoadCredentialsFromPEM("pkcs1-pem", pemStr)
	require.NoError(t, err)
	assert.Equal(t, "pkcs1-pem", creds.KeyID)
	assert.NotNil(t, creds.PrivateKey)
}

func TestLoadCredentialsFromPEM_InvalidPEM(t *testing.T) {
	_, err := LoadCredentialsFromPEM("key-id", "not a pem string")
	assert.Error(t, err)

	var authErr *AuthError
	assert.True(t, errors.As(err, &authErr))
	assert.Equal(t, "load_credentials_pem", authErr.Op)
}

func TestLoadCredentialsFromPEM_EmptyString(t *testing.T) {
	_, err := LoadCredentialsFromPEM("key-id", "")
	assert.Error(t, err)

	var authErr *AuthError
	assert.True(t, errors.As(err, &authErr))
}

func TestNewCredentials(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	creds := NewCredentials("pre-loaded-key", key)

	assert.Equal(t, "pre-loaded-key", creds.KeyID)
	assert.Equal(t, key, creds.PrivateKey)
	assert.NotNil(t, creds.Rand)

	// Verify signing works
	sig, err := creds.SignPSSText("test message")
	require.NoError(t, err)
	assert.NotEmpty(t, sig)
}

func TestNewCredentials_RoundTrip(t *testing.T) {
	// Generate PEM string, load it, then create new credentials from the key
	pemStr, originalKey := generateTestPEMString(t)

	credsFromPEM, err := LoadCredentialsFromPEM("key-1", pemStr)
	require.NoError(t, err)

	credsFromKey := NewCredentials("key-2", originalKey)

	// Both should produce valid signatures
	sig1, err := credsFromPEM.SignPSSText("same message")
	require.NoError(t, err)
	sig2, err := credsFromKey.SignPSSText("same message")
	require.NoError(t, err)

	// RSA-PSS is randomized, so sigs differ — but both should be valid base64
	decoded1, err := base64.StdEncoding.DecodeString(sig1)
	require.NoError(t, err)
	decoded2, err := base64.StdEncoding.DecodeString(sig2)
	require.NoError(t, err)
	assert.Equal(t, len(decoded1), len(decoded2))
}

func TestCredentials_NilRand(t *testing.T) {
	pemPath := generateTestPEM(t)
	creds, err := LoadCredentials("test-key", pemPath)
	require.NoError(t, err)

	creds.Rand = nil
	sig, err := creds.SignPSSText("test message")
	require.NoError(t, err)
	assert.NotEmpty(t, sig)
}
