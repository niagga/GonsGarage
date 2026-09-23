package crypto

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiscalCredentialCipherAADBindingRejectsWrongConnection(t *testing.T) {
	t.Parallel()

	cipher, err := NewFiscalCredentialCipher("v1", bytes.Repeat([]byte{0x44}, 32))
	require.NoError(t, err)

	connA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	connB := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	binding := FiscalCredentialBinding{
		ConnectionID: connA,
		ProviderKey:  "cloudware",
		ScopeKey:     "default",
	}

	envelope, err := cipher.EncryptWithBinding([]byte("access-and-refresh-token"), binding)
	require.NoError(t, err)

	ok, err := cipher.DecryptWithBinding(envelope, binding)
	require.NoError(t, err)
	assert.Equal(t, []byte("access-and-refresh-token"), ok)

	_, err = cipher.DecryptWithBinding(envelope, FiscalCredentialBinding{
		ConnectionID: connB,
		ProviderKey:  "cloudware",
		ScopeKey:     "default",
	})
	require.Error(t, err)

	_, err = cipher.DecryptWithBinding(envelope, FiscalCredentialBinding{
		ConnectionID: connA,
		ProviderKey:  "mock",
		ScopeKey:     "default",
	})
	require.Error(t, err)

	_, err = cipher.DecryptWithBinding(envelope, FiscalCredentialBinding{
		ConnectionID: connA,
		ProviderKey:  "cloudware",
		ScopeKey:     "other",
	})
	require.Error(t, err)
}

func TestFiscalCredentialKeyringRotationPreservesPlaintext(t *testing.T) {
	t.Parallel()

	oldKey := bytes.Repeat([]byte{0x55}, 32)
	newKey := bytes.Repeat([]byte{0x66}, 32)
	keyring, err := NewFiscalCredentialKeyring("v2", map[string][]byte{
		"v1": oldKey,
		"v2": newKey,
	})
	require.NoError(t, err)

	binding := FiscalCredentialBinding{
		ConnectionID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		ProviderKey:  "cloudware",
		ScopeKey:     "workshop-1",
	}

	// Encrypt under old key version by temporarily using a v1-only keyring.
	legacy, err := NewFiscalCredentialKeyring("v1", map[string][]byte{"v1": oldKey})
	require.NoError(t, err)
	oldEnv, err := legacy.Encrypt(binding, []byte("rotate-me"))
	require.NoError(t, err)
	assert.Equal(t, "v1", oldEnv.KeyVersion)

	rotated, err := keyring.Rotate(binding, oldEnv)
	require.NoError(t, err)
	assert.Equal(t, "v2", rotated.KeyVersion)
	assert.NotEqual(t, oldEnv.Ciphertext, rotated.Ciphertext)

	plain, err := keyring.Decrypt(binding, rotated)
	require.NoError(t, err)
	assert.Equal(t, []byte("rotate-me"), plain)

	// Old key remains decrypt-only for unread envelopes.
	stillOld, err := keyring.Decrypt(binding, oldEnv)
	require.NoError(t, err)
	assert.Equal(t, []byte("rotate-me"), stillOld)
}

func TestValidateProductionKeyringRejectsDefaultAndShortKeys(t *testing.T) {
	t.Parallel()

	err := ValidateProductionKeyring("production", "v1", bytes.Repeat([]byte{0x00}, 32))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFiscalCredentialInvalidKey)

	err = ValidateProductionKeyring("production", "v1", []byte("too-short"))
	require.Error(t, err)

	err = ValidateProductionKeyring("production", "", bytes.Repeat([]byte{0x77}, 32))
	require.Error(t, err)

	err = ValidateProductionKeyring("production", "v1", bytes.Repeat([]byte{0x77}, 32))
	require.NoError(t, err)

	// Non-production may accept temporary keys for local tests.
	err = ValidateProductionKeyring("development", "v1", bytes.Repeat([]byte{0x00}, 32))
	require.NoError(t, err)
}

func TestFiscalCredentialEnvelopeRedactionOmitsSecrets(t *testing.T) {
	t.Parallel()

	cipher, err := NewFiscalCredentialCipher("v1", bytes.Repeat([]byte{0x88}, 32))
	require.NoError(t, err)
	env, err := cipher.EncryptWithBinding([]byte("super-secret-token-value"), FiscalCredentialBinding{
		ConnectionID: uuid.New(),
		ProviderKey:  "cloudware",
		ScopeKey:     "default",
	})
	require.NoError(t, err)

	redacted := env.Redacted()
	assert.NotContains(t, redacted, "super-secret-token-value")
	assert.NotContains(t, redacted, string(env.Ciphertext))
	assert.Contains(t, redacted, "key=v1")
}
