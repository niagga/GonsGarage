package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

type SupplierRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo ports.SupplierRepository
}

func (suite *SupplierRepositoryTestSuite) SetupSuite() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(suite.T(), err)
	require.NoError(suite.T(), db.AutoMigrate(&domain.Supplier{}, &domain.ReceivedInvoice{}))
	suite.db = db
	suite.repo = NewPostgresSupplierRepository(db)
}

func (suite *SupplierRepositoryTestSuite) TearDownTest() {
	suite.db.Exec("DELETE FROM received_invoices")
	suite.db.Exec("DELETE FROM suppliers")
}

func (suite *SupplierRepositoryTestSuite) TestCreateAndGetByID() {
	id := uuid.New()
	s := &domain.Supplier{
		ID: id, Name: "ACME", ContactEmail: "a@ac.me", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(suite.T(), suite.repo.Create(context.Background(), s))
	got, err := suite.repo.GetByID(context.Background(), id)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), got)
	assert.Equal(suite.T(), "ACME", got.Name)
	assert.Equal(suite.T(), "a@ac.me", got.ContactEmail)
}

func (suite *SupplierRepositoryTestSuite) TestGetByID_NotFound() {
	_, err := suite.repo.GetByID(context.Background(), uuid.New())
	require.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, domain.ErrSupplierNotFound)
}

func (suite *SupplierRepositoryTestSuite) TestUpdate_NotFound() {
	s := &domain.Supplier{ID: uuid.New(), Name: "X", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	err := suite.repo.Update(context.Background(), s)
	require.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, domain.ErrSupplierNotFound)
}

func (suite *SupplierRepositoryTestSuite) TestDelete_SoftThenInvisible() {
	id := uuid.New()
	s := &domain.Supplier{ID: id, Name: "DelCo", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(suite.T(), suite.repo.Create(context.Background(), s))
	require.NoError(suite.T(), suite.repo.Delete(context.Background(), id))
	_, err := suite.repo.GetByID(context.Background(), id)
	require.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, domain.ErrSupplierNotFound)
}

func (suite *SupplierRepositoryTestSuite) TestDelete_NullsReceivedInvoiceSupplierID() {
	id := uuid.New()
	s := &domain.Supplier{ID: id, Name: "DelCo", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(suite.T(), suite.repo.Create(context.Background(), s))

	invID := uuid.New()
	inv := &domain.ReceivedInvoice{
		ID:          invID,
		SupplierID:  &id,
		VendorName:  "Paper Co",
		Category:    "supplies",
		Amount:      120.5,
		InvoiceDate: time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(suite.T(), suite.db.Create(inv).Error)

	require.NoError(suite.T(), suite.repo.Delete(context.Background(), id))

	_, err := suite.repo.GetByID(context.Background(), id)
	require.Error(suite.T(), err)
	assert.ErrorIs(suite.T(), err, domain.ErrSupplierNotFound)

	var got domain.ReceivedInvoice
	require.NoError(suite.T(), suite.db.Where("id = ?", invID).First(&got).Error)
	assert.Nil(suite.T(), got.SupplierID)
	assert.Nil(suite.T(), got.DeletedAt)
}

func (suite *SupplierRepositoryTestSuite) TestList_TotalAndPagination() {
	for i := 0; i < 3; i++ {
		id := uuid.New()
		s := &domain.Supplier{ID: id, Name: "S", IsActive: true, CreatedAt: time.Now().Add(time.Duration(i) * time.Second), UpdatedAt: time.Now()}
		require.NoError(suite.T(), suite.repo.Create(context.Background(), s))
	}
	list, total, err := suite.repo.List(context.Background(), 2, 0)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), total)
	assert.Len(suite.T(), list, 2)
	list2, total2, err := suite.repo.List(context.Background(), 2, 2)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(3), total2)
	assert.Len(suite.T(), list2, 1)
}

func TestSupplierRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(SupplierRepositoryTestSuite))
}
