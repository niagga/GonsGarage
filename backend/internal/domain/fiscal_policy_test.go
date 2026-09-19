package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyVersionResolve_ApprovalGateAndImmutability(t *testing.T) {
	t.Parallel()

	policyVersion := testApprovedPolicyVersion(t, RoundingScopePerLine)
	resolver := StaticFiscalPolicyResolver{
		policyVersion.PolicyKey: map[int]PolicyVersion{
			policyVersion.Version: policyVersion,
		},
	}

	policy, err := resolver.Resolve(policyVersion.PolicyKey, policyVersion.Version)
	require.NoError(t, err)
	assert.Equal(t, policyVersion.PolicyKey, policy.PolicyKey)
	assert.Equal(t, policyVersion.Config.Currency, policy.Currency)
	assert.True(t, policy.CanIssue(DocumentKindFT))
	assert.True(t, policy.CanIssue(DocumentKindFR))
	assert.True(t, policy.CanVoid(DocumentKindFT))
	assert.True(t, policy.CanVoid(DocumentKindFR))
	assert.Equal(t, policyVersion.ConfigSHA256, policy.ConfigSHA256)

	policyVersion.Config.AllowedKinds[DocumentKindFT] = false
	policyVersion.Config.TaxTreatments["VAT23"] = TaxTreatmentRule{Code: "VAT23", Rate: MustParseDecimal("99")}
	assert.True(t, policy.CanIssue(DocumentKindFT))
	rule, ok := policy.TaxTreatment("VAT23")
	require.True(t, ok)
	assert.Equal(t, "23", rule.Rate.String())

	draft := policyVersion
	draft.Status = PolicyStatusDraft
	_, err = draft.Resolve()
	assert.ErrorIs(t, err, ErrPolicyNotApproved)

	emptyDigest := policyVersion
	emptyDigest.ConfigSHA256 = ""
	_, err = emptyDigest.Resolve()
	assert.ErrorIs(t, err, ErrPolicyConfigInvalid)

	_, err = resolver.Resolve("missing-policy", 1)
	assert.ErrorIs(t, err, ErrPolicyNotFound)
}

func TestCalculate_TaxableAndExemptLines_CanonicalOrderingForFTAndFR(t *testing.T) {
	t.Parallel()

	policy := testApprovedPolicy(t, RoundingScopePerLine)
	baseInput := CalculationInput{
		Currency: policy.Currency,
		Customer: CustomerIdentity{
			LegalName:     "Garage Customer",
			TaxIdentifier: "PT123456789",
			CountryCode:   "PT",
		},
		Lines: []LineInput{
			{
				Position:         2,
				Description:      "Exempt line",
				UnitCode:         "H",
				Quantity:         MustParseDecimal("1"),
				UnitPrice:        MustParseDecimal("5"),
				Discount:         DiscountInput{Kind: DiscountKindAmount, Value: MustParseDecimal("1")},
				TaxTreatmentCode: "EXEMPT",
				TaxRate:          MustParseDecimal("0"),
				ExemptionCode:    "A1",
				ExemptionReason:  "Approved exemption",
			},
			{
				Position:         1,
				Description:      "Taxable line",
				UnitCode:         "EA",
				Quantity:         MustParseDecimal("2"),
				UnitPrice:        MustParseDecimal("10"),
				Discount:         DiscountInput{Kind: DiscountKindNone, Value: ZeroDecimal()},
				TaxTreatmentCode: "VAT23",
				TaxRate:          MustParseDecimal("23"),
			},
		},
	}

	for _, kind := range []DocumentKind{DocumentKindFT, DocumentKindFR} {
		kind := kind
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()

			inputA := baseInput
			inputA.Kind = kind
			inputA.Lines = []LineInput{baseInput.Lines[0], baseInput.Lines[1]}

			inputB := baseInput
			inputB.Kind = kind
			inputB.Lines = []LineInput{baseInput.Lines[1], baseInput.Lines[0]}

			resultA, err := Calculate(policy, inputA)
			require.NoError(t, err)
			resultB, err := Calculate(policy, inputB)
			require.NoError(t, err)

			assert.Equal(t, resultA.CanonicalSHA256, resultB.CanonicalSHA256)
			assert.Equal(t, resultA.CanonicalBytes, resultB.CanonicalBytes)
			assert.Len(t, resultA.Lines, 2)
			assert.Equal(t, 1, resultA.Lines[0].Position)
			assert.Equal(t, 2, resultA.Lines[1].Position)
			assert.Equal(t, "25", resultA.Totals.GrossTotal.String())
			assert.Equal(t, "1", resultA.Totals.DiscountTotal.String())
			assert.Equal(t, "24", resultA.Totals.NetTotal.String())
			assert.Equal(t, "4.6", resultA.Totals.TaxTotal.String())
			assert.Equal(t, "0", resultA.Totals.RoundingAdjustment.String())
			assert.Equal(t, "28.6", resultA.Totals.PayableTotal.String())
			assert.Equal(t, "4.6", resultA.Lines[0].TaxAmount.String())
			assert.Equal(t, "1", resultA.Lines[1].DiscountAmount.String())
		})
	}
}

