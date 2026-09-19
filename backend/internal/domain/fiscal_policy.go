package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	// ErrPolicyNotFound is returned when a requested policy version cannot be located.
	ErrPolicyNotFound = errors.New("fiscal policy version not found")
	// ErrPolicyNotApproved is returned when a policy version has not passed approval.
	ErrPolicyNotApproved = errors.New("fiscal policy version is not approved")
	// ErrPolicyConfigInvalid is returned when a policy configuration is incomplete or inconsistent.
	ErrPolicyConfigInvalid = errors.New("fiscal policy configuration is invalid")
	// ErrDocumentKindNotAllowed is returned when a document kind is blocked by policy.
	ErrDocumentKindNotAllowed = errors.New("document kind is not allowed by policy")
	// ErrCurrencyMismatch is returned when the request currency does not match the policy currency.
	ErrCurrencyMismatch = errors.New("currency does not match policy currency")
	// ErrCustomerIdentityMissing is returned when a required customer identity field is absent.
	ErrCustomerIdentityMissing = errors.New("customer identity requirement is not satisfied")
	// ErrTaxTreatmentMissing is returned when a tax treatment code is absent or unknown.
	ErrTaxTreatmentMissing = errors.New("tax treatment is missing or unknown")
	// ErrDiscountKindNotAllowed is returned when a discount kind is blocked by policy.
	ErrDiscountKindNotAllowed = errors.New("discount kind is not allowed by policy")
	// ErrDeclaredTotalsMismatch is returned when declared totals differ from calculated totals.
	ErrDeclaredTotalsMismatch = errors.New("declared totals do not match calculated totals")
	// ErrCalculationLineInvalid is returned when a line fails arithmetic or policy validation.
	ErrCalculationLineInvalid = errors.New("calculation line is invalid")
	// ErrDuplicateLinePosition is returned when multiple lines share the same position.
	ErrDuplicateLinePosition = errors.New("duplicate line position")
	// ErrRoundingAdjustmentOutOfRange is returned when the computed document adjustment exceeds policy limits.
	ErrRoundingAdjustmentOutOfRange = errors.New("rounding adjustment is outside policy range")
)

// DocumentKind identifies the fiscal document shape.
type DocumentKind string

const (
	DocumentKindFT DocumentKind = "FT"
	DocumentKindFR DocumentKind = "FR"
)

func (k DocumentKind) IsValid() bool {
	switch k {
	case DocumentKindFT, DocumentKindFR:
		return true
	default:
		return false
	}
}

// DiscountKind identifies how a line discount is expressed.
type DiscountKind string

const (
	DiscountKindNone    DiscountKind = "none"
	DiscountKindAmount  DiscountKind = "amount"
	DiscountKindPercent DiscountKind = "percent"
)

func (k DiscountKind) IsValid() bool {
	switch k {
	case DiscountKindNone, DiscountKindAmount, DiscountKindPercent:
		return true
	default:
		return false
	}
}

// RoundingMode identifies how midpoint values are handled.
type RoundingMode string

const (
	RoundingModeHalfUp   RoundingMode = "half_up"
	RoundingModeHalfEven RoundingMode = "half_even"
	RoundingModeDown     RoundingMode = "down"
	RoundingModeUp       RoundingMode = "up"
	RoundingModeFloor    RoundingMode = "floor"
	RoundingModeCeiling  RoundingMode = "ceiling"
)

func (m RoundingMode) IsValid() bool {
	switch m {
	case RoundingModeHalfUp, RoundingModeHalfEven, RoundingModeDown, RoundingModeUp, RoundingModeFloor, RoundingModeCeiling:
		return true
	default:
		return false
	}
}

// RoundingScope identifies where rounding occurs.
type RoundingScope string

const (
	RoundingScopePerLine  RoundingScope = "per_line"
	RoundingScopeDocument RoundingScope = "document"
)

func (s RoundingScope) IsValid() bool {
	switch s {
	case RoundingScopePerLine, RoundingScopeDocument:
		return true
	default:
		return false
	}
}

// PolicyStatus identifies the lifecycle gate of a policy row.
type PolicyStatus string

const (
	PolicyStatusDraft    PolicyStatus = "draft"
	PolicyStatusApproved PolicyStatus = "approved"
	PolicyStatusRetired  PolicyStatus = "retired"
)

func (s PolicyStatus) IsValid() bool {
	switch s {
	case PolicyStatusDraft, PolicyStatusApproved, PolicyStatusRetired:
		return true
	default:
		return false
	}
}

// PolicyClassification identifies whether a policy is legal or mock.
type PolicyClassification string

const (
	PolicyClassificationLegal PolicyClassification = "legal"
	PolicyClassificationMock  PolicyClassification = "mock"
)

func (c PolicyClassification) IsValid() bool {
	switch c {
	case PolicyClassificationLegal, PolicyClassificationMock:
		return true
	default:
		return false
	}
}

// AdjustmentRange constrains the permitted rounding adjustment.
type AdjustmentRange struct {
	Minimum Decimal `json:"minimum"`
	Maximum Decimal `json:"maximum"`
}

func (r AdjustmentRange) Validate() error {
	if r.Minimum.Cmp(r.Maximum) > 0 {
		return fmt.Errorf("%w: minimum %s is greater than maximum %s", ErrPolicyConfigInvalid, r.Minimum, r.Maximum)
	}
	return nil
}

func (r AdjustmentRange) Contains(value Decimal) bool {
	return value.Cmp(r.Minimum) >= 0 && value.Cmp(r.Maximum) <= 0
}

