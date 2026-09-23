package fiscalartifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/google/uuid"
)

const (
	// DefaultMaxArtifactBytes caps streamed PDF archives (20 MiB).
	DefaultMaxArtifactBytes int64 = 20 << 20
	pdfMediaType                  = "application/pdf"
	pdfMagic                      = "%PDF"
)

// LocalStore is a development/test private filesystem FiscalArtifactStore.
type LocalStore struct {
	root     string
	maxBytes int64
}

// Option configures LocalStore.
type Option func(*LocalStore)

// WithMaxBytes overrides the default size limit.
func WithMaxBytes(n int64) Option {
	return func(s *LocalStore) {
		if n > 0 {
			s.maxBytes = n
		}
	}
}

// NewLocalStore creates a private local artifact root.
func NewLocalStore(root string, opts ...Option) (*LocalStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("fiscal artifact local root is required")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	s := &LocalStore{root: root, maxBytes: DefaultMaxArtifactBytes}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// BuildStorageKey returns the deterministic private key path.
func BuildStorageKey(environment string, documentID uuid.UUID) string {
	env := strings.Trim(strings.ToLower(strings.TrimSpace(environment)), "/")
	if env == "" {
		env = "default"
	}
	return fmt.Sprintf("fiscal/%s/%s/provider.pdf", env, documentID.String())
}

type localMeta struct {
	MediaType string `json:"mediaType"`
	ByteSize  int64  `json:"byteSize"`
	SHA256    string `json:"sha256"`
}

func (s *LocalStore) objectPath(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") || strings.Contains(key, `\`) {
		return "", ports.ErrFiscalArtifactNotFound
	}
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}

func (s *LocalStore) metaPath(objectPath string) string {
	return objectPath + ".meta.json"
}

// PutImmutable writes bytes with create-if-absent checksum matching.
func (s *LocalStore) PutImmutable(ctx context.Context, key, mediaType string, body io.Reader) (ports.StoredArtifact, error) {
	if err := ctx.Err(); err != nil {
		return ports.StoredArtifact{}, err
	}
	if strings.TrimSpace(mediaType) != pdfMediaType {
		return ports.StoredArtifact{}, ports.ErrFiscalArtifactMediaType
	}
	objectPath, err := s.objectPath(key)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	if err := os.MkdirAll(filepath.Dir(objectPath), 0o700); err != nil {
		return ports.StoredArtifact{}, err
	}

	tmp, err := os.CreateTemp(filepath.Dir(objectPath), ".fiscal-*.tmp")
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	hasher := sha256.New()
	limited := &io.LimitedReader{R: body, N: s.maxBytes + 1}
	tee := io.TeeReader(limited, hasher)

	var header [4]byte
	n, err := io.ReadFull(tee, header[:])
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return ports.StoredArtifact{}, err
	}
	if n < len(pdfMagic) || string(header[:len(pdfMagic)]) != pdfMagic {
		return ports.StoredArtifact{}, ports.ErrFiscalArtifactSignature
	}
	if _, err := tmp.Write(header[:n]); err != nil {
		return ports.StoredArtifact{}, err
	}
	written, err := io.Copy(tmp, tee)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	total := int64(n) + written
	if total > s.maxBytes || limited.N == 0 {
		return ports.StoredArtifact{}, ports.ErrFiscalArtifactTooLarge
	}
	if err := tmp.Sync(); err != nil {
		return ports.StoredArtifact{}, err
	}
	if err := tmp.Close(); err != nil {
		return ports.StoredArtifact{}, err
	}

	shaHex := hex.EncodeToString(hasher.Sum(nil))
	meta := localMeta{MediaType: pdfMediaType, ByteSize: total, SHA256: shaHex}

	if _, err := os.Stat(objectPath); err == nil {
		existing, err := s.readMeta(objectPath)
		if err != nil {
			return ports.StoredArtifact{}, err
		}
		if existing.SHA256 != shaHex || existing.ByteSize != total {
			return ports.StoredArtifact{}, ports.ErrFiscalArtifactCollision
		}
		return ports.StoredArtifact{
			Key: key, MediaType: existing.MediaType, ByteSize: existing.ByteSize, SHA256: existing.SHA256, Created: false,
		}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return ports.StoredArtifact{}, err
	}

	if err := os.Rename(tmpName, objectPath); err != nil {
		return ports.StoredArtifact{}, err
	}
	tmpName = "" // renamed; skip remove
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	if err := os.WriteFile(s.metaPath(objectPath), metaBytes, 0o600); err != nil {
		return ports.StoredArtifact{}, err
	}
	return ports.StoredArtifact{
		Key: key, MediaType: pdfMediaType, ByteSize: total, SHA256: shaHex, Created: true,
	}, nil
}

// Open returns a reader for a stored object.
func (s *LocalStore) Open(ctx context.Context, key string) (io.ReadCloser, ports.StoredArtifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, ports.StoredArtifact{}, err
	}
	objectPath, err := s.objectPath(key)
	if err != nil {
		return nil, ports.StoredArtifact{}, err
	}
	meta, err := s.readMeta(objectPath)
	if err != nil {
		return nil, ports.StoredArtifact{}, err
	}
	f, err := os.Open(objectPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ports.StoredArtifact{}, ports.ErrFiscalArtifactNotFound
		}
		return nil, ports.StoredArtifact{}, err
	}
	return f, ports.StoredArtifact{
		Key: key, MediaType: meta.MediaType, ByteSize: meta.ByteSize, SHA256: meta.SHA256,
	}, nil
}

// Stat returns metadata without opening the full stream body beyond metadata.
func (s *LocalStore) Stat(ctx context.Context, key string) (ports.StoredArtifact, error) {
	if err := ctx.Err(); err != nil {
		return ports.StoredArtifact{}, err
	}
	objectPath, err := s.objectPath(key)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	meta, err := s.readMeta(objectPath)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	return ports.StoredArtifact{
		Key: key, MediaType: meta.MediaType, ByteSize: meta.ByteSize, SHA256: meta.SHA256,
	}, nil
}

func (s *LocalStore) readMeta(objectPath string) (localMeta, error) {
	raw, err := os.ReadFile(s.metaPath(objectPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return localMeta{}, ports.ErrFiscalArtifactNotFound
		}
		return localMeta{}, err
	}
	var meta localMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return localMeta{}, err
	}
	return meta, nil
}

// VerifyPDFPrefix is a pure helper used by callers that buffer small headers.
func VerifyPDFPrefix(b []byte) error {
	if !bytes.HasPrefix(b, []byte(pdfMagic)) {
		return ports.ErrFiscalArtifactSignature
	}
	return nil
}
