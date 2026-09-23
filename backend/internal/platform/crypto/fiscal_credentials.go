package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
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

// FiscalCredentialBinding is included in AES-GCM AAD so ciphertext cannot be moved across connections.
type FiscalCredentialBinding struct {
	ConnectionID uuid.UUID
	ProviderKey  string
	ScopeKey     string
}

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
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: AES-256 key must be 32 bytes", ErrFiscalCredentialInvalidKey)
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

// Encrypt seals the plaintext with a random nonce and empty binding (legacy path).
func (c *FiscalCredentialCipher) Encrypt(plaintext []byte) (FiscalCredentialEnvelope, error) {
	return c.EncryptWithBinding(plaintext, FiscalCredentialBinding{})
}

// EncryptWithBinding seals plaintext under connection-scoped AAD.
func (c *FiscalCredentialCipher) EncryptWithBinding(plaintext []byte, binding FiscalCredentialBinding) (FiscalCredentialEnvelope, error) {
	if c == nil || c.aead == nil {
		return FiscalCredentialEnvelope{}, fmt.Errorf("%w: cipher is nil", ErrFiscalCredentialEnvelope)
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
		return FiscalCredentialEnvelope{}, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := c.aead.Seal(nil, nonce, plaintext, c.additionalData(binding))
	return FiscalCredentialEnvelope{
		FormatVersion: c.formatVersion,
		KeyVersion:    c.keyVersion,
		Nonce:         nonce,
		Ciphertext:    ciphertext,
	}, nil
}

// Decrypt opens an envelope encrypted with empty binding (legacy path).
func (c *FiscalCredentialCipher) Decrypt(envelope FiscalCredentialEnvelope) ([]byte, error) {
	return c.DecryptWithBinding(envelope, FiscalCredentialBinding{})
}

// DecryptWithBinding opens an envelope under connection-scoped AAD.
func (c *FiscalCredentialCipher) DecryptWithBinding(envelope FiscalCredentialEnvelope, binding FiscalCredentialBinding) ([]byte, error) {
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
	plaintext, err := c.aead.Open(nil, envelope.Nonce, envelope.Ciphertext, c.additionalData(binding))
	if err != nil {
		return nil, fmt.Errorf("decrypt credential: %w", err)
	}
	return plaintext, nil
}

func (c *FiscalCredentialCipher) additionalData(binding FiscalCredentialBinding) []byte {
	return []byte(fmt.Sprintf(
		"fmt=%d|conn=%s|provider=%s|scope=%s|alg=%s",
		c.formatVersion,
		binding.ConnectionID.String(),
		strings.TrimSpace(binding.ProviderKey),
		strings.TrimSpace(binding.ScopeKey),
		fiscalCredentialAlgorithm,
	))
}

// FiscalCredentialKeyring holds an active encrypt key plus decrypt-only prior versions.
type FiscalCredentialKeyring struct {
	activeVersion string
	ciphers       map[string]*FiscalCredentialCipher
}

// NewFiscalCredentialKeyring builds a rotation-aware keyring. activeVersion must exist in keys.
func NewFiscalCredentialKeyring(activeVersion string, keys map[string][]byte) (*FiscalCredentialKeyring, error) {
	activeVersion = strings.TrimSpace(activeVersion)
	if activeVersion == "" {
		return nil, fmt.Errorf("%w: active key version is required", ErrFiscalCredentialInvalidKey)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: keyring is empty", ErrFiscalCredentialInvalidKey)
	}
	if _, ok := keys[activeVersion]; !ok {
		return nil, fmt.Errorf("%w: active version %q missing", ErrFiscalCredentialInvalidKey, activeVersion)
	}
	ciphers := make(map[string]*FiscalCredentialCipher, len(keys))
	for version, key := range keys {
		cipher, err := NewFiscalCredentialCipher(version, key)
		if err != nil {
			return nil, err
		}
		ciphers[version] = cipher
	}
	return &FiscalCredentialKeyring{activeVersion: activeVersion, ciphers: ciphers}, nil
}

// ActiveVersion returns the encrypting key version.
func (k *FiscalCredentialKeyring) ActiveVersion() string {
	if k == nil {
		return ""
	}
	return k.activeVersion
}

// Encrypt seals under the active key version.
func (k *FiscalCredentialKeyring) Encrypt(binding FiscalCredentialBinding, plaintext []byte) (FiscalCredentialEnvelope, error) {
	if k == nil {
		return FiscalCredentialEnvelope{}, fmt.Errorf("%w: keyring is nil", ErrFiscalCredentialEnvelope)
	}
	active := k.ciphers[k.activeVersion]
	if active == nil {
		return FiscalCredentialEnvelope{}, fmt.Errorf("%w: active cipher missing", ErrFiscalCredentialInvalidKey)
	}
	return active.EncryptWithBinding(plaintext, binding)
}

// Decrypt opens an envelope with the matching key version.
func (k *FiscalCredentialKeyring) Decrypt(binding FiscalCredentialBinding, envelope FiscalCredentialEnvelope) ([]byte, error) {
	if k == nil {
		return nil, fmt.Errorf("%w: keyring is nil", ErrFiscalCredentialEnvelope)
	}
	cipher := k.ciphers[envelope.KeyVersion]
	if cipher == nil {
		return nil, fmt.Errorf("%w: unknown key version %q", ErrFiscalCredentialVersion, envelope.KeyVersion)
	}
	return cipher.DecryptWithBinding(envelope, binding)
}

// Rotate re-encrypts an envelope under the active key without exposing plaintext to callers.
func (k *FiscalCredentialKeyring) Rotate(binding FiscalCredentialBinding, envelope FiscalCredentialEnvelope) (FiscalCredentialEnvelope, error) {
	plain, err := k.Decrypt(binding, envelope)
	if err != nil {
		return FiscalCredentialEnvelope{}, err
	}
	return k.Encrypt(binding, plain)
}

// ValidateProductionKeyring refuses default/zero keys and short material in production.
func ValidateProductionKeyring(appEnv, keyVersion string, key []byte) error {
	keyVersion = strings.TrimSpace(keyVersion)
	if keyVersion == "" {
		return fmt.Errorf("%w: key version is required", ErrFiscalCredentialInvalidKey)
	}
	if len(key) != 32 {
		return fmt.Errorf("%w: AES-256 key must be 32 bytes", ErrFiscalCredentialInvalidKey)
	}
	if !strings.EqualFold(strings.TrimSpace(appEnv), "production") {
		return nil
	}
	allZero := true
	for _, b := range key {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return fmt.Errorf("%w: production refuses all-zero credential key", ErrFiscalCredentialInvalidKey)
	}
	return nil
}

func cloneBytes(values []byte) []byte {
	if values == nil {
		return nil
	}
	return append([]byte(nil), values...)
}
