package ports

import (
	"context"
	"io"
)

// FiscalArtifactStore is the immutable private binary store contract.
// Storage backends are implemented in later work units; WU3 only needs the port.
type FiscalArtifactStore interface {
	PutIfAbsent(ctx context.Context, key string, contentType string, r io.Reader, size int64) (sha256Hex string, created bool, err error)
	Open(ctx context.Context, key string) (rc io.ReadCloser, contentType string, size int64, sha256Hex string, err error)
}