// TaxTreatmentRule describes a tax treatment code and its behavior.
type TaxTreatmentRule struct {
	Code                     string                  `json:"code"`
	AllowedKinds             map[DocumentKind]bool   `json:"allowedKinds,omitempty"`
	Rate                     Decimal                 `json:"rate"`
	Exempt                   bool                    `json:"exempt"`
	ExemptionCodeRequired    bool                    `json:"exemptionCodeRequired"`
	ExemptionReasonRequired  bool                    `json:"exemptionReasonRequired"`
}

func (r TaxTreatmentRule) Validate(maxRateScale int32) error {
	if r.Code == "" {
		return fmt.Errorf("%w: missing tax treatment code", ErrPolicyConfigInvalid)
	}
	if maxRateScale < 0 || maxRateScale > 18 {
		return fmt.Errorf("%w: invalid tax-rate scale limit %d", ErrPolicyConfigInvalid, maxRateScale)
	}
	if r.Rate.Scale() > maxRateScale {
		return fmt.Errorf("%w: tax treatment %s exceeds maximum rate scale %d", ErrPolicyConfigInvalid, r.Code, maxRateScale)
	}
	if r.Exempt {
		if !r.Rate.IsZero() {
			return fmt.Errorf("%w: exempt tax treatment %s must have zero rate", ErrPolicyConfigInvalid, r.Code)
		}
	}
	for kind, allowed := range r.AllowedKinds {
		if !kind.IsValid() {
			return fmt.Errorf("%w: invalid allowed kind %q on tax treatment %s", ErrPolicyConfigInvalid, kind, r.Code)
		}
		_ = allowed
	}
	return nil
}

// CustomerIdentityRequirement specifies which identity fields are mandatory.
type CustomerIdentityRequirement struct {
	LegalName    bool `json:"legalName"`
	TaxIdentifier bool `json:"taxIdentifier"`
	CountryCode  bool `json:"countryCode"`
}

// PolicyConfig captures the schema-versioned policy JSON.
type PolicyConfig struct {
	Currency               string                             `json:"currency"`
	CurrencyScale          int32                              `json:"currencyScale"`
	MaximumQuantityScale   int32                              `json:"maximumQuantityScale"`
	MaximumUnitPriceScale  int32                              `json:"maximumUnitPriceScale"`
	MaximumTaxRateScale    int32                              `json:"maximumTaxRateScale"`
	RoundingMode           RoundingMode                       `json:"roundingMode"`
	RoundingScope          RoundingScope                      `json:"roundingScope"`
	AdjustmentRange        AdjustmentRange                    `json:"adjustmentRange"`
	AllowedKinds           map[DocumentKind]bool              `json:"allowedKinds,omitempty"`
	VoidAllowedKinds       map[DocumentKind]bool              `json:"voidAllowedKinds,omitempty"`
	TaxTreatments          map[string]TaxTreatmentRule        `json:"taxTreatments,omitempty"`
	CustomerRequirements   map[DocumentKind]CustomerIdentityRequirement `json:"customerRequirements,omitempty"`
	DiscountKinds          map[DiscountKind]bool              `json:"discountKinds,omitempty"`
}

