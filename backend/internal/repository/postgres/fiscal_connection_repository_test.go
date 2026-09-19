package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type FiscalConnectionRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo ports.FiscalConnectionRepository
}

func (suite *FiscalConnectionRepositoryTestSuite) SetupSuite() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(suite.T(), err)
	require.NoError(suite.T(), db.AutoMigrate(&fiscalConnectionModel{}))
	suite.db = db
	suite.repo = NewPostgresFiscalConnectionRepository(db)
}

func (suite *FiscalConnectionRepositoryTestSuite) TearDownTest() {
	suite.db.Exec("DELETE FROM fiscal_provider_connections")
}

func (suite *FiscalConnectionRepositoryTestSuite) TestSaveAndLoadDeepCopy() {
	record := &ports.FiscalConnectionRecord{
		ID:                      uuid.New(),
		ScopeKey:                "default",
		ProviderKey:             "mock",
		State:                   ports.FiscalConnectionStateAuthorizing,
		ProviderReference:       "org-1",
		GrantedScopes:           []string{"issue", "void"},
		AccessExpiresAt:         timePtr(time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)),
		CredentialCiphertext:    []byte{1, 2, 3},
		CredentialNonce:         []byte{4, 5, 6},
		CredentialKeyVersion:    "v1",
		CredentialFormatVersion: 1,
		LastVerifiedAt:          timePtr(time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)),
		ConnectedAt:             timePtr(time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC)),
		RevokedAt:               timePtr(time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC)),
		CreatedBy:               uuid.New(),
		UpdatedBy:               uuidPtr(uuid.New()),
		CreatedAt:               time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt:               time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC),
		Version:                 1,
	}
	require.NoError(suite.T(), suite.repo.Save(context.Background(), record))

	record.GrantedScopes[0] = "changed"
	record.CredentialCiphertext[0] = 9
	record.CredentialNonce[0] = 8

	got, err := suite.repo.GetByScopeProvider(context.Background(), "default", "mock")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), got)
	assert.Equal(suite.T(), []string{"issue", "void"}, got.GrantedScopes)
	assert.Equal(suite.T(), byte(1), got.CredentialCiphertext[0])
	assert.Equal(suite.T(), byte(4), got.CredentialNonce[0])
	assert.Equal(suite.T(), record.ID, got.ID)
	assert.Equal(suite.T(), record.CreatedBy, got.CreatedBy)
	assert.NotNil(suite.T(), got.UpdatedBy)
	assert.Equal(suite.T(), *record.UpdatedBy, *got.UpdatedBy)
}

func (suite *FiscalConnectionRepositoryTestSuite) TestSaveUpdatesSingleRow() {
	first := &ports.FiscalConnectionRecord{
		ID:                      uuid.New(),
		ScopeKey:                "default",
		ProviderKey:             "mock",
		State:                   ports.FiscalConnectionStateAuthorizing,
		CredentialCiphertext:    []byte{1},
		CredentialNonce:         []byte{2},
		CredentialKeyVersion:    "v1",
		CredentialFormatVersion: 1,
		CreatedBy:               uuid.New(),
		UpdatedBy:               uuidPtr(uuid.New()),
		CreatedAt:               time.Now().UTC(),
		UpdatedAt:               time.Now().UTC(),
		Version:                 1,
	}
	require.NoError(suite.T(), suite.repo.Save(context.Background(), first))

	second := first.Clone()
	second.CredentialCiphertext = []byte{9, 9}
	second.CredentialNonce = []byte{8, 8}
	second.State = ports.FiscalConnectionStateConnected
	require.NoError(suite.T(), suite.repo.Save(context.Background(), &second))

	got, err := suite.repo.GetByScopeProvider(context.Background(), "default", "mock")
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), got)
	assert.Equal(suite.T(), first.ID, got.ID)
	assert.Equal(suite.T(), ports.FiscalConnectionStateConnected, got.State)
	assert.Equal(suite.T(), int64(1), countFiscalConnectionRows(suite.T(), suite.db))
	assert.GreaterOrEqual(suite.T(), got.Version, 2)
}

func (suite *FiscalConnectionRepositoryTestSuite) TestGetByIDAndMissingRecord() {
	none, err := suite.repo.GetByID(context.Background(), uuid.New())
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), none)

	none, err = suite.repo.GetByScopeProvider(context.Background(), "missing", "mock")
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), none)
}

func TestFiscalConnectionRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(FiscalConnectionRepositoryTestSuite))
}

func timePtr(t time.Time) *time.Time { return &t }

func uuidPtr(v uuid.UUID) *uuid.UUID { return &v }

func countFiscalConnectionRows(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&fiscalConnectionModel{}).Count(&count).Error)
	return count
}
