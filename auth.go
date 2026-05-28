package gokalshi

import (
	"crypto"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Credentials holds the RSA key pair used for API authentication.
type Credentials struct {
	KeyID      string
	PrivateKey *rsa.PrivateKey
	// Rand is the random source for PSS signing. Defaults to crypto/rand.Reader.
	Rand io.Reader
}

// LoadCredentials reads an RSA private key from a PEM file and returns Credentials.
func LoadCredentials(keyID, privateKeyPath string) (*Credentials, error) {
	pemData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, &AuthError{Op: "load_credentials", Err: fmt.Errorf("failed to read private key file %s: %w", privateKeyPath, err)}
	}

	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, &AuthError{Op: "load_credentials", Err: fmt.Errorf("failed to decode PEM block from %s", privateKeyPath)}
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Fallback: try PKCS1 format
		pkcs1Key, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if pkcs1Err != nil {
			return nil, &AuthError{Op: "load_credentials", Err: fmt.Errorf("failed to parse private key from %s: %w (also tried PKCS1: %v)", privateKeyPath, err, pkcs1Err)}
		}
		key = pkcs1Key
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, &AuthError{Op: "load_credentials", Err: fmt.Errorf("private key in %s is not RSA", privateKeyPath)}
	}

	return &Credentials{
		KeyID:      keyID,
		PrivateKey: rsaKey,
		Rand:       cryptorand.Reader,
	}, nil
}

// RequestHeaders generates the authentication headers required by the Kalshi API.
func (c *Credentials) RequestHeaders(method, path string) (map[string]string, error) {
	timestampMs := strconv.FormatInt(time.Now().UnixMilli(), 10)

	// Strip query parameters from path for signing
	pathOnly := strings.SplitN(path, "?", 2)[0]
	msgString := timestampMs + method + pathOnly

	signature, err := c.SignPSSText(msgString)
	if err != nil {
		return nil, &AuthError{Op: "request_headers", Err: fmt.Errorf("failed to sign request: %w", err)}
	}

	return map[string]string{
		"Content-Type":            "application/json",
		"KALSHI-ACCESS-KEY":       c.KeyID,
		"KALSHI-ACCESS-SIGNATURE": signature,
		"KALSHI-ACCESS-TIMESTAMP": timestampMs,
	}, nil
}

// SignPSSText signs the given text using RSA-PSS with SHA-256.
func (c *Credentials) SignPSSText(text string) (string, error) {
	message := []byte(text)
	hash := sha256.Sum256(message)

	signature, err := rsa.SignPSS(
		c.randReader(),
		c.PrivateKey,
		crypto.SHA256,
		hash[:],
		&rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthEqualsHash,
		},
	)
	if err != nil {
		return "", &AuthError{Op: "sign", Err: fmt.Errorf("RSA-PSS sign failed: %w", err)}
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// LoadCredentialsFromPEM parses an RSA private key from a PEM-encoded string
// and returns Credentials. This is useful when the key is stored in an
// environment variable rather than a file (e.g. CI/CD secrets).
func LoadCredentialsFromPEM(keyID, pemString string) (*Credentials, error) {
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return nil, &AuthError{Op: "load_credentials_pem", Err: fmt.Errorf("failed to decode PEM block from string")}
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Fallback: try PKCS1 format
		pkcs1Key, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if pkcs1Err != nil {
			return nil, &AuthError{Op: "load_credentials_pem", Err: fmt.Errorf("failed to parse private key from PEM string: %w (also tried PKCS1: %v)", err, pkcs1Err)}
		}
		key = pkcs1Key
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, &AuthError{Op: "load_credentials_pem", Err: fmt.Errorf("PEM string does not contain an RSA private key")}
	}

	return &Credentials{
		KeyID:      keyID,
		PrivateKey: rsaKey,
		Rand:       cryptorand.Reader,
	}, nil
}

// NewCredentials creates Credentials from a pre-loaded RSA private key.
// Use this when you have already parsed the key yourself.
func NewCredentials(keyID string, key *rsa.PrivateKey) *Credentials {
	return &Credentials{
		KeyID:      keyID,
		PrivateKey: key,
		Rand:       cryptorand.Reader,
	}
}

func (c *Credentials) randReader() io.Reader {
	if c.Rand != nil {
		return c.Rand
	}
	return cryptorand.Reader
}
