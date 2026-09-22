package mock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// OperationRecord is the persisted first normalized mock result.
type OperationRecord struct {
	OperationKey      string          `json:"operationKey"`
	Scenario          string          `json:"scenario"`
	CanonicalSHA256   string          `json:"canonicalSha256"`
	ProviderReference string          `json:"providerReference"`
	Result            json.RawMessage `json:"result"`
	PDFSHA256         string          `json:"pdfSha256,omitempty"`
	PDFBytes          []byte          `json:"pdfBytes,omitempty"`
	Voided            bool            `json:"voided"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

// IssueResultPayload is stored inside OperationRecord.Result for issue outcomes.
type IssueResultPayload struct {
	DocumentID     uuid.UUID `json:"documentId"`
	ArtifactID     uuid.UUID `json:"artifactId"`
	ProviderNumber string    `json:"providerNumber"`
	DocumentKind   string    `json:"documentKind"`
	Ambiguous      bool      `json:"ambiguous"`
	IssuedAtUnix   int64     `json:"issuedAtUnix,omitempty"`
	ObservedAtUnix int64     `json:"observedAtUnix"`
	CorrelationKey string    `json:"correlationKey"`
	ProviderKey    string    `json:"providerKey"`
	ConnectionID   uuid.UUID `json:"connectionId"`
}

// OperationStore persists the first mock result for an operation key.
type OperationStore interface {
	Get(ctx context.Context, operationKey string) (*OperationRecord, error)
	GetByProviderReference(ctx context.Context, providerReference string) (*OperationRecord, error)
	PutIfAbsent(ctx context.Context, record *OperationRecord) (*OperationRecord, error)
	MarkVoided(ctx context.Context, operationKey string) error
}

// MemoryOperationStore is an in-process store for unit tests.
type MemoryOperationStore struct {
	mu    sync.Mutex
	byKey map[string]*OperationRecord
	byRef map[string]string
}

// NewMemoryOperationStore creates an empty in-memory store.
func NewMemoryOperationStore() *MemoryOperationStore {
	return &MemoryOperationStore{
		byKey: make(map[string]*OperationRecord),
		byRef: make(map[string]string),
	}
}

func (s *MemoryOperationStore) Get(_ context.Context, operationKey string) (*OperationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.byKey[operationKey]
	if !ok {
		return nil, nil
	}
	return cloneRecord(rec), nil
}

func (s *MemoryOperationStore) GetByProviderReference(_ context.Context, providerReference string) (*OperationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.byRef[providerReference]
	if !ok {
		return nil, nil
	}
	return cloneRecord(s.byKey[key]), nil
}

func (s *MemoryOperationStore) PutIfAbsent(_ context.Context, record *OperationRecord) (*OperationRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.byKey[record.OperationKey]; ok {
		return cloneRecord(existing), nil
	}
	stored := cloneRecord(record)
	now := time.Now().UTC()
	if stored.CreatedAt.IsZero() {
		stored.CreatedAt = now
	}
	stored.UpdatedAt = now
	s.byKey[record.OperationKey] = stored
	s.byRef[record.ProviderReference] = record.OperationKey
	return cloneRecord(stored), nil
}

func (s *MemoryOperationStore) MarkVoided(_ context.Context, operationKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.byKey[operationKey]
	if !ok {
		return fmt.Errorf("mock operation %q not found", operationKey)
	}
	rec.Voided = true
	rec.UpdatedAt = time.Now().UTC()
	return nil
}

func cloneRecord(in *OperationRecord) *OperationRecord {
	if in == nil {
		return nil
	}
	out := *in
	if in.Result != nil {
		out.Result = append(json.RawMessage(nil), in.Result...)
	}
	if in.PDFBytes != nil {
		out.PDFBytes = append([]byte(nil), in.PDFBytes...)
	}
	return &out
}

func canonicalSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func mockProviderReference(operationKey, canonical string) string {
	sum := sha256.Sum256([]byte(operationKey + "|" + canonical))
	return "MOCK-" + hex.EncodeToString(sum[:16])
}

func mockProviderNumber(seed string) string {
	sum := sha256.Sum256([]byte("number|" + seed))
	return "MOCK-NUM-" + hex.EncodeToString(sum[:8])
}

func mockArtifactID(operationKey, canonical string, documentID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("mock-artifact|"+operationKey+"|"+canonical+"|"+documentID.String()))
}
