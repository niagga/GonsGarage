package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrFiscalDocumentStateInvalid is returned when a document state is unknown.
	ErrFiscalDocumentStateInvalid = errors.New("invalid fiscal document state")
	// ErrFiscalDocumentTransitionNotAllowed is returned when a transition is not permitted.
	ErrFiscalDocumentTransitionNotAllowed = errors.New("fiscal document transition not allowed")
	// ErrFiscalDocumentFrozen is returned when a frozen field would be changed.
	ErrFiscalDocumentFrozen = errors.New("fiscal document is frozen")
	// ErrFiscalDocumentRoleDenied is returned when the supplied role cannot perform the action.
	ErrFiscalDocumentRoleDenied = errors.New("fiscal document role is not allowed")
	// ErrFiscalDocumentSupersessionNotAllowed is returned when rejected-intent supersession is invalid.
	ErrFiscalDocumentSupersessionNotAllowed = errors.New("rejected fiscal document may only be superseded")
)

// FiscalDocumentState models the provider-neutral lifecycle.
type FiscalDocumentState string

const (
	FiscalDocumentStateDraft                  FiscalDocumentState = "draft"
	FiscalDocumentStatePending                FiscalDocumentState = "pending"
	FiscalDocumentStateDispatching            FiscalDocumentState = "dispatching"
	FiscalDocumentStateIssued                 FiscalDocumentState = "issued"
	FiscalDocumentStateRejected               FiscalDocumentState = "rejected"
	FiscalDocumentStateConnectionActionNeeded FiscalDocumentState = "connection_action_required"
	FiscalDocumentStateRetryableFailure       FiscalDocumentState = "retryable_failure"
	FiscalDocumentStateOutcomeUnknown         FiscalDocumentState = "outcome_unknown"
	FiscalDocumentStateVoidPending            FiscalDocumentState = "void_pending"
	FiscalDocumentStateVoidOutcomeUnknown     FiscalDocumentState = "void_outcome_unknown"
	FiscalDocumentStateVoided                 FiscalDocumentState = "voided"
)

// IsValid reports whether the state is one of the supported lifecycle values.
func (s FiscalDocumentState) IsValid() bool {
	switch s {
	case FiscalDocumentStateDraft,
		FiscalDocumentStatePending,
		FiscalDocumentStateDispatching,
		FiscalDocumentStateIssued,
		FiscalDocumentStateRejected,
		FiscalDocumentStateConnectionActionNeeded,
		FiscalDocumentStateRetryableFailure,
		FiscalDocumentStateOutcomeUnknown,
		FiscalDocumentStateVoidPending,
		FiscalDocumentStateVoidOutcomeUnknown,
		FiscalDocumentStateVoided:
		return true
	default:
		return false
	}
}

// IsFrozen reports whether the document is beyond draft editing.
func (s FiscalDocumentState) IsFrozen() bool {
	return s != FiscalDocumentStateDraft
}

// FiscalPresentationState is the simplified provider-neutral status shown to clients.
type FiscalPresentationState string

const (
	FiscalPresentationStateDraft       FiscalPresentationState = "draft"
	FiscalPresentationStatePending     FiscalPresentationState = "pending"
	FiscalPresentationStateFinalized   FiscalPresentationState = "finalized"
	FiscalPresentationStateVoided      FiscalPresentationState = "voided"
	FiscalPresentationStateUnavailable FiscalPresentationState = "unavailable"
)

// Presentation maps the lifecycle to the simplified client-facing state.
func (s FiscalDocumentState) Presentation() FiscalPresentationState {
	switch s {
	case FiscalDocumentStateDraft:
		return FiscalPresentationStateDraft
	case FiscalDocumentStatePending, FiscalDocumentStateDispatching, FiscalDocumentStateVoidPending:
		return FiscalPresentationStatePending
	case FiscalDocumentStateIssued:
		return FiscalPresentationStateFinalized
	case FiscalDocumentStateVoided:
		return FiscalPresentationStateVoided
	default:
		return FiscalPresentationStateUnavailable
	}
}

// FiscalActorRole identifies the caller role used by the domain matrix.
type FiscalActorRole string