// CanonicalBytes returns the canonical JSON payload for the policy configuration.
func (c PolicyConfig) CanonicalBytes() ([]byte, string, error) {
	if err := c.Validate(); err != nil {
		return nil, "", err
	}

	payload := policyConfigCanonical{
		Currency:              c.Currency,
		CurrencyScale:         c.CurrencyScale,
		MaximumQuantityScale:  c.MaximumQuantityScale,
		MaximumUnitPriceScale: c.MaximumUnitPriceScale,
		MaximumTaxRateScale:   c.MaximumTaxRateScale,
		RoundingMode:          c.RoundingMode,
		RoundingScope:         c.RoundingScope,
		AdjustmentRange:       c.AdjustmentRange,
		AllowedKinds:          c.AllowedKinds,
		VoidAllowedKinds:      c.VoidAllowedKinds,
		TaxTreatments:         c.TaxTreatments,
		CustomerRequirements:  c.CustomerRequirements,
		DiscountKinds:         c.DiscountKinds,
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(bytes)
	return bytes, hex.EncodeToString(sum[:]), nil
}

// Validate checks the policy configuration for the invariants needed by the calculator.
func (c PolicyConfig) Validate() error {
	if !isUpperASCIIAlpha3(c.Currency) {
		return fmt.Errorf("%w: currency must be an uppercase ISO-like 3-letter code", ErrPolicyConfigInvalid)
	}
	if !c.RoundingMode.IsValid() {
		return fmt.Errorf("%w: invalid rounding mode %q", ErrPolicyConfigInvalid, c.RoundingMode)
	}
	if !c.RoundingScope.IsValid() {
		return fmt.Errorf("%w: invalid rounding scope %q", ErrPolicyConfigInvalid, c.RoundingScope)
	}
	for name, scale := range map[string]int32{
		"currencyScale":         c.CurrencyScale,
		"maximumQuantityScale":   c.MaximumQuantityScale,
		"maximumUnitPriceScale":  c.MaximumUnitPriceScale,
		"maximumTaxRateScale":    c.MaximumTaxRateScale,
	} {
		if scale < 0 || scale > 18 {
			return fmt.Errorf("%w: %s out of range: %d", ErrPolicyConfigInvalid, name, scale)
		}
	}
	if len(c.TaxTreatments) == 0 {
		return fmt.Errorf("%w: at least one tax treatment is required", ErrPolicyConfigInvalid)
	}
	if err := c.AdjustmentRange.Validate(); err != nil {
		return err
	}
	for kind, requirement := range c.CustomerRequirements {
		if !kind.IsValid() {
			return fmt.Errorf("%w: invalid customer requirement kind %q", ErrPolicyConfigInvalid, kind)
		}
		_ = requirement
	}
	allowedCount := 0
	for kind, allowed := range c.AllowedKinds {
		if !kind.IsValid() {
			return fmt.Errorf("%w: invalid allowed document kind %q", ErrPolicyConfigInvalid, kind)
		}
		if !allowed {
			continue
		}
		allowedCount++
		if _, ok := c.CustomerRequirements[kind]; !ok {
			return fmt.Errorf("%w: missing customer requirement for allowed kind %s", ErrPolicyConfigInvalid, kind)
		}
	}
	if allowedCount == 0 {
		return fmt.Errorf("%w: at least one document kind must be allowed", ErrPolicyConfigInvalid)
	}
	for kind, allowed := range c.VoidAllowedKinds {
		if !kind.IsValid() {
			return fmt.Errorf("%w: invalid void-allowed document kind %q", ErrPolicyConfigInvalid, kind)
		}
		_ = allowed
	}
	for code, rule := range c.TaxTreatments {
		if code == "" {
			return fmt.Errorf("%w: empty tax treatment code", ErrPolicyConfigInvalid)
		}
		rule.Code = code
		if err := rule.Validate(c.MaximumTaxRateScale); err != nil {
			return err
		}
	}
	for kind, allowed := range c.DiscountKinds {
		if !kind.IsValid() {
			return fmt.Errorf("%w: invalid discount kind %q", ErrPolicyConfigInvalid, kind)
		}
		_ = allowed
	}
	return nil
}

// PolicyVersion captures the stored policy row metadata.
type PolicyVersion struct {
	PolicyKey      string            `json:"policyKey"`
	Version        int               `json:"version"`
	SchemaVersion  int               `json:"schemaVersion"`
	Classification PolicyClassification `json:"classification"`
	Status         PolicyStatus      `json:"status"`
	Config         PolicyConfig      `json:"config"`
	ConfigSHA256   string            `json:"configSha256"`
	ApprovedAt     *time.Time        `json:"approvedAt,omitempty"`
}

// Resolve converts an approved policy row into an immutable arithmetic policy.
func (v PolicyVersion) Resolve() (ArithmeticPolicy, error) {
	if v.PolicyKey == "" || v.Version <= 0 {
		return ArithmeticPolicy{}, fmt.Errorf("%w: invalid policy identity", ErrPolicyConfigInvalid)
	}
	if v.SchemaVersion <= 0 {
		return ArithmeticPolicy{}, fmt.Errorf("%w: invalid schema version %d", ErrPolicyConfigInvalid, v.SchemaVersion)
	}
	if !v.Classification.IsValid() {
		return ArithmeticPolicy{}, fmt.Errorf("%w: invalid policy classification %q", ErrPolicyConfigInvalid, v.Classification)
	}
	if v.Status != PolicyStatusApproved {
		return ArithmeticPolicy{}, ErrPolicyNotApproved
	}
	if v.ApprovedAt == nil {
		return ArithmeticPolicy{}, fmt.Errorf("%w: approved policy is missing approval time", ErrPolicyConfigInvalid)
	}

	_, configDigest, err := v.Config.CanonicalBytes()
	if err != nil {
		return ArithmeticPolicy{}, err
	}
	if v.ConfigSHA256 == "" || !strings.EqualFold(v.ConfigSHA256, configDigest) {
		return ArithmeticPolicy{}, fmt.Errorf("%w: config digest mismatch", ErrPolicyConfigInvalid)
	}

	allowedKinds := cloneDocumentKindMap(v.Config.AllowedKinds)
	voidAllowedKinds := cloneDocumentKindMap(v.Config.VoidAllowedKinds)
	taxTreatments := cloneTaxTreatmentRules(v.Config.TaxTreatments)
	customerRequirements := cloneCustomerRequirementMap(v.Config.CustomerRequirements)
	discountKinds := cloneDiscountKindMap(v.Config.DiscountKinds)

	return ArithmeticPolicy{
		PolicyKey:             v.PolicyKey,
		Version:               v.Version,
		SchemaVersion:         v.SchemaVersion,
		Classification:        v.Classification,
		Currency:              v.Config.Currency,
		CurrencyScale:         v.Config.CurrencyScale,
		MaximumQuantityScale:  v.Config.MaximumQuantityScale,
		MaximumUnitPriceScale: v.Config.MaximumUnitPriceScale,
		MaximumTaxRateScale:   v.Config.MaximumTaxRateScale,
		RoundingMode:          v.Config.RoundingMode,
		RoundingScope:         v.Config.RoundingScope,
		AdjustmentRange:       v.Config.AdjustmentRange,
		AllowedKinds:          allowedKinds,
		VoidAllowedKinds:      voidAllowedKinds,
		TaxTreatments:         taxTreatments,
		CustomerRequirements:  customerRequirements,
		DiscountKinds:         discountKinds,
		ConfigSHA256:          configDigest,
	}, nil
}

// ArithmeticPolicy is the immutable policy consumed by the calculator.
type ArithmeticPolicy struct {
	PolicyKey             string                             `json:"policyKey"`
	Version               int                                `json:"version"`
	SchemaVersion         int                                `json:"schemaVersion"`
	Classification        PolicyClassification               `json:"classification"`
	Currency              string                             `json:"currency"`
	CurrencyScale         int32                              `json:"currencyScale"`
	MaximumQuantityScale  int32                              `json:"maximumQuantityScale"`
	MaximumUnitPriceScale int32                              `json:"maximumUnitPriceScale"`
	MaximumTaxRateScale   int32                              `json:"maximumTaxRateScale"`
	RoundingMode          RoundingMode                       `json:"roundingMode"`
	RoundingScope         RoundingScope                      `json:"roundingScope"`
	AdjustmentRange       AdjustmentRange                    `json:"adjustmentRange"`
	AllowedKinds          map[DocumentKind]bool              `json:"allowedKinds,omitempty"`
	VoidAllowedKinds      map[DocumentKind]bool              `json:"voidAllowedKinds,omitempty"`
	TaxTreatments         map[string]TaxTreatmentRule        `json:"taxTreatments,omitempty"`
	CustomerRequirements  map[DocumentKind]CustomerIdentityRequirement `json:"customerRequirements,omitempty"`
	DiscountKinds         map[DiscountKind]bool              `json:"discountKinds,omitempty"`
	ConfigSHA256          string                             `json:"configSha256"`
}

// CanIssue reports whether the policy authorizes the supplied document kind.
func (p ArithmeticPolicy) CanIssue(kind DocumentKind) bool {
	return p.AllowedKinds[kind]
}

// CanVoid reports whether the policy authorizes voiding the supplied document kind.
func (p ArithmeticPolicy) CanVoid(kind DocumentKind) bool {
	return p.VoidAllowedKinds[kind]
}

// CustomerRequirement returns the configured customer identity requirement for a document kind.
func (p ArithmeticPolicy) CustomerRequirement(kind DocumentKind) (CustomerIdentityRequirement, bool) {
	req, ok := p.CustomerRequirements[kind]
	return req, ok
}

// TaxTreatment returns the configured tax treatment rule for a code.
func (p ArithmeticPolicy) TaxTreatment(code string) (TaxTreatmentRule, bool) {
	rule, ok := p.TaxTreatments[code]
	return rule, ok
}

// FiscalPolicyResolver resolves an approved policy row into an immutable policy.
type FiscalPolicyResolver interface {
	Resolve(policyKey string, version int) (ArithmeticPolicy, error)
}

// StaticFiscalPolicyResolver is a simple in-memory resolver useful for tests.
type StaticFiscalPolicyResolver map[string]map[int]PolicyVersion

// Resolve returns an approved immutable policy or a fail-closed error.
func (r StaticFiscalPolicyResolver) Resolve(policyKey string, version int) (ArithmeticPolicy, error) {
	versions, ok := r[policyKey]
	if !ok {
		return ArithmeticPolicy{}, ErrPolicyNotFound
	}
	stored, ok := versions[version]
	if !ok {
		return ArithmeticPolicy{}, ErrPolicyNotFound
	}
	if stored.PolicyKey != "" && stored.PolicyKey != policyKey {
		return ArithmeticPolicy{}, fmt.Errorf("%w: policy key mismatch", ErrPolicyConfigInvalid)
	}
	if stored.Version != 0 && stored.Version != version {
		return ArithmeticPolicy{}, fmt.Errorf("%w: version mismatch", ErrPolicyConfigInvalid)
	}
	return stored.Resolve()
}

// CustomerIdentity captures the customer inputs used by the calculator.
type CustomerIdentity struct {
	LegalName    string `json:"legalName,omitempty"`
	TaxIdentifier string `json:"taxIdentifier,omitempty"`
	CountryCode  string `json:"countryCode,omitempty"`
}

// DiscountInput describes the requested line discount.
type DiscountInput struct {
	Kind  DiscountKind `json:"kind"`
	Value Decimal      `json:"value"`
}

// LineInput captures the exact arithmetic inputs for one fiscal line.
type LineInput struct {
	Position         int           `json:"position"`
	Description      string        `json:"description,omitempty"`
	UnitCode         string        `json:"unitCode,omitempty"`
	Quantity         Decimal       `json:"quantity"`
	UnitPrice        Decimal       `json:"unitPrice"`
	Discount         DiscountInput `json:"discount"`
	TaxTreatmentCode string        `json:"taxTreatmentCode"`
	TaxRate          Decimal       `json:"taxRate"`
	ExemptionCode    string        `json:"exemptionCode,omitempty"`
	ExemptionReason  string        `json:"exemptionReason,omitempty"`
	SourceType       string        `json:"sourceType,omitempty"`
	SourceID         string        `json:"sourceId,omitempty"`
}

// DeclaredTotals holds optional caller-provided comparison values.
type DeclaredTotals struct {
	GrossTotal       *Decimal `json:"grossTotal,omitempty"`
	DiscountTotal    *Decimal `json:"discountTotal,omitempty"`
	NetTotal        *Decimal `json:"netTotal,omitempty"`
	TaxTotal        *Decimal `json:"taxTotal,omitempty"`
	RoundingAdjustment *Decimal `json:"roundingAdjustment,omitempty"`
	PayableTotal    *Decimal `json:"payableTotal,omitempty"`
}

// CalculationInput provides the exact line inputs for the calculator.
type CalculationInput struct {
	Kind           DocumentKind   `json:"kind"`
	Currency       string         `json:"currency"`
	Customer       CustomerIdentity `json:"customer"`
	Lines          []LineInput    `json:"lines"`
	DeclaredTotals *DeclaredTotals `json:"declaredTotals,omitempty"`
}

// CalculatedLine is the canonical output for a single line.
type CalculatedLine struct {
	Position         int           `json:"position"`
	Description      string        `json:"description,omitempty"`
	UnitCode         string        `json:"unitCode,omitempty"`
	Quantity         Decimal       `json:"quantity"`
	UnitPrice        Decimal       `json:"unitPrice"`
	DiscountKind     DiscountKind  `json:"discountKind"`
	DiscountValue    Decimal       `json:"discountValue"`
	DiscountAmount   Decimal       `json:"discountAmount"`
	GrossAmount      Decimal       `json:"grossAmount"`
	NetAmount        Decimal       `json:"netAmount"`
	TaxTreatmentCode string        `json:"taxTreatmentCode"`
	TaxRate          Decimal       `json:"taxRate"`
	TaxAmount        Decimal       `json:"taxAmount"`
	ExemptionCode    string        `json:"exemptionCode,omitempty"`
	ExemptionReason  string        `json:"exemptionReason,omitempty"`
	SourceType       string        `json:"sourceType,omitempty"`
	SourceID         string        `json:"sourceId,omitempty"`
	LineTotal        Decimal       `json:"lineTotal"`
}

// Totals aggregates the calculated monetary totals.
type Totals struct {
	GrossTotal        Decimal `json:"grossTotal"`
	DiscountTotal     Decimal `json:"discountTotal"`
	NetTotal         Decimal `json:"netTotal"`
	TaxTotal         Decimal `json:"taxTotal"`
	RoundingAdjustment Decimal `json:"roundingAdjustment"`
	PayableTotal     Decimal `json:"payableTotal"`
}

// CalculationResult is the canonical snapshot-like output from the pure calculator.
type CalculationResult struct {
	PolicyKey        string          `json:"policyKey"`
	PolicyVersion    int             `json:"policyVersion"`
	SchemaVersion    int             `json:"schemaVersion"`
	Classification   PolicyClassification `json:"classification"`
	Kind             DocumentKind    `json:"kind"`
	Currency         string          `json:"currency"`
	Customer         CustomerIdentity `json:"customer"`
	Lines            []CalculatedLine `json:"lines"`
	Totals           Totals          `json:"totals"`
	CanonicalBytes   []byte          `json:"canonicalBytes,omitempty"`
	CanonicalSHA256  string          `json:"canonicalSha256,omitempty"`
}

// Calculate performs the pure exact-amount computation under a resolved policy.
func Calculate(policy ArithmeticPolicy, input CalculationInput) (CalculationResult, error) {
	if err := policy.validateResolved(); err != nil {
		return CalculationResult{}, err
	}
	if !policy.CanIssue(input.Kind) {
		return CalculationResult{}, fmt.Errorf("%w: %s", ErrDocumentKindNotAllowed, input.Kind)
	}
	if !isUpperASCIIAlpha3(input.Currency) {
		return CalculationResult{}, fmt.Errorf("%w: invalid request currency %q", ErrCurrencyMismatch, input.Currency)
	}
	if !strings.EqualFold(input.Currency, policy.Currency) {
		return CalculationResult{}, fmt.Errorf("%w: request %s policy %s", ErrCurrencyMismatch, input.Currency, policy.Currency)
	}

	requirement, ok := policy.CustomerRequirement(input.Kind)
	if !ok {
		return CalculationResult{}, fmt.Errorf("%w: missing customer requirement for %s", ErrPolicyConfigInvalid, input.Kind)
	}
	if err := validateCustomerIdentity(requirement, input.Customer); err != nil {
		return CalculationResult{}, err
	}

	if len(input.Lines) == 0 {
		return CalculationResult{}, fmt.Errorf("%w: at least one line is required", ErrCalculationLineInvalid)
	}

	lines, err := sortAndValidateLines(input.Lines)
	if err != nil {
		return CalculationResult{}, err
	}

	calculatedLines := make([]CalculatedLine, 0, len(lines))
	var exactNet Decimal = ZeroDecimal()
	var exactTax Decimal = ZeroDecimal()
	var roundedGross Decimal = ZeroDecimal()
	var roundedDiscount Decimal = ZeroDecimal()
	var roundedNet Decimal = ZeroDecimal()
	var roundedTax Decimal = ZeroDecimal()
	var roundedLineTotal Decimal = ZeroDecimal()

	for _, line := range lines {
		computed, exactLine, err := calculateLine(policy, input.Kind, line)
		if err != nil {
			return CalculationResult{}, err
		}
		calculatedLines = append(calculatedLines, computed)
		exactNet = exactNet.Add(exactLine.Net)
		exactTax = exactTax.Add(exactLine.Tax)
		roundedGross = roundedGross.Add(computed.GrossAmount)
		roundedDiscount = roundedDiscount.Add(computed.DiscountAmount)
		roundedNet = roundedNet.Add(computed.NetAmount)
		roundedTax = roundedTax.Add(computed.TaxAmount)
		roundedLineTotal = roundedLineTotal.Add(computed.LineTotal)
	}

	payableExact := exactNet.Add(exactTax)
	payableRounded, err := payableExact.Round(policy.CurrencyScale, policy.RoundingMode)
	if err != nil {
		return CalculationResult{}, err
	}

	var adjustment Decimal
	if policy.RoundingScope == RoundingScopePerLine {
		adjustment = ZeroDecimal()
		payableRounded = roundedLineTotal
	} else {
		adjustment = payableRounded.Sub(roundedLineTotal)
		if !policy.AdjustmentRange.Contains(adjustment) {
			return CalculationResult{}, fmt.Errorf("%w: %s", ErrRoundingAdjustmentOutOfRange, adjustment)
		}
	}

	result := CalculationResult{
		PolicyKey:      policy.PolicyKey,
		PolicyVersion:  policy.Version,
		SchemaVersion:  policy.SchemaVersion,
		Classification: policy.Classification,
		Kind:           input.Kind,
		Currency:       policy.Currency,
		Customer:       input.Customer,
		Lines:          calculatedLines,
		Totals: Totals{
			GrossTotal:        roundedGross,
			DiscountTotal:     roundedDiscount,
			NetTotal:          roundedNet,
			TaxTotal:          roundedTax,
			RoundingAdjustment: adjustment,
			PayableTotal:      payableRounded,
		},
	}

	if err := compareDeclaredTotals(input.DeclaredTotals, result.Totals); err != nil {
		return CalculationResult{}, err
	}

	canonicalBytes, canonicalHash, err := result.canonicalBytes()
	if err != nil {
		return CalculationResult{}, err
	}
	result.CanonicalBytes = canonicalBytes
	result.CanonicalSHA256 = canonicalHash
	return result, nil
}

type lineExactTotals struct {
	Gross   Decimal
	Discount Decimal
	Net     Decimal
	Tax     Decimal
}

func calculateLine(policy ArithmeticPolicy, kind DocumentKind, line LineInput) (CalculatedLine, lineExactTotals, error) {
	if line.Position <= 0 {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line position must be positive", ErrCalculationLineInvalid)
	}
	if line.Quantity.Cmp(ZeroDecimal()) <= 0 {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d quantity must be greater than zero", ErrCalculationLineInvalid, line.Position)
	}
	if line.UnitPrice.Cmp(ZeroDecimal()) < 0 {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d unit price must not be negative", ErrCalculationLineInvalid, line.Position)
	}
	if line.Quantity.Scale() > policy.MaximumQuantityScale {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d quantity scale exceeds policy limit", ErrCalculationLineInvalid, line.Position)
	}
	if line.UnitPrice.Scale() > policy.MaximumUnitPriceScale {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d unit price scale exceeds policy limit", ErrCalculationLineInvalid, line.Position)
	}
	if line.TaxRate.Scale() > policy.MaximumTaxRateScale {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d tax rate scale exceeds policy limit", ErrCalculationLineInvalid, line.Position)
	}
	if line.TaxTreatmentCode == "" {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d missing tax treatment code", ErrTaxTreatmentMissing, line.Position)
	}

	rule, ok := policy.TaxTreatment(line.TaxTreatmentCode)
	if !ok {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d tax treatment %q", ErrTaxTreatmentMissing, line.Position, line.TaxTreatmentCode)
	}
	if len(rule.AllowedKinds) > 0 && !rule.AllowedKinds[kind] {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: tax treatment %s is not allowed for %s", ErrTaxTreatmentMissing, line.TaxTreatmentCode, kind)
	}

	discountKind := line.Discount.Kind
	if discountKind == "" {
		if line.Discount.Value.IsZero() {
			discountKind = DiscountKindNone
		} else {
			return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d discount kind is required when a discount value is present", ErrDiscountKindNotAllowed, line.Position)
		}
	}
	if !discountKind.IsValid() {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d has invalid discount kind %q", ErrDiscountKindNotAllowed, line.Position, discountKind)
	}
	if discountKind != DiscountKindNone && !policy.DiscountKinds[discountKind] {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d discount kind %s", ErrDiscountKindNotAllowed, line.Position, discountKind)
	}
	if discountKind == DiscountKindAmount && line.Discount.Value.Scale() > policy.CurrencyScale {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d discount scale exceeds currency scale", ErrCalculationLineInvalid, line.Position)
	}
	if discountKind == DiscountKindPercent && line.Discount.Value.Scale() > policy.MaximumTaxRateScale {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d discount percent scale exceeds policy limit", ErrCalculationLineInvalid, line.Position)
	}

	gross := line.Quantity.Mul(line.UnitPrice)
	if line.Discount.Value.Cmp(ZeroDecimal()) < 0 {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d discount value must not be negative", ErrCalculationLineInvalid, line.Position)
	}

	var discountAmount Decimal
	switch discountKind {
	case DiscountKindNone:
		if !line.Discount.Value.IsZero() {
			return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d discount value must be zero when kind is none", ErrCalculationLineInvalid, line.Position)
		}
		discountAmount = ZeroDecimal()
	case DiscountKindAmount:
		discountAmount = line.Discount.Value
	case DiscountKindPercent:
		if line.Discount.Value.Cmp(MustParseDecimal("100")) > 0 {
			return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d discount percent exceeds 100", ErrCalculationLineInvalid, line.Position)
		}
		discountAmount = gross.Mul(line.Discount.Value).Div(MustParseDecimal("100"))
	default:
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d has unsupported discount kind %q", ErrDiscountKindNotAllowed, line.Position, discountKind)
	}
	if discountAmount.Cmp(gross) > 0 {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d discount exceeds gross amount", ErrCalculationLineInvalid, line.Position)
	}

	net := gross.Sub(discountAmount)
	if net.Cmp(ZeroDecimal()) < 0 {
		return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d net amount is negative", ErrCalculationLineInvalid, line.Position)
	}

	var taxAmount Decimal
	if rule.Exempt {
		if !line.TaxRate.IsZero() {
			return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: exempt tax treatment %s must use zero tax rate", ErrTaxTreatmentMissing, line.TaxTreatmentCode)
		}
		if rule.ExemptionCodeRequired && strings.TrimSpace(line.ExemptionCode) == "" {
			return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d exemption code is required", ErrTaxTreatmentMissing, line.Position)
		}
		if rule.ExemptionReasonRequired && strings.TrimSpace(line.ExemptionReason) == "" {
			return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d exemption reason is required", ErrTaxTreatmentMissing, line.Position)
		}
		taxAmount = ZeroDecimal()
	} else {
		if line.TaxRate.Cmp(rule.Rate) != 0 {
			return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d tax rate %s does not match policy rate %s", ErrTaxTreatmentMissing, line.Position, line.TaxRate, rule.Rate)
		}
		if strings.TrimSpace(line.ExemptionCode) != "" || strings.TrimSpace(line.ExemptionReason) != "" {
			return CalculatedLine{}, lineExactTotals{}, fmt.Errorf("%w: line %d exemption fields must be empty for taxable lines", ErrTaxTreatmentMissing, line.Position)
		}
		taxAmount = net.Mul(rule.Rate).Div(MustParseDecimal("100"))
	}


	roundedGross, err := gross.Round(policy.CurrencyScale, policy.RoundingMode)
	if err != nil {
		return CalculatedLine{}, lineExactTotals{}, err
	}
	roundedDiscount, err := discountAmount.Round(policy.CurrencyScale, policy.RoundingMode)
	if err != nil {
		return CalculatedLine{}, lineExactTotals{}, err
	}
	roundedNet, err := net.Round(policy.CurrencyScale, policy.RoundingMode)
	if err != nil {
		return CalculatedLine{}, lineExactTotals{}, err
	}
	roundedTax, err := taxAmount.Round(policy.CurrencyScale, policy.RoundingMode)
	if err != nil {
		return CalculatedLine{}, lineExactTotals{}, err
	}
	roundedLineTotal, err := roundedNet.Add(roundedTax).Round(policy.CurrencyScale, policy.RoundingMode)
	if err != nil {
		return CalculatedLine{}, lineExactTotals{}, err
	}

	calculated := CalculatedLine{
		Position:         line.Position,
		Description:      line.Description,
		UnitCode:         line.UnitCode,
		Quantity:         line.Quantity,
		UnitPrice:        line.UnitPrice,
		DiscountKind:     discountKind,
		DiscountValue:    line.Discount.Value,
		DiscountAmount:   roundedDiscount,
		GrossAmount:      roundedGross,
		NetAmount:        roundedNet,
		TaxTreatmentCode: line.TaxTreatmentCode,
		TaxRate:          line.TaxRate,
		TaxAmount:        roundedTax,
		ExemptionCode:    line.ExemptionCode,
		ExemptionReason:  line.ExemptionReason,
		SourceType:       line.SourceType,
		SourceID:         line.SourceID,
		LineTotal:        roundedLineTotal,
	}

	if policy.RoundingScope == RoundingScopeDocument {
		// Document rounding still uses the rounded line values for presentation,
		// while the document adjustment is derived from exact totals.
	}

	return calculated, lineExactTotals{Gross: gross, Discount: discountAmount, Net: net, Tax: taxAmount}, nil
}

func (p ArithmeticPolicy) validateResolved() error {
	if p.PolicyKey == "" || p.Version <= 0 {
		return fmt.Errorf("%w: invalid policy identity", ErrPolicyConfigInvalid)
	}
	if p.SchemaVersion <= 0 {
		return fmt.Errorf("%w: invalid schema version %d", ErrPolicyConfigInvalid, p.SchemaVersion)
	}
	if !p.Classification.IsValid() {
		return fmt.Errorf("%w: invalid policy classification %q", ErrPolicyConfigInvalid, p.Classification)
	}
	if !isUpperASCIIAlpha3(p.Currency) {
		return fmt.Errorf("%w: invalid policy currency %q", ErrPolicyConfigInvalid, p.Currency)
	}
	if !p.RoundingMode.IsValid() || !p.RoundingScope.IsValid() {
		return fmt.Errorf("%w: invalid rounding configuration", ErrPolicyConfigInvalid)
	}
	if err := p.AdjustmentRange.Validate(); err != nil {
		return err
	}
	if len(p.TaxTreatments) == 0 {
		return fmt.Errorf("%w: no tax treatments configured", ErrPolicyConfigInvalid)
	}
	allowedCount := 0
	for kind, allowed := range p.AllowedKinds {
		if kind.IsValid() && allowed {
			allowedCount++
			if _, ok := p.CustomerRequirements[kind]; !ok {
				return fmt.Errorf("%w: missing customer requirement for %s", ErrPolicyConfigInvalid, kind)
			}
		}
	}
	if allowedCount == 0 {
		return fmt.Errorf("%w: no allowed document kinds configured", ErrPolicyConfigInvalid)
	}
	for code, rule := range p.TaxTreatments {
		if code == "" {
			return fmt.Errorf("%w: empty tax treatment code", ErrPolicyConfigInvalid)
		}
		rule.Code = code
		if err := rule.Validate(p.MaximumTaxRateScale); err != nil {
			return err
		}
	}
	return nil
}

func (r CalculationResult) canonicalBytes() ([]byte, string, error) {
	orderedLines := make([]CalculatedLine, len(r.Lines))
	copy(orderedLines, r.Lines)
	sort.Slice(orderedLines, func(i, j int) bool {
		if orderedLines[i].Position != orderedLines[j].Position {
			return orderedLines[i].Position < orderedLines[j].Position
		}
		if orderedLines[i].Description != orderedLines[j].Description {
			return orderedLines[i].Description < orderedLines[j].Description
		}
		return orderedLines[i].UnitCode < orderedLines[j].UnitCode
	})

	payload := calculationCanonical{
		PolicyKey:      r.PolicyKey,
		PolicyVersion:   r.PolicyVersion,
		SchemaVersion:   r.SchemaVersion,
		Classification:  r.Classification,
		Kind:            r.Kind,
		Currency:        r.Currency,
		Customer:        r.Customer,
		Lines:           orderedLines,
		Totals:          r.Totals,
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(bytes)
	return bytes, hex.EncodeToString(sum[:]), nil
}

func compareDeclaredTotals(expected *DeclaredTotals, actual Totals) error {
	if expected == nil {
		return nil
	}
	var mismatches []string
	check := func(field string, declared *Decimal, computed Decimal) {
		if declared == nil {
			return
		}
		if !declared.Equal(computed) {
			mismatches = append(mismatches, fmt.Sprintf("%s expected %s got %s", field, declared.String(), computed.String()))
		}
	}

	check("grossTotal", expected.GrossTotal, actual.GrossTotal)
	check("discountTotal", expected.DiscountTotal, actual.DiscountTotal)
	check("netTotal", expected.NetTotal, actual.NetTotal)
	check("taxTotal", expected.TaxTotal, actual.TaxTotal)
	check("roundingAdjustment", expected.RoundingAdjustment, actual.RoundingAdjustment)
	check("payableTotal", expected.PayableTotal, actual.PayableTotal)

	if len(mismatches) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrDeclaredTotalsMismatch, strings.Join(mismatches, "; "))
}

func validateCustomerIdentity(requirement CustomerIdentityRequirement, customer CustomerIdentity) error {
	if requirement.LegalName && strings.TrimSpace(customer.LegalName) == "" {
		return fmt.Errorf("%w: legal name is required", ErrCustomerIdentityMissing)
	}
	if requirement.TaxIdentifier && strings.TrimSpace(customer.TaxIdentifier) == "" {
		return fmt.Errorf("%w: tax identifier is required", ErrCustomerIdentityMissing)
	}
	if requirement.CountryCode && strings.TrimSpace(customer.CountryCode) == "" {
		return fmt.Errorf("%w: country code is required", ErrCustomerIdentityMissing)
	}
	return nil
}

func sortAndValidateLines(lines []LineInput) ([]LineInput, error) {
	seen := make(map[int]struct{}, len(lines))
	ordered := make([]LineInput, len(lines))
	copy(ordered, lines)
	for _, line := range ordered {
		if line.Position <= 0 {
			return nil, fmt.Errorf("%w: line position must be positive", ErrCalculationLineInvalid)
		}
		if _, ok := seen[line.Position]; ok {
			return nil, fmt.Errorf("%w: %d", ErrDuplicateLinePosition, line.Position)
		}
		seen[line.Position] = struct{}{}
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Position < ordered[j].Position
	})
	return ordered, nil
}

