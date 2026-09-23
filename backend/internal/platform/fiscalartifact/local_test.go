package fiscalartifact_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/platform/fiscalartifact"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func samplePDF(label string) []byte {
	return []byte("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF\n%" + label + "\n")
}

func TestLocalStore_PutImmutableCreateIfAbsentMatchingChecksum(t *testing.T) {
	store, err := fiscalartifact.NewLocalStore(t.TempDir())
	require.NoError(t, err)
	ctx := context.Background()
	key := fiscalartifact.BuildStorageKey("dev", uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	pdf := samplePDF("match")

	first, err := store.PutImmutable(ctx, key, "application/pdf", bytes.NewReader(pdf))
	require.NoError(t, err)
	assert.True(t, first.Created)
	sum := sha256.Sum256(pdf)
	assert.Equal(t, hex.EncodeToString(sum[:]), first.SHA256)
	assert.Equal(t, int64(len(pdf)), first.ByteSize)
	assert.Equal(t, "application/pdf", first.MediaType)
	assert.Equal(t, key, first.Key)

	second, err := store.PutImmutable(ctx, key, "application/pdf", bytes.NewReader(pdf))
	require.NoError(t, err)
	assert.False(t, second.Created)
	assert.Equal(t, first.SHA256, second.SHA256)
	assert.Equal(t, first.ByteSize, second.ByteSize)
}

func TestLocalStore_PutImmutableRejectsChecksumCollision(t *testing.T) {
	store, err := fiscalartifact.NewLocalStore(t.TempDir())
	require.NoError(t, err)
	ctx := context.Background()
	key := fiscalartifact.BuildStorageKey("dev", uuid.New())

	_, err = store.PutImmutable(ctx, key, "application/pdf", bytes.NewReader(samplePDF("a")))
	require.NoError(t, err)

	_, err = store.PutImmutable(ctx, key, "application/pdf", bytes.NewReader(samplePDF("b")))
	require.ErrorIs(t, err, ports.ErrFiscalArtifactCollision)
}

func TestLocalStore_StreamingChecksumAndOpenRoundTrip(t *testing.T) {
	store, err := fiscalartifact.NewLocalStore(t.TempDir())
	require.NoError(t, err)
	ctx := context.Background()
	key := fiscalartifact.BuildStorageKey("test", uuid.New())
	pdf := samplePDF("stream")

	stored, err := store.PutImmutable(ctx, key, "application/pdf", bytes.NewReader(pdf))
	require.NoError(t, err)

	stat, err := store.Stat(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, stored.SHA256, stat.SHA256)

	rc, meta, err := store.Open(ctx, key)
	require.NoError(t, err)
	defer rc.Close()
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, pdf, got)
	assert.Equal(t, stored.SHA256, meta.SHA256)
	assert.Equal(t, "application/pdf", meta.MediaType)
}

func TestLocalStore_RejectsNonPDFMediaAndSignatureAndOversize(t *testing.T) {
	store, err := fiscalartifact.NewLocalStore(t.TempDir(), fiscalartifact.WithMaxBytes(64))
	require.NoError(t, err)
	ctx := context.Background()
	key := fiscalartifact.BuildStorageKey("dev", uuid.New())

	_, err = store.PutImmutable(ctx, key, "text/plain", bytes.NewReader(samplePDF("x")))
	require.ErrorIs(t, err, ports.ErrFiscalArtifactMediaType)

	_, err = store.PutImmutable(ctx, key, "application/pdf", bytes.NewReader([]byte("NOTPDF")))
	require.ErrorIs(t, err, ports.ErrFiscalArtifactSignature)

	big := append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte("A"), 128)...)
	_, err = store.PutImmutable(ctx, key, "application/pdf", bytes.NewReader(big))
	require.ErrorIs(t, err, ports.ErrFiscalArtifactTooLarge)
}

func TestBuildStorageKey_IsDeterministicPrivatePath(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	key := fiscalartifact.BuildStorageKey("prod", id)
	assert.Equal(t, "fiscal/prod/11111111-2222-3333-4444-555555555555/provider.pdf", key)
	assert.False(t, strings.Contains(key, "http"))
	assert.False(t, strings.HasPrefix(key, "/"))
}
