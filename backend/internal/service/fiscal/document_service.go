package fiscal

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

const defaultIntentSlot = "primary_sale"

// DocumentService is the invoice-scoped fiscalization application service.
type DocumentService struct {
	drafts       *DraftService
	finals       *FinalizationService
	artifacts    *ArtifactService
	repo         ports.FiscalRepository
	invoices     ports.InvoiceRepository
	users        ports.UserRepository
	artifactRepo ports.FiscalArtifactRepository
}

// NewDocumentService builds the fiscalization service used by HTTP handlers.
func NewDocumentService(
	drafts *DraftService,
	finals *FinalizationService,
	artifacts *ArtifactService,
	repo ports.FiscalRepository,
	invoices ports.InvoiceRepository,
	users ports.UserRepository,
	artifactRepo ports.FiscalArtifactRepository,
) *DocumentService {
	return &DocumentService{
		drafts: drafts, finals: finals, artifacts: artifacts,
		repo: repo, invoices: invoices, users: users, artifactRepo: artifactRepo,
	}
}

var _ ports.FiscalizationService = (*DocumentService)(nil)

func (s *DocumentService) GetProjection(ctx context.Context, actorID, invoiceID uuid.UUID) (*ports.FiscalizationProjection, error) {
	role, inv, err := s.authorizeInvoiceAccess(ctx, actorID, invoiceID)
	if err != nil {
		return nil, err
	}
	agg, err := s.repo.GetCurrentIntent(ctx, invoiceID, defaultIntentSlot)
	if err != nil {
		if errors.Is(err, ports.ErrFiscalDocumentNotFound) {
			return legacyProjection(inv), nil
		}
		return nil, err
	}
	return s.toProjection(ctx, role, inv.CustomerID == actorID, agg), nil
}

func (s *DocumentService) UpsertDraft(ctx context.Context, actorID, invoiceID uuid.UUID, draft ports.FiscalDraftCommand) (*ports.FiscalizationProjection, bool, error) {
	role, inv, err := s.authorizeInvoiceAccess(ctx, actorID, invoiceID)
	if err != nil {
		return nil, false, err
	}
	if !role.IsStaff() {
		return nil, false, domain.ErrUnauthorizedAccess
	}
	slot := strings.TrimSpace(draft.IntentSlot)
	if slot == "" {
		slot = defaultIntentSlot
	}
	snapshot := snapshotFromDraft(draft)
	agg, err := s.repo.GetCurrentIntent(ctx, invoiceID, slot)
	if err != nil && !errors.Is(err, ports.ErrFiscalDocumentNotFound) {
		return nil, false, err
	}
	created := false
	if errors.Is(err, ports.ErrFiscalDocumentNotFound) {
		kind := domain.DocumentKind(draft.Kind)
		agg, err = s.drafts.CreateDraft(ctx, actorID, CreateDraftRequest{
			SourceInvoiceID: invoiceID, IntentSlot: slot, Kind: kind, Snapshot: snapshot,
		})
		created = true
	} else {
		if agg.Document.State != domain.FiscalDocumentStateDraft {
			return nil, false, ports.ErrFiscalActionNotAllowed
		}
		expected := draft.ExpectedVersion
		if expected == 0 {
			expected = agg.Document.Version
		}
		agg, err = s.drafts.UpdateDraft(ctx, actorID, agg.Document.ID, expected, snapshot)
	}
	if err != nil {
		return nil, false, err
	}
	return s.toProjection(ctx, role, inv.CustomerID == actorID, agg), created, nil
}

func (s *DocumentService) DeleteDraft(ctx context.Context, actorID, invoiceID uuid.UUID, expectedVersion int64) error {
	role, _, err := s.authorizeInvoiceAccess(ctx, actorID, invoiceID)
	if err != nil {
		return err
	}
	if !role.IsStaff() {
		return domain.ErrUnauthorizedAccess
	}
	agg, err := s.repo.GetCurrentIntent(ctx, invoiceID, defaultIntentSlot)
	if err != nil {
		return err
	}
	return s.drafts.DeleteDraft(ctx, actorID, agg.Document.ID, expectedVersion)
}

