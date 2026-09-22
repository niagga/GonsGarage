package fiscal

import (
	"context"
	"fmt"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

// Dispatcher orquestates the interaction between the Outbox/Repository and the Cloudware Adapter.
type Dispatcher struct {
	repo      ports.FiscalRepository
	provider  ports.FiscalProvider
	evaluator *CloudwareEnablementEvaluator
}

// NewDispatcher builds a new fiscal dispatcher.
func NewDispatcher(repo ports.FiscalRepository, provider ports.FiscalProvider, evaluator *CloudwareEnablementEvaluator) *Dispatcher {
	return &Dispatcher{repo: repo, provider: provider, evaluator: evaluator}
}

// ProcessNext fetches the next pending fiscal document and dispatches it.
func (d *Dispatcher) ProcessNext(ctx context.Context) error {
	doc, err := d.repo.GetNextPending(ctx)
	if err != nil {
		if err == ports.ErrFiscalDocumentNotFound {
			return nil // No pending documents
		}
		return err
	}

	// 1. Evaluate CloudwareEnablementEvaluator
	if !d.evaluator.Evaluate(doc) {
		return fmt.Errorf("production Cloudware mutation disabled for document %s", doc.ID)
	}

	// Fetch full aggregate for snapshot data
	agg, err := d.repo.GetAggregate(ctx, doc.ID)
	if err != nil {
		return err
	}

	// 2. Dispatch to provider based on document state
	switch doc.State {
	case domain.FiscalDocumentStatePending:
		return d.issueDocument(ctx, agg)
	case domain.FiscalDocumentStateVoidPending:
		return fmt.Errorf("void not implemented yet for %s", doc.ID)
	default:
		return nil
	}
}

func (d *Dispatcher) issueDocument(ctx context.Context, agg *ports.FiscalDocumentAggregate) error {
	// 1. Prepare Request
	req := ports.FiscalIssueRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: *agg.Document.ConnectionID,
			ProviderKey:  agg.Document.ProviderKey,
			OperationKey: agg.Document.IssueOperationKey,
		},
		DocumentID: agg.Document.ID,
		Payload:    agg.Snapshot.CanonicalBytes,
	}

	// 2. Call Provider
	_, err := d.provider.Issue(ctx, req)
	if err != nil {
		// Aquí deberíamos manejar el error y quizás encolar una transición a RetryableFailure
		return fmt.Errorf("provider issue failed: %w", err)
	}

	// 3. Update state to Issued using EnqueueAction
	_, err = d.repo.EnqueueAction(ctx, ports.EnqueueActionCommand{
		DocumentID:      agg.Document.ID,
		ExpectedVersion: agg.Document.Version,
		ActorID:         agg.Document.CreatedBy, // O el actor del sistema
		EventType:       "issue_success",
		OperationKey:    agg.Document.IssueOperationKey,
		NextState:       domain.FiscalDocumentStateIssued,
	})
	if err != nil {
		return fmt.Errorf("failed to update state to issued: %w", err)
	}

	return nil
}