func isUpperASCIIAlpha3(value string) bool {
	if len(value) != 3 {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

func cloneDocumentKindMap(src map[DocumentKind]bool) map[DocumentKind]bool {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[DocumentKind]bool, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneDiscountKindMap(src map[DiscountKind]bool) map[DiscountKind]bool {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[DiscountKind]bool, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneCustomerRequirementMap(src map[DocumentKind]CustomerIdentityRequirement) map[DocumentKind]CustomerIdentityRequirement {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[DocumentKind]CustomerIdentityRequirement, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneTaxTreatmentRules(src map[string]TaxTreatmentRule) map[string]TaxTreatmentRule {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]TaxTreatmentRule, len(src))
	for key, value := range src {
		value.Code = key
		if len(value.AllowedKinds) > 0 {
			value.AllowedKinds = cloneDocumentKindMap(value.AllowedKinds)
		}
		dst[key] = value
	}
	return dst
}

type policyConfigCanonical struct {
	Currency              string                             `json:"currency"`
	CurrencyScale         int32                              `json:"currencyScale"`
	MaximumQuantityScale  int32                              `json:"maximumQuantityScale"`
	MaximumUnitPriceScale int32                              `json:"maximumUnitPriceScale"`
	MaximumTaxRateScale   int32                              `json:"maximumTaxRateScale"`
	RoundingMode          RoundingMode                       `json:"roundingMode"`
	RoundingScope         RoundingScope                      `json:"roundingScope"`
	AdjustmentRange       AdjustmentRange                    `json:"adjustmentRange"`
	AllowedKinds          map[DocumentKind]bool              `json:"allowedKinds,omitempty"`
	VoidAllowedKinds      map[DocumentKind]bool              `json:"voidAllowedKinds,omitempty"`
	TaxTreatments         map[string]TaxTreatmentRule        `json:"taxTreatments,omitempty"`
	CustomerRequirements  map[DocumentKind]CustomerIdentityRequirement `json:"customerRequirements,omitempty"`
	DiscountKinds         map[DiscountKind]bool              `json:"discountKinds,omitempty"`
}

type calculationCanonical struct {
	PolicyKey      string            `json:"policyKey"`
	PolicyVersion   int               `json:"policyVersion"`
	SchemaVersion   int               `json:"schemaVersion"`
	Classification  PolicyClassification `json:"classification"`
	Kind            DocumentKind     `json:"kind"`
	Currency        string           `json:"currency"`
	Customer        CustomerIdentity `json:"customer"`
	Lines           []CalculatedLine `json:"lines"`
	Totals          Totals           `json:"totals"`
}