func (s *DocumentService) Finalize(ctx context.Context, actorID, invoiceID uuid.UUID, cmd ports.FiscalFinalizeCommand) (*ports.FiscalizationProjection, error) {
	role, inv, err := s.authorizeInvoiceAccess(ctx, actorID, invoiceID)
	if err != nil {
		return nil, err
	}
	agg, err := s.repo.GetCurrentIntent(ctx, invoiceID, defaultIntentSlot)
	if err != nil {
		return nil, err
	}
	res, err := s.finals.Finalize(ctx, actorID, FinalizeRequest{
		DocumentID: agg.Document.ID, ExpectedVersion: cmd.ExpectedVersion,
		ProviderKey: cmd.ProviderKey, ConnectionID: cmd.ConnectionID,
		PolicyVersionID: cmd.PolicyVersionID, IssuerProfileID: cmd.IssuerProfileID,
	})
	if err != nil {
		return nil, err
	}
	fresh, err := s.repo.GetAggregate(ctx, res.Document.ID)
	if err != nil {
		return nil, err
	}
	return s.toProjection(ctx, role, inv.CustomerID == actorID, fresh), nil
}

func (s *DocumentService) Retry(ctx context.Context, actorID, invoiceID uuid.UUID, cmd ports.FiscalActionCommand) (*ports.FiscalizationProjection, error) {
	return s.runLegalAction(ctx, actorID, invoiceID, func(docID uuid.UUID) (*domain.FiscalDocument, error) {
		return s.finals.Retry(ctx, actorID, ActionRequest{DocumentID: docID, ExpectedVersion: cmd.ExpectedVersion})
	})
}

func (s *DocumentService) Reconcile(ctx context.Context, actorID, invoiceID uuid.UUID, cmd ports.FiscalActionCommand) (*ports.FiscalizationProjection, error) {
	return s.runLegalAction(ctx, actorID, invoiceID, func(docID uuid.UUID) (*domain.FiscalDocument, error) {
		return s.finals.Reconcile(ctx, actorID, ActionRequest{DocumentID: docID, ExpectedVersion: cmd.ExpectedVersion})
	})
}

func (s *DocumentService) Void(ctx context.Context, actorID, invoiceID uuid.UUID, cmd ports.FiscalActionCommand) (*ports.FiscalizationProjection, error) {
	return s.runLegalAction(ctx, actorID, invoiceID, func(docID uuid.UUID) (*domain.FiscalDocument, error) {
		return s.finals.Void(ctx, actorID, ActionRequest{DocumentID: docID, ExpectedVersion: cmd.ExpectedVersion})
	})
}

func (s *DocumentService) runLegalAction(ctx context.Context, actorID, invoiceID uuid.UUID, fn func(uuid.UUID) (*domain.FiscalDocument, error)) (*ports.FiscalizationProjection, error) {
	role, inv, err := s.authorizeInvoiceAccess(ctx, actorID, invoiceID)
	if err != nil {
		return nil, err
	}
	agg, err := s.repo.GetCurrentIntent(ctx, invoiceID, defaultIntentSlot)
	if err != nil {
		return nil, err
	}
	doc, err := fn(agg.Document.ID)
	if err != nil {
		return nil, err
	}
	fresh, err := s.repo.GetAggregate(ctx, doc.ID)
	if err != nil {
		return nil, err
	}
	return s.toProjection(ctx, role, inv.CustomerID == actorID, fresh), nil
}

