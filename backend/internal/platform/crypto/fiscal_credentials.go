package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	fiscalCredentialFormatVersionV1 = 1
	fiscalCredentialAlgorithm       = "AES-GCM"
)

var (
	// ErrFiscalCredentialInvalidKey reports a bad key or key version.
	ErrFiscalCredentialInvalidKey = errors.New("invalid fiscal credential key")
	// ErrFiscalCredentialEnvelope reports malformed envelope data.
	ErrFiscalCredentialEnvelope = errors.New("invalid fiscal credential envelope")
	// ErrFiscalCredentialVersion reports a version mismatch during decrypt.
	ErrFiscalCredentialVersion = errors.New("fiscal credential version mismatch")
)

// FiscalCredentialCipher encrypts and decrypts raw credential bytes.
type FiscalCredentialCipher struct {
	formatVersion int
	keyVersion    string
	aead          cipher.AEAD
}

// FiscalCredentialEnvelope holds the encrypted credential bytes.
type FiscalCredentialEnvelope struct {
	FormatVersion int
	KeyVersion    string
	Nonce         []byte
	Ciphertext    []byte
}

// Clone returns a deep copy of the envelope.
func (e FiscalCredentialEnvelope) Clone() FiscalCredentialEnvelope {
	return FiscalCredentialEnvelope{
		FormatVersion: e.FormatVersion,
		KeyVersion:    e.KeyVersion,
		Nonce:         cloneBytes(e.Nonce),
		Ciphertext:    cloneBytes(e.Ciphertext),
	}
}

// Metadata returns redaction-friendly envelope details.
func (e FiscalCredentialEnvelope) Metadata() FiscalCredentialMetadata {
	return FiscalCredentialMetadata{
		FormatVersion:  e.FormatVersion,
		KeyVersion:     e.KeyVersion,
		Algorithm:      fiscalCredentialAlgorithm,
		NonceSize:      len(e.Nonce),
		CiphertextSize: len(e.Ciphertext),
	}
}

// Redacted returns a safe summary for logs.
func (e FiscalCredentialEnvelope) Redacted() string {
	return e.Metadata().Redacted()
}

// FiscalCredentialMetadata is the log-safe envelope summary.
type FiscalCredentialMetadata struct {
	FormatVersion  int
	KeyVersion     string
	Algorithm      string
	NonceSize      int
	CiphertextSize int
}

// Redacted returns a log-safe string without raw bytes.
func (m FiscalCredentialMetadata) Redacted() string {
	return fmt.Sprintf("format=%d key=%s algorithm=%s nonce=%d ciphertext=%d", m.FormatVersion, m.KeyVersion, m.Algorithm, m.NonceSize, m.CiphertextSize)
}

// NewFiscalCredentialCipher creates a versioned AES-GCM helper.
func NewFiscalCredentialCipher(keyVersion string, key []byte) (*FiscalCredentialCipher, error) {
	keyVersion = strings.TrimSpace(keyVersion)
	if keyVersion == "" {
		return nil, fmt.Errorf("%w: key version is required", ErrFiscalCredentialInvalidKey)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFiscalCredentialInvalidKey, err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFiscalCredentialInvalidKey, err)
	}
	return &FiscalCredentialCipher{
		formatVersion: fiscalCredentialFormatVersionV1,
		keyVersion:    keyVersion,
		aead:          aead,
	}, nil
}

// Encrypt seals the plaintext with a random nonce.
func (c *FiscalCredentialCipher) Encrypt(plaintext []byte) (FiscalCredentialEnvelope, error) {
	if c == nil || c.aead == nil {
		return FiscalCredentialEnvelope{}, fmt.Errorf("%w: cipher is nil", ErrFiscalCredentialEnvelope)
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
		return FiscalCredentialEnvelope{}, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := c.aead.Seal(nil, nonce, plaintext, c.additionalData())
	return FiscalCredentialEnvelope{
		FormatVersion: c.formatVersion,
		KeyVersion:    c.keyVersion,
		Nonce:         nonce,
		Ciphertext:    ciphertext,
	}, nil
}

// Decrypt opens an envelope and returns the original credential bytes.
func (c *FiscalCredentialCipher) Decrypt(envelope FiscalCredentialEnvelope) ([]byte, error) {
	if c == nil || c.aead == nil {
		return nil, fmt.Errorf("%w: cipher is nil", ErrFiscalCredentialEnvelope)
	}
	if envelope.FormatVersion != c.formatVersion {
		return nil, fmt.Errorf("%w: format version %d", ErrFiscalCredentialVersion, envelope.FormatVersion)
	}
	if envelope.KeyVersion != c.keyVersion {
		return nil, fmt.Errorf("%w: key version %q", ErrFiscalCredentialVersion, envelope.KeyVersion)
	}
	if len(envelope.Nonce) != c.aead.NonceSize() {
		return nil, fmt.Errorf("%w: nonce length %d", ErrFiscalCredentialEnvelope, len(envelope.Nonce))
	}
	plaintext, err := c.aead.Open(nil, envelope.Nonce, envelope.Ciphertext, c.additionalData())
	if err != nil {
		return nil, fmt.Errorf("decrypt credential: %w", err)
	}
	return plaintext, nil
}

func (c *FiscalCredentialCipher) additionalData() []byte {
	return []byte(fmt.Sprintf("fmt=%d|key=%s|alg=%s", c.formatVersion, c.keyVersion, fiscalCredentialAlgorithm))
}

func cloneBytes(values []byte) []byte {
	if values == nil {
		return nil
	}
	return append([]byte(nil), values...)
}
