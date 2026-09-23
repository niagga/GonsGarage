package ports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrFiscalOAuthStateInvalid reports a missing or mismatched OAuth state.
	ErrFiscalOAuthStateInvalid = errors.New("fiscal oauth state is invalid")
	// ErrFiscalOAuthStateReused reports a one-time state that was already consumed.
	ErrFiscalOAuthStateReused = errors.New("fiscal oauth state was already used")
	// ErrFiscalOAuthStateExpired reports an expired authorization attempt.
	ErrFiscalOAuthStateExpired = errors.New("fiscal oauth state expired")
	// ErrFiscalOAuthUnavailable reports missing OAuth wiring.
	ErrFiscalOAuthUnavailable = errors.New("fiscal oauth is unavailable")
)

// FiscalOAuthAuthorization is the persisted hashed OAuth state row.
type FiscalOAuthAuthorization struct {
	ID           uuid.UUID
	ConnectionID uuid.UUID
	ActorID      uuid.UUID
	StateSHA256  string
	RedirectURI  string
	ExpiresAt    time.Time
	ConsumedAt   *time.Time
	CreatedAt    time.Time
}

// Clone returns a deep copy.
func (a FiscalOAuthAuthorization) Clone() FiscalOAuthAuthorization {
	clone := a
	if a.ConsumedAt != nil {
		t := *a.ConsumedAt
		clone.ConsumedAt = &t
	}
	return clone
}

// FiscalOAuthStore persists hashed OAuth authorizations.
type FiscalOAuthStore interface {
	SaveOAuthAuthorization(ctx context.Context, auth *FiscalOAuthAuthorization) error
	ConsumeOAuthAuthorization(ctx context.Context, stateSHA256 string, now time.Time) (*FiscalOAuthAuthorization, error)
}

// FiscalOAuthTokenSet is the server-side token envelope (never returned to clients).
type FiscalOAuthTokenSet struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	Scopes       []string
	Organization string
}

// FiscalOAuthTokenExchanger exchanges codes / refreshes tokens against the provider.
type FiscalOAuthTokenExchanger interface {
	ExchangeAuthorizationCode(ctx context.Context, code, redirectURI string) (FiscalOAuthTokenSet, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (FiscalOAuthTokenSet, error)
}

// FiscalOAuthStartRequest begins a manager/admin authorization-code flow.
type FiscalOAuthStartRequest struct {
	ScopeKey    string
	ProviderKey string
	RedirectURI string
}

// FiscalOAuthStartResult returns the one-time raw state only for the redirect URL.
type FiscalOAuthStartResult struct {
	Status           *FiscalConnectionStatus
	AuthorizationURL string
	RawState         string
	ExpiresAt        time.Time
}

// FiscalOAuthCallbackRequest completes the provider callback.
type FiscalOAuthCallbackRequest struct {
	State string
	Code  string
}

// FiscalEnablementGateView is a provider-neutral readiness gate projection.
type FiscalEnablementGateView struct {
	Name      string
	Status    string
	Guidance  string
	Rationale string
}

// FiscalReadinessReport is the safe readiness projection for managers/admins.
type FiscalReadinessReport struct {
	Ready                 bool
	Gates                 []FiscalEnablementGateView
	ATCommunicationStatus string
}
