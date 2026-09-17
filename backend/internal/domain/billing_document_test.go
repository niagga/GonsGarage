package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBillingDocumentKind_IsValid(t *testing.T) {
	t.Parallel()
	assert.False(t, BillingDocumentKind("client_invoice").IsValid())
	assert.True(t, BillingDocumentKindPayroll.IsValid())
	assert.True(t, BillingDocumentKindIRS.IsValid())
	assert.True(t, BillingDocumentKindOther.IsValid())
	assert.False(t, BillingDocumentKind("").IsValid())
	assert.False(t, BillingDocumentKind("unknown").IsValid())
}

func TestBillingDocument_Validate_MinFields(t *testing.T) {
	t.Parallel()
	b := &BillingDocument{Kind: BillingDocumentKindPayroll, Title: "Q1", Amount: 100}
	assert.NoError(t, b.Validate())
	b2 := &BillingDocument{Kind: BillingDocumentKind("x"), Title: "t", Amount: 1}
	assert.Error(t, b2.Validate())
	b3 := &BillingDocument{Kind: BillingDocumentKindPayroll, Title: "", Amount: 1}
	assert.Error(t, b3.Validate())
	b4 := &BillingDocument{Kind: BillingDocumentKind("client_invoice"), Title: "Client invoice", Amount: 10}
	assert.Error(t, b4.Validate())
}
