package crypto

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiscalCredentialCipherRoundTrip(t *testing.T) {
	t.Parallel()

	cipher, err := NewFiscalCredentialCipher("v1", bytes.Repeat([]byte{0x11}, 32))
	require.NoError(t, err)

	plaintext := []byte("super-secret-credential-bytes")
	envelope, err := cipher.Encrypt(plaintext)
	require.NoError(t, err)

	assert.Equal(t, fiscalCredentialFormatVersionV1, envelope.FormatVersion)
	assert.Equal(t, "v1", envelope.KeyVersion)
	assert.Len(t, envelope.Nonce, 12)
	assert.NotEqual(t, plaintext, envelope.Ciphertext)
	assert.NotContains(t, envelope.Redacted(), string(plaintext))
	assert.Contains(t, envelope.Redacted(), "format=1")
	assert.Contains(t, envelope.Redacted(), "key=v1")

	decrypted, err := cipher.Decrypt(envelope)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestFiscalCredentialCipherRejectsTampering(t *testing.T) {
	t.Parallel()

	cipher, err := NewFiscalCredentialCipher("v1", bytes.Repeat([]byte{0x22}, 32))
	require.NoError(t, err)

	envelope, err := cipher.Encrypt([]byte("secret"))
	require.NoError(t, err)

	mutated := envelope.Clone()
	mutated.KeyVersion = "v2"
	_, err = cipher.Decrypt(mutated)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFiscalCredentialVersion)

	mutated = envelope.Clone()
	mutated.FormatVersion = 2
	_, err = cipher.Decrypt(mutated)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFiscalCredentialVersion)

	mutated = envelope.Clone()
	mutated.Nonce = mutated.Nonce[:len(mutated.Nonce)-1]
	_, err = cipher.Decrypt(mutated)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFiscalCredentialEnvelope)
}

func TestFiscalCredentialCipherRejectsBadKeyVersion(t *testing.T) {
	t.Parallel()

	_, err := NewFiscalCredentialCipher("", bytes.Repeat([]byte{0x33}, 32))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFiscalCredentialInvalidKey)
	assert.True(t, errors.Is(err, ErrFiscalCredentialInvalidKey))
}
