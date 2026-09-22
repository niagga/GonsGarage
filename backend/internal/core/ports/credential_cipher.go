package ports

// CredentialCipher encrypts and decrypts versioned fiscal credential envelopes.
// Concrete AES-GCM implementation lives under platform/crypto; WU3 only needs the port.
type CredentialCipher interface {
	Encrypt(plaintext []byte) (ciphertext, nonce []byte, keyVersion string, formatVersion int, err error)
	Decrypt(ciphertext, nonce []byte, keyVersion string, formatVersion int) (plaintext []byte, err error)
}