func TestCalculate_DocumentRoundingAdjustmentAndDeclaredMismatch(t *testing.T) {
	t.Parallel()

	policy := testApprovedPolicy(t, RoundingScopeDocument)
	input := CalculationInput{
		Kind:     DocumentKindFT,
		Currency: policy.Currency,
		Customer: CustomerIdentity{
			LegalName:     "Garage Customer",
			TaxIdentifier: "PT123456789",
			CountryCode:   "PT",
		},
		Lines: []LineInput{
			{
				Position:         1,
				Description:      "Line one",
				UnitCode:         "EA",
				Quantity:         MustParseDecimal("1"),
				UnitPrice:        MustParseDecimal("0.333"),
				Discount:         DiscountInput{Kind: DiscountKindNone, Value: ZeroDecimal()},
				TaxTreatmentCode: "EXEMPT",
				TaxRate:          MustParseDecimal("0"),
				ExemptionCode:    "A1",
				ExemptionReason:  "Approved exemption",
			},
			{
				Position:         2,
				Description:      "Line two",
				UnitCode:         "EA",
				Quantity:         MustParseDecimal("1"),
				UnitPrice:        MustParseDecimal("0.333"),
				Discount:         DiscountInput{Kind: DiscountKindNone, Value: ZeroDecimal()},
				TaxTreatmentCode: "EXEMPT",
				TaxRate:          MustParseDecimal("0"),
				ExemptionCode:    "A1",
				ExemptionReason:  "Approved exemption",
			},
			{
				Position:         3,
				Description:      "Line three",
				UnitCode:         "EA",
				Quantity:         MustParseDecimal("1"),
				UnitPrice:        MustParseDecimal("0.333"),
				Discount:         DiscountInput{Kind: DiscountKindNone, Value: ZeroDecimal()},
				TaxTreatmentCode: "EXEMPT",
				TaxRate:          MustParseDecimal("0"),
				ExemptionCode:    "A1",
				ExemptionReason:  "Approved exemption",
			},
		},
	}

	result, err := Calculate(policy, input)
	require.NoError(t, err)
	assert.Equal(t, "1", result.Totals.PayableTotal.String())
	assert.Equal(t, "0.01", result.Totals.RoundingAdjustment.String())
	assert.Equal(t, "0.99", result.Totals.GrossTotal.String())
	assert.Equal(t, "0.99", result.Totals.NetTotal.String())
	assert.Equal(t, "0", result.Totals.TaxTotal.String())

	mismatch := input
	mismatch.DeclaredTotals = &DeclaredTotals{
		PayableTotal: decimalPtr(MustParseDecimal("0")),
	}
	_, err = Calculate(policy, mismatch)
	assert.ErrorIs(t, err, ErrDeclaredTotalsMismatch)
}

func testApprovedPolicy(t *testing.T, scope RoundingScope) ArithmeticPolicy {
	t.Helper()
	pv := testApprovedPolicyVersion(t, scope)
	policy, err := pv.Resolve()
	require.NoError(t, err)
	return policy
}

func testApprovedPolicyVersion(t *testing.T, scope RoundingScope) PolicyVersion {
	t.Helper()

	config := PolicyConfig{
		Currency:              "EUR",
		CurrencyScale:         2,
		MaximumQuantityScale:  3,
		MaximumUnitPriceScale: 3,
		MaximumTaxRateScale:   2,
		RoundingMode:          RoundingModeHalfUp,
		RoundingScope:         scope,
		AdjustmentRange: AdjustmentRange{
			Minimum: MustParseDecimal("-0.05"),
			Maximum: MustParseDecimal("0.05"),
		},
		AllowedKinds: map[DocumentKind]bool{
			DocumentKindFT: true,
			DocumentKindFR: true,
		},
		VoidAllowedKinds: map[DocumentKind]bool{
			DocumentKindFT: true,
			DocumentKindFR: true,
		},
		TaxTreatments: map[string]TaxTreatmentRule{
			"VAT23": {
				Rate: MustParseDecimal("23"),
				AllowedKinds: map[DocumentKind]bool{
					DocumentKindFT: true,
					DocumentKindFR: true,
				},
			},
			"EXEMPT": {
				Rate:                    MustParseDecimal("0"),
				Exempt:                  true,
				ExemptionCodeRequired:   true,
				ExemptionReasonRequired: true,
				AllowedKinds: map[DocumentKind]bool{
					DocumentKindFT: true,
					DocumentKindFR: true,
				},
			},
		},
		CustomerRequirements: map[DocumentKind]CustomerIdentityRequirement{
			DocumentKindFT: {
				LegalName:     true,
				TaxIdentifier: true,
				CountryCode:   true,
			},
			DocumentKindFR: {
				LegalName:   true,
				CountryCode: true,
			},
		},
		DiscountKinds: map[DiscountKind]bool{
			DiscountKindAmount:  true,
			DiscountKindPercent: true,
		},
	}

	_, digest, err := config.CanonicalBytes()
	require.NoError(t, err)
	approvedAt := time.Unix(1, 0).UTC()
	return PolicyVersion{
		PolicyKey:      "sales",
		Version:        1,
		SchemaVersion:  1,
		Classification: PolicyClassificationLegal,
		Status:         PolicyStatusApproved,
		Config:         config,
		ConfigSHA256:   digest,
		ApprovedAt:     &approvedAt,
	}
}

func decimalPtr(value Decimal) *Decimal {
	return &value
}