const (
	FiscalActorRoleEmployee FiscalActorRole = "employee"
	FiscalActorRoleManager  FiscalActorRole = "manager"
	FiscalActorRoleAdmin    FiscalActorRole = "admin"
	FiscalActorRoleClient   FiscalActorRole = "client"
	FiscalActorRoleSystem   FiscalActorRole = "system"
)

// IsValid reports whether the role is recognized.
func (r FiscalActorRole) IsValid() bool {
	switch r {
	case FiscalActorRoleEmployee, FiscalActorRoleManager, FiscalActorRoleAdmin, FiscalActorRoleClient, FiscalActorRoleSystem:
		return true
	default:
		return false
	}
}

// IsStaff reports whether the role belongs to an internal staff user.
func (r FiscalActorRole) IsStaff() bool {
	switch r {
	case FiscalActorRoleEmployee, FiscalActorRoleManager, FiscalActorRoleAdmin:
		return true
	default:
		return false
	}
}

// CanManage reports whether the role may run privileged fiscal actions.
func (r FiscalActorRole) CanManage() bool {
	return r == FiscalActorRoleManager || r == FiscalActorRoleAdmin
}

// FiscalDocumentAction identifies the visible action matrix.
type FiscalDocumentAction string

const (
	FiscalDocumentActionView      FiscalDocumentAction = "view"
	FiscalDocumentActionEdit      FiscalDocumentAction = "edit"
	FiscalDocumentActionDelete    FiscalDocumentAction = "delete"
	FiscalDocumentActionFinalize  FiscalDocumentAction = "finalize"
	FiscalDocumentActionRetry     FiscalDocumentAction = "retry"
	FiscalDocumentActionReconcile FiscalDocumentAction = "reconcile"
	FiscalDocumentActionVoid      FiscalDocumentAction = "void"
	FiscalDocumentActionSupersede FiscalDocumentAction = "supersede"
)

// IsValid reports whether the action is recognized.
func (a FiscalDocumentAction) IsValid() bool {
	switch a {
	case FiscalDocumentActionView,
		FiscalDocumentActionEdit,
		FiscalDocumentActionDelete,
		FiscalDocumentActionFinalize,
		FiscalDocumentActionRetry,
		FiscalDocumentActionReconcile,
		FiscalDocumentActionVoid,
		FiscalDocumentActionSupersede:
		return true
	default:
		return false
	}
}

// FiscalFreezeInput captures the immutable finalization fields.
type FiscalFreezeInput struct {
	ProviderKey       string
	ConnectionID      uuid.UUID
	IntentKey         uuid.UUID
	IssueOperationKey string
	FinalizedBy       uuid.UUID
	FrozenAt          time.Time
}

// FrozenFiscalDocument is the immutable DTO used after finalization.
type FrozenFiscalDocument struct {
	ID                   uuid.UUID
	SourceInvoiceID      uuid.UUID
	IntentSlot           string
	Kind                 DocumentKind
	State                FiscalDocumentState
	Version              int64
	SupersedesDocumentID *uuid.UUID
	ProviderKey          string
	ConnectionID         *uuid.UUID
	IntentKey            *uuid.UUID
	IssueOperationKey    string
	VoidOperationKey     string
	ProviderReference    string
	ProviderNumber       string
	ProviderConfirmedAt  *time.Time
	IssuedAt             *time.Time
	VoidedAt             *time.Time
	FrozenAt             *time.Time
	FinalizedBy          *uuid.UUID
}

// HasProviderFixation reports whether the provider, connection, and intent were fixed.
func (f FrozenFiscalDocument) HasProviderFixation() bool {
	return f.ProviderKey != "" && f.ConnectionID != nil && *f.ConnectionID != uuid.Nil && f.IntentKey != nil && *f.IntentKey != uuid.Nil && strings.TrimSpace(f.IssueOperationKey) != ""
}