func (s *DocumentService) Summaries(ctx context.Context, actorID uuid.UUID, invoiceIDs []uuid.UUID) ([]ports.FiscalizationSummary, error) {
	if len(invoiceIDs) == 0 {
		return []ports.FiscalizationSummary{}, nil
	}
	if len(invoiceIDs) > 100 {
		return nil, ports.ErrFiscalNotReady
	}
	u, err := s.users.GetByID(ctx, actorID)
	if err != nil || u == nil {
		return nil, domain.ErrUserNotFound
	}
	role := fiscalRoleFromUser(u)
	out := make([]ports.FiscalizationSummary, 0, len(invoiceIDs))
	for _, id := range invoiceIDs {
		inv, err := s.invoices.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if inv == nil {
			return nil, domain.ErrInvoiceNotFound
		}
		owns := inv.CustomerID == actorID
		if role == domain.FiscalActorRoleClient && !owns {
			return nil, domain.ErrInvoiceNotFound
		}
		if !role.IsStaff() && !owns {
			return nil, domain.ErrInvoiceNotFound
		}
		proj, err := s.GetProjection(ctx, actorID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, ports.FiscalizationSummary{
			InvoiceID: id.String(), Status: proj.Status, Lifecycle: proj.Lifecycle, AllowedActions: proj.AllowedActions,
		})
	}
	return out, nil
}

func (s *DocumentService) OpenArtifact(ctx context.Context, actorID, invoiceID, artifactID uuid.UUID, meta ports.FiscalArtifactAccessMeta) (*ports.FiscalArtifactStream, error) {
	role, inv, err := s.authorizeInvoiceAccess(ctx, actorID, invoiceID)
	if err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, ports.ErrFiscalArtifactUnavailable
	}
	dl, err := s.artifacts.OpenDownload(ctx, DownloadArtifactCommand{
		ArtifactID: artifactID, ActorID: actorID, ActorRole: role,
		CorrelationID: meta.CorrelationID, IPHash: meta.IPHash,
	})
	if err != nil {
		if errors.Is(err, ports.ErrFiscalArtifactDenied) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, err
	}
	_ = inv
	return &ports.FiscalArtifactStream{
		Body: dl.Body, MediaType: dl.MediaType, ByteSize: dl.ByteSize,
		Filename: dl.Filename, Headers: dl.Headers,
	}, nil
}

func (s *DocumentService) authorizeInvoiceAccess(ctx context.Context, actorID, invoiceID uuid.UUID) (domain.FiscalActorRole, *domain.Invoice, error) {
	u, err := s.users.GetByID(ctx, actorID)
	if err != nil {
		return "", nil, err
	}
	if u == nil {
		return "", nil, domain.ErrUserNotFound
	}
	role := fiscalRoleFromUser(u)
	inv, err := s.invoices.GetByID(ctx, invoiceID)
	if err != nil {
		return "", nil, err
	}
	if inv == nil {
		return "", nil, domain.ErrInvoiceNotFound
	}
	owns := inv.CustomerID == actorID
	if role == domain.FiscalActorRoleClient {
		if !owns {
			return "", nil, domain.ErrInvoiceNotFound
		}
		return role, inv, nil
	}
	if !role.IsStaff() {
		return "", nil, domain.ErrUnauthorizedAccess
	}
	return role, inv, nil
}

