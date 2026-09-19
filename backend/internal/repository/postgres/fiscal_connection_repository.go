package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type fiscalConnectionModel struct {
	ID                      uuid.UUID      `gorm:"type:uuid;primaryKey;column:id"`
	ScopeKey                string         `gorm:"type:varchar(80);not null;column:scope_key;index:idx_fiscal_connection_scope_provider,unique"`
	ProviderKey             string         `gorm:"type:varchar(40);not null;column:provider_key;index:idx_fiscal_connection_scope_provider,unique"`
	State                   string         `gorm:"type:varchar(24);not null;column:state"`
	ProviderReference       string         `gorm:"type:varchar(255);column:provider_organization_ref"`
	GrantedScopes           pq.StringArray `gorm:"type:text[];column:granted_scopes"`
	AccessExpiresAt         *time.Time     `gorm:"column:access_expires_at"`
	CredentialCiphertext    []byte         `gorm:"column:credential_ciphertext"`
	CredentialNonce         []byte         `gorm:"column:credential_nonce"`
	CredentialKeyVersion    string         `gorm:"type:varchar(40);column:credential_key_version"`
	CredentialFormatVersion int            `gorm:"column:credential_format_version"`
	LastVerifiedAt          *time.Time     `gorm:"column:last_verified_at"`
	ConnectedAt             *time.Time     `gorm:"column:connected_at"`
	RevokedAt               *time.Time     `gorm:"column:revoked_at"`
	CreatedBy               uuid.UUID      `gorm:"type:uuid;column:created_by"`
	UpdatedBy               *uuid.UUID     `gorm:"type:uuid;column:updated_by"`
	CreatedAt               time.Time      `gorm:"column:created_at"`
	UpdatedAt               time.Time      `gorm:"column:updated_at"`
	Version                 int            `gorm:"column:version"`
}

func (fiscalConnectionModel) TableName() string {
	return "fiscal_provider_connections"
}

type PostgresFiscalConnectionRepository struct {
	db *gorm.DB
}

var _ ports.FiscalConnectionRepository = (*PostgresFiscalConnectionRepository)(nil)

func NewPostgresFiscalConnectionRepository(db *gorm.DB) *PostgresFiscalConnectionRepository {
	return &PostgresFiscalConnectionRepository{db: db}
}

func (r *PostgresFiscalConnectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*ports.FiscalConnectionRecord, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("fiscal connection repository is nil")
	}
	var model fiscalConnectionModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get fiscal connection by id: %w", err)
	}
	return fiscalModelToRecord(&model), nil
}

func (r *PostgresFiscalConnectionRepository) GetByScopeProvider(ctx context.Context, scopeKey, providerKey string) (*ports.FiscalConnectionRecord, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("fiscal connection repository is nil")
	}
	scopeKey = strings.TrimSpace(scopeKey)
	providerKey = strings.TrimSpace(providerKey)
	if scopeKey == "" || providerKey == "" {
		return nil, fmt.Errorf("scope and provider keys are required")
	}
	var model fiscalConnectionModel
	if err := r.db.WithContext(ctx).Where("scope_key = ? AND provider_key = ?", scopeKey, providerKey).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get fiscal connection by scope/provider: %w", err)
	}
	return fiscalModelToRecord(&model), nil
}

func (r *PostgresFiscalConnectionRepository) Save(ctx context.Context, record *ports.FiscalConnectionRecord) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("fiscal connection repository is nil")
	}
	if record == nil {
		return fmt.Errorf("fiscal connection record is nil")
	}
	record.ScopeKey = strings.TrimSpace(record.ScopeKey)
	record.ProviderKey = strings.TrimSpace(record.ProviderKey)
	if record.ScopeKey == "" || record.ProviderKey == "" {
		return fmt.Errorf("scope and provider keys are required")
	}
	now := time.Now().UTC()
	model := recordToFiscalModel(record)
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
	}
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
	}
	if model.Version <= 0 {
		model.Version = 1
	}
	if model.UpdatedAt.IsZero() || model.UpdatedAt.Before(model.CreatedAt) {
		model.UpdatedAt = now
	}
	existing, err := r.GetByScopeProvider(ctx, model.ScopeKey, model.ProviderKey)
	if err != nil {
		return err
	}
	if existing != nil {
		model.ID = existing.ID
		model.CreatedAt = existing.CreatedAt
		if model.Version <= existing.Version {
			model.Version = existing.Version + 1
		}
		if model.CreatedBy == uuid.Nil {
			model.CreatedBy = existing.CreatedBy
		}
		if model.UpdatedBy == nil {
			model.UpdatedBy = uuidPtrClone(existing.UpdatedBy)
		}
	}
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return fmt.Errorf("save fiscal connection: %w", err)
	}
	return nil
}

func fiscalModelToRecord(model *fiscalConnectionModel) *ports.FiscalConnectionRecord {
	if model == nil {
		return nil
	}
	return &ports.FiscalConnectionRecord{
		ID:                      model.ID,
		ScopeKey:                model.ScopeKey,
		ProviderKey:             model.ProviderKey,
		State:                   ports.FiscalConnectionState(model.State),
		ProviderReference:       model.ProviderReference,
		GrantedScopes:           append([]string(nil), []string(model.GrantedScopes)...),
		AccessExpiresAt:         timePtrClone(model.AccessExpiresAt),
		CredentialCiphertext:    append([]byte(nil), model.CredentialCiphertext...),
		CredentialNonce:         append([]byte(nil), model.CredentialNonce...),
		CredentialKeyVersion:    model.CredentialKeyVersion,
		CredentialFormatVersion: model.CredentialFormatVersion,
		LastVerifiedAt:          timePtrClone(model.LastVerifiedAt),
		ConnectedAt:             timePtrClone(model.ConnectedAt),
		RevokedAt:               timePtrClone(model.RevokedAt),
		CreatedBy:               model.CreatedBy,
		UpdatedBy:               uuidPtrClone(model.UpdatedBy),
		CreatedAt:               model.CreatedAt,
		UpdatedAt:               model.UpdatedAt,
		Version:                 model.Version,
	}
}

func recordToFiscalModel(record *ports.FiscalConnectionRecord) fiscalConnectionModel {
	if record == nil {
		return fiscalConnectionModel{}
	}
	return fiscalConnectionModel{
		ID:                      record.ID,
		ScopeKey:                strings.TrimSpace(record.ScopeKey),
		ProviderKey:             strings.TrimSpace(record.ProviderKey),
		State:                   string(record.State),
		ProviderReference:       record.ProviderReference,
		GrantedScopes:           append(pq.StringArray(nil), record.GrantedScopes...),
		AccessExpiresAt:         timePtrClone(record.AccessExpiresAt),
		CredentialCiphertext:    append([]byte(nil), record.CredentialCiphertext...),
		CredentialNonce:         append([]byte(nil), record.CredentialNonce...),
		CredentialKeyVersion:    record.CredentialKeyVersion,
		CredentialFormatVersion: record.CredentialFormatVersion,
		LastVerifiedAt:          timePtrClone(record.LastVerifiedAt),
		ConnectedAt:             timePtrClone(record.ConnectedAt),
		RevokedAt:               timePtrClone(record.RevokedAt),
		CreatedBy:               record.CreatedBy,
		UpdatedBy:               uuidPtrClone(record.UpdatedBy),
		CreatedAt:               record.CreatedAt,
		UpdatedAt:               record.UpdatedAt,
		Version:                 record.Version,
	}
}

func timePtrClone(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func uuidPtrClone(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