// FiscalDocument is the lifecycle aggregate persisted by the schema.
type FiscalDocument struct {
	ID                   uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey"`
	SourceInvoiceID      uuid.UUID           `json:"sourceInvoiceId" gorm:"type:uuid;not null;index"`
	IntentSlot           string              `json:"intentSlot" gorm:"type:varchar(40);not null;default:primary_sale;index"`
	Kind                 DocumentKind        `json:"kind" gorm:"type:varchar(2);not null;index"`
	State                FiscalDocumentState `json:"state" gorm:"type:varchar(32);not null;index"`
	Version              int64               `json:"version" gorm:"not null;default:1"`
	SupersedesDocumentID *uuid.UUID          `json:"supersedesDocumentId,omitempty" gorm:"type:uuid;index"`
	SupersededAt         *time.Time          `json:"supersededAt,omitempty" gorm:"column:superseded_at;index"`
	ProviderKey          string              `json:"providerKey,omitempty" gorm:"type:varchar(40);index"`
	ConnectionID         *uuid.UUID          `json:"connectionId,omitempty" gorm:"type:uuid;index"`
	IntentKey            *uuid.UUID          `json:"intentKey,omitempty" gorm:"type:uuid;index"`
	IssueOperationKey    string              `json:"issueOperationKey,omitempty" gorm:"type:varchar(160);index"`
	VoidOperationKey     string              `json:"voidOperationKey,omitempty" gorm:"type:varchar(160);index"`
	ProviderReference    string              `json:"providerReference,omitempty" gorm:"type:varchar(255);index"`
	ProviderNumber       string              `json:"providerNumber,omitempty" gorm:"type:varchar(255)"`
	ProviderConfirmedAt  *time.Time          `json:"providerConfirmedAt,omitempty" gorm:"column:provider_confirmed_at"`
	IssuedAt             *time.Time          `json:"issuedAt,omitempty" gorm:"column:issued_at"`
	VoidedAt             *time.Time          `json:"voidedAt,omitempty" gorm:"column:voided_at"`
	FrozenAt             *time.Time          `json:"frozenAt,omitempty" gorm:"column:frozen_at"`
	LastErrorClass       string              `json:"lastErrorClass,omitempty" gorm:"column:last_error_class;type:varchar(32)"`
	LastErrorCode        string              `json:"lastErrorCode,omitempty" gorm:"column:last_error_code;type:varchar(80)"`
	LastErrorMessage     string              `json:"lastErrorMessage,omitempty" gorm:"column:last_error_message;type:text"`
	CreatedBy            uuid.UUID           `json:"createdBy" gorm:"type:uuid;not null;index"`
	FinalizedBy          *uuid.UUID          `json:"finalizedBy,omitempty" gorm:"type:uuid;index"`
	CreatedAt            time.Time           `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time           `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

func (FiscalDocument) TableName() string {
	return "fiscal_documents"
}

// Validate checks the structural invariants used by the domain slice.
func (d *FiscalDocument) Validate() error {
	if d == nil {
		return errors.New("fiscal document is nil")
	}
	if d.SourceInvoiceID == uuid.Nil {
		return errors.New("source invoice is required")
	}
	if !d.Kind.IsValid() {
		return errors.New("invalid fiscal document kind")
	}
	if !d.State.IsValid() {
		return ErrFiscalDocumentStateInvalid
	}
	if d.State == FiscalDocumentStateDraft {
		if d.ProviderKey != "" || d.ConnectionID != nil || d.IntentKey != nil || d.IssueOperationKey != "" || d.VoidOperationKey != "" || d.FrozenAt != nil || d.FinalizedBy != nil || d.ProviderConfirmedAt != nil || d.IssuedAt != nil || d.VoidedAt != nil || d.SupersedesDocumentID != nil || d.SupersededAt != nil {
			return fmt.Errorf("%w: draft may not contain frozen fields", ErrFiscalDocumentFrozen)
		}
		return nil
	}
	if strings.TrimSpace(d.ProviderKey) == "" || d.ConnectionID == nil || *d.ConnectionID == uuid.Nil || d.IntentKey == nil || *d.IntentKey == uuid.Nil || strings.TrimSpace(d.IssueOperationKey) == "" || d.FrozenAt == nil || d.FrozenAt.IsZero() {
		return fmt.Errorf("%w: frozen fiscal document is missing required fields", ErrFiscalDocumentFrozen)
	}
	if d.State == FiscalDocumentStateVoidPending || d.State == FiscalDocumentStateVoidOutcomeUnknown || d.State == FiscalDocumentStateVoided {
		if strings.TrimSpace(d.VoidOperationKey) == "" {
			return fmt.Errorf("%w: void operation key is required", ErrFiscalDocumentFrozen)
		}
	}
	return nil
}

// Frozen returns an immutable DTO copy of the current document fields.
func (d FiscalDocument) Frozen() FrozenFiscalDocument {
	return FrozenFiscalDocument{
		ID:                   d.ID,
		SourceInvoiceID:      d.SourceInvoiceID,
		IntentSlot:           d.IntentSlot,
		Kind:                 d.Kind,
		State:                d.State,
		Version:              d.Version,
		SupersedesDocumentID: cloneUUIDPtr(d.SupersedesDocumentID),
		ProviderKey:          d.ProviderKey,
		ConnectionID:         cloneUUIDPtr(d.ConnectionID),
		IntentKey:            cloneUUIDPtr(d.IntentKey),
		IssueOperationKey:    d.IssueOperationKey,
		VoidOperationKey:     d.VoidOperationKey,
		ProviderReference:    d.ProviderReference,
		ProviderNumber:       d.ProviderNumber,
		ProviderConfirmedAt:  cloneTimePtr(d.ProviderConfirmedAt),
		IssuedAt:             cloneTimePtr(d.IssuedAt),
		VoidedAt:             cloneTimePtr(d.VoidedAt),
		FrozenAt:             cloneTimePtr(d.FrozenAt),
		FinalizedBy:          cloneUUIDPtr(d.FinalizedBy),
	}
}

// ProviderFixed reports whether the provider and connection are already fixed.
func (d FiscalDocument) ProviderFixed() bool {
	return strings.TrimSpace(d.ProviderKey) != "" && d.ConnectionID != nil && *d.ConnectionID != uuid.Nil
}

// IntentFixed reports whether the issue intent is already fixed.
func (d FiscalDocument) IntentFixed() bool {
	return d.IntentKey != nil && *d.IntentKey != uuid.Nil && strings.TrimSpace(d.IssueOperationKey) != ""
}

// AllowedActions returns the role/action matrix for the current lifecycle state.
func (d FiscalDocument) AllowedActions(role FiscalActorRole, owns bool) []FiscalDocumentAction {
	actions := make([]FiscalDocumentAction, 0, 4)
	if role == FiscalActorRoleClient {
		if owns && d.State != FiscalDocumentStateDraft {
			actions = append(actions, FiscalDocumentActionView)
		}
		return actions
	}
	if !role.IsStaff() {
		return actions
	}
	actions = append(actions, FiscalDocumentActionView)
	switch d.State {
	case FiscalDocumentStateDraft:
		actions = append(actions, FiscalDocumentActionEdit, FiscalDocumentActionDelete)
		if role.CanManage() {
			actions = append(actions, FiscalDocumentActionFinalize)
		}
	case FiscalDocumentStateRejected:
		actions = append(actions, FiscalDocumentActionSupersede)
	case FiscalDocumentStateConnectionActionNeeded, FiscalDocumentStateRetryableFailure:
		if role.CanManage() {
			actions = append(actions, FiscalDocumentActionRetry)
		}
	case FiscalDocumentStateOutcomeUnknown, FiscalDocumentStateVoidOutcomeUnknown:
		if role.CanManage() {
			actions = append(actions, FiscalDocumentActionReconcile)
		}
	case FiscalDocumentStateIssued:
		if role.CanManage() {
			actions = append(actions, FiscalDocumentActionVoid)
		}
	}
	return dedupeFiscalDocumentActions(actions)
}

// CanAct reports whether the role may perform the requested action.
func (d FiscalDocument) CanAct(role FiscalActorRole, owns bool, action FiscalDocumentAction) bool {
	for _, allowed := range d.AllowedActions(role, owns) {
		if allowed == action {
			return true
		}
	}
	return false
}

// TransitionTo applies a guarded lifecycle transition.
func (d *FiscalDocument) TransitionTo(next FiscalDocumentState, at time.Time) error {
	if d == nil {
		return errors.New("fiscal document is nil")
	}
	if !next.IsValid() {
		return ErrFiscalDocumentStateInvalid
	}
	if d.State == next {
		return fmt.Errorf("%w: %s -> %s", ErrFiscalDocumentTransitionNotAllowed, d.State, next)
	}
	if !fiscalDocumentTransitionAllowed[d.State][next] {
		return fmt.Errorf("%w: %s -> %s", ErrFiscalDocumentTransitionNotAllowed, d.State, next)
	}
	if next == FiscalDocumentStatePending && !d.ProviderFixed() {
		return fmt.Errorf("%w: provider fixation is required before pending", ErrFiscalDocumentFrozen)
	}
	if next == FiscalDocumentStatePending && !d.IntentFixed() {
		return fmt.Errorf("%w: intent fixation is required before pending", ErrFiscalDocumentFrozen)
	}
	if next == FiscalDocumentStateDispatching && (!d.ProviderFixed() || !d.IntentFixed()) {
		return fmt.Errorf("%w: frozen intent is required before dispatching", ErrFiscalDocumentFrozen)
	}
	if next == FiscalDocumentStateVoidPending && strings.TrimSpace(d.VoidOperationKey) == "" {
		return fmt.Errorf("%w: void operation key is required before void_pending", ErrFiscalDocumentFrozen)
	}
	d.State = next
	if !at.IsZero() {
		d.UpdatedAt = at
	}
	return nil
}

// Finalize freezes the document and moves it to pending.
func (d *FiscalDocument) Finalize(role FiscalActorRole, input FiscalFreezeInput) (FrozenFiscalDocument, error) {
	if d == nil {
		return FrozenFiscalDocument{}, errors.New("fiscal document is nil")
	}
	if !d.CanAct(role, false, FiscalDocumentActionFinalize) {
		return FrozenFiscalDocument{}, ErrFiscalDocumentRoleDenied
	}
	if d.State != FiscalDocumentStateDraft {
		return FrozenFiscalDocument{}, fmt.Errorf("%w: finalize requires draft", ErrFiscalDocumentTransitionNotAllowed)
	}
	if strings.TrimSpace(input.ProviderKey) == "" {
		return FrozenFiscalDocument{}, fmt.Errorf("%w: provider key is required", ErrFiscalDocumentFrozen)
	}
	if input.ConnectionID == uuid.Nil {
		return FrozenFiscalDocument{}, fmt.Errorf("%w: connection is required", ErrFiscalDocumentFrozen)
	}
	if input.IntentKey == uuid.Nil {
		return FrozenFiscalDocument{}, fmt.Errorf("%w: intent key is required", ErrFiscalDocumentFrozen)
	}
	if strings.TrimSpace(input.IssueOperationKey) == "" {
		return FrozenFiscalDocument{}, fmt.Errorf("%w: issue operation key is required", ErrFiscalDocumentFrozen)
	}
	if input.FinalizedBy == uuid.Nil {
		return FrozenFiscalDocument{}, fmt.Errorf("%w: finalizer is required", ErrFiscalDocumentFrozen)
	}
	if input.FrozenAt.IsZero() {
		return FrozenFiscalDocument{}, fmt.Errorf("%w: frozen time is required", ErrFiscalDocumentFrozen)
	}
	if err := d.FixProvider(input.ProviderKey, input.ConnectionID); err != nil {
		return FrozenFiscalDocument{}, err
	}
	if err := d.fixIntent(input.IntentKey, input.IssueOperationKey); err != nil {
		return FrozenFiscalDocument{}, err
	}
	d.FrozenAt = cloneTimePtr(&input.FrozenAt)
	d.FinalizedBy = cloneUUIDPtr(&input.FinalizedBy)
	d.State = FiscalDocumentStatePending
	d.UpdatedAt = input.FrozenAt
	return d.Frozen(), nil
}

// FixProvider fixes the provider and connection for the document.
func (d *FiscalDocument) FixProvider(providerKey string, connectionID uuid.UUID) error {
	if d == nil {
		return errors.New("fiscal document is nil")
	}
	if strings.TrimSpace(providerKey) == "" {
		return fmt.Errorf("%w: provider key is required", ErrFiscalDocumentFrozen)
	}
	if connectionID == uuid.Nil {
		return fmt.Errorf("%w: connection is required", ErrFiscalDocumentFrozen)
	}
	if d.ProviderKey != "" && d.ProviderKey != providerKey {
		return ErrFiscalDocumentFrozen
	}
	if d.ConnectionID != nil && *d.ConnectionID != connectionID {
		return ErrFiscalDocumentFrozen
	}
	d.ProviderKey = providerKey
	d.ConnectionID = cloneUUIDPtr(&connectionID)
	return nil
}

// SupersedeRejected links a new draft to a rejected intent and marks the rejected one superseded.
func (d *FiscalDocument) SupersedeRejected(rejected *FiscalDocument, actor FiscalActorRole, at time.Time) error {
	if d == nil || rejected == nil {
		return errors.New("fiscal document is nil")
	}
	if !actor.IsStaff() {
		return ErrFiscalDocumentRoleDenied
	}
	if d.State != FiscalDocumentStateDraft {
		return fmt.Errorf("%w: replacement must remain draft", ErrFiscalDocumentTransitionNotAllowed)
	}
	if rejected.State != FiscalDocumentStateRejected {
		return ErrFiscalDocumentSupersessionNotAllowed
	}
	if d.SourceInvoiceID != rejected.SourceInvoiceID || d.IntentSlot != rejected.IntentSlot || d.Kind != rejected.Kind {
		return ErrFiscalDocumentSupersessionNotAllowed
	}
	if rejected.SupersededAt != nil {
		return ErrFiscalDocumentSupersessionNotAllowed
	}
	d.SupersedesDocumentID = cloneUUIDPtr(&rejected.ID)
	rejected.SupersededAt = cloneTimePtr(&at)
	d.UpdatedAt = at
	rejected.UpdatedAt = at
	return nil
}

func (d *FiscalDocument) fixIntent(intentKey uuid.UUID, issueOperationKey string) error {
	if intentKey == uuid.Nil {
		return fmt.Errorf("%w: intent key is required", ErrFiscalDocumentFrozen)
	}
	if strings.TrimSpace(issueOperationKey) == "" {
		return fmt.Errorf("%w: issue operation key is required", ErrFiscalDocumentFrozen)
	}
	if d.IntentKey != nil && *d.IntentKey != intentKey {
		return ErrFiscalDocumentFrozen
	}
	if strings.TrimSpace(d.IssueOperationKey) != "" && d.IssueOperationKey != issueOperationKey {
		return ErrFiscalDocumentFrozen
	}
	d.IntentKey = cloneUUIDPtr(&intentKey)
	d.IssueOperationKey = issueOperationKey
	return nil
}

var fiscalDocumentTransitionAllowed = map[FiscalDocumentState]map[FiscalDocumentState]bool{
	FiscalDocumentStateDraft: {
		FiscalDocumentStatePending: true,
	},
	FiscalDocumentStatePending: {
		FiscalDocumentStateDispatching: true,
	},
	FiscalDocumentStateDispatching: {
		FiscalDocumentStateIssued:                 true,
		FiscalDocumentStateRejected:               true,
		FiscalDocumentStateConnectionActionNeeded: true,
		FiscalDocumentStateRetryableFailure:       true,
		FiscalDocumentStateOutcomeUnknown:         true,
	},
	FiscalDocumentStateRejected: {},
	FiscalDocumentStateConnectionActionNeeded: {
		FiscalDocumentStatePending: true,
	},
	FiscalDocumentStateRetryableFailure: {
		FiscalDocumentStatePending: true,
	},
	FiscalDocumentStateOutcomeUnknown: {
		FiscalDocumentStateIssued:           true,
		FiscalDocumentStateRejected:         true,
		FiscalDocumentStateRetryableFailure: true,
		FiscalDocumentStateOutcomeUnknown:   true,
	},
	FiscalDocumentStateIssued: {
		FiscalDocumentStateVoidPending: true,
	},
	FiscalDocumentStateVoidPending: {
		FiscalDocumentStateIssued:             true,
		FiscalDocumentStateVoidOutcomeUnknown: true,
		FiscalDocumentStateVoided:             true,
	},
	FiscalDocumentStateVoidOutcomeUnknown: {
		FiscalDocumentStateIssued:             true,
		FiscalDocumentStateVoidOutcomeUnknown: true,
		FiscalDocumentStateVoided:             true,
	},
	FiscalDocumentStateVoided: {},
}

func dedupeFiscalDocumentActions(actions []FiscalDocumentAction) []FiscalDocumentAction {
	seen := make(map[FiscalDocumentAction]bool, len(actions))
	result := make([]FiscalDocumentAction, 0, len(actions))
	for _, action := range actions {
		if seen[action] {
			continue
		}
		seen[action] = true
		result = append(result, action)
	}
	return result
}

func cloneUUIDPtr(src *uuid.UUID) *uuid.UUID {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}

func cloneTimePtr(src *time.Time) *time.Time {
	if src == nil {
		return nil
	}
	value := *src
	return &value
}