func (s *DocumentService) toProjection(ctx context.Context, role domain.FiscalActorRole, owns bool, agg *ports.FiscalDocumentAggregate) *ports.FiscalizationProjection {
	doc := agg.Document
	status := string(doc.State.Presentation())
	actions := make([]string, 0)
	for _, a := range doc.AllowedActions(role, owns) {
		actions = append(actions, string(a))
	}
	docID := doc.ID.String()
	proj := &ports.FiscalizationProjection{
		InvoiceID: doc.SourceInvoiceID.String(), DocumentID: &docID, Kind: string(doc.Kind),
		Lifecycle: string(doc.State), Status: status, Version: doc.Version,
		Currency: agg.Snapshot.Currency, PayableTotal: agg.Snapshot.PayableTotal,
		GrossTotal: agg.Snapshot.GrossTotal, TaxTotal: agg.Snapshot.TaxTotal,
		AllowedActions: actions, LastErrorCode: doc.LastErrorCode, LastErrorSafe: RedactSecrets(doc.LastErrorMessage),
	}
	if s.artifactRepo != nil {
		if art, err := s.artifactRepo.GetByDocumentID(ctx, doc.ID); err == nil && art != nil {
			proj.ArtifactStatus = string(art.Status)
			proj.ArtifactID = art.ID.String()
		}
	}
	if role == domain.FiscalActorRoleClient {
		if doc.State == domain.FiscalDocumentStateDraft {
			proj.Status = string(domain.FiscalPresentationStateUnavailable)
			proj.DocumentID = nil
			proj.Kind = ""
			proj.Lifecycle = ""
			proj.Version = 0
			proj.PayableTotal = ""
			proj.GrossTotal = ""
			proj.TaxTotal = ""
			proj.Currency = ""
			proj.AllowedActions = []string{}
			proj.ArtifactStatus = ""
			proj.ArtifactID = ""
			proj.LastErrorCode = ""
			proj.LastErrorSafe = ""
			return proj
		}
		filtered := make([]string, 0, 1)
		for _, a := range actions {
			if a == string(domain.FiscalDocumentActionView) {
				filtered = append(filtered, a)
			}
		}
		proj.AllowedActions = filtered
		proj.LastErrorCode = ""
		proj.LastErrorSafe = ""
	}
	return proj
}

func legacyProjection(inv *domain.Invoice) *ports.FiscalizationProjection {
	status := string(domain.FiscalPresentationStateLegacyUnfiscalized)
	if inv != nil && inv.FiscalEligibility == domain.FiscalEligibilityEligible {
		status = string(domain.FiscalPresentationStateUnavailable)
	}
	id := ""
	if inv != nil {
		id = inv.ID.String()
	}
	return &ports.FiscalizationProjection{
		InvoiceID: id, Status: status, AllowedActions: []string{},
	}
}

func snapshotFromDraft(draft ports.FiscalDraftCommand) ports.FiscalSnapshotRecord {
	lines := make([]ports.FiscalLineRecord, 0, len(draft.Lines))
	for _, line := range draft.Lines {
		lines = append(lines, ports.FiscalLineRecord{
			Position: line.Position, Description: line.Description, UnitCode: line.UnitCode,
			Quantity: line.Quantity, UnitPrice: line.UnitPrice, DiscountKind: line.DiscountKind,
			DiscountValue: line.DiscountValue, TaxTreatmentCode: line.TaxTreatmentCode, TaxRate: line.TaxRate,
			ExemptionCode: line.ExemptionCode, SourceType: line.SourceType, SourceID: line.SourceID,
		})
	}
	customer := json.RawMessage(draft.Customer)
	if len(customer) == 0 {
		customer = json.RawMessage(`{}`)
	}
	billing := json.RawMessage(draft.BillingAddress)
	if len(billing) == 0 {
		billing = json.RawMessage(`{}`)
	}
	payable := draft.DeclaredPayable
	if payable == "" {
		payable = "0"
	}
	return ports.FiscalSnapshotRecord{
		PolicyVersionID: nil, IssuerProfileID: draft.IssuerProfileID,
		Issuer: json.RawMessage(`{}`), Customer: customer, BillingAddress: billing,
		Currency: draft.Currency, GrossTotal: "0", DiscountTotal: "0", NetTotal: "0",
		TaxTotal: "0", RoundingAdj: "0", PayableTotal: payable, Lines: lines,
	}
}

func fiscalRoleFromUser(u *domain.User) domain.FiscalActorRole {
	switch u.Role {
	case domain.RoleEmployee:
		return domain.FiscalActorRoleEmployee
	case domain.RoleManager:
		return domain.FiscalActorRoleManager
	case domain.RoleAdmin:
		return domain.FiscalActorRoleAdmin
	case domain.RoleClient:
		return domain.FiscalActorRoleClient
	default:
		return domain.FiscalActorRoleClient
	}
}
