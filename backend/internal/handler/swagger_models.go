package handler

// Tipos exportados solo para documentación OpenAPI (swag).

// SwaggerLoginOK respuesta de POST /auth/login.
type SwaggerLoginOK struct {
	Message string `json:"message"`
	Token   string `json:"token"`
}

// SwaggerRegisterUser usuario creado (subconjunto estable para el contrato JSON).
type SwaggerRegisterUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Role      string `json:"role"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// SwaggerRegisterOK respuesta de POST /auth/register.
type SwaggerRegisterOK struct {
	Message string              `json:"message"`
	User    SwaggerRegisterUser `json:"user"`
}

// SwaggerMeUser perfil en GET /auth/me.
type SwaggerMeUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Role      string `json:"role"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// SwaggerMeOK envoltorio { "user": ... }.
type SwaggerMeOK struct {
	User SwaggerMeUser `json:"user"`
}

// SwaggerProvisionUserOK respuesta de POST /admin/users (aprovisionamento staff).
type SwaggerProvisionUserOK struct {
	User SwaggerMeUser `json:"user"`
}

// SwaggerMessage error o mensagem genérico.
type SwaggerMessage struct {
	Error   string `json:"error,omitempty"`
	Details string `json:"details,omitempty"`
	Message string `json:"message,omitempty"`
}

// FiscalizationProjectionResponse documents the provider-neutral fiscalization projection.
type FiscalizationProjectionResponse struct {
	InvoiceID      string   `json:"invoiceId"`
	DocumentID     *string  `json:"documentId,omitempty"`
	Kind           string   `json:"kind,omitempty"`
	Lifecycle      string   `json:"lifecycle,omitempty"`
	Status         string   `json:"status"`
	Version        int64    `json:"version,omitempty"`
	Currency       string   `json:"currency,omitempty"`
	PayableTotal   string   `json:"payableTotal,omitempty"`
	GrossTotal     string   `json:"grossTotal,omitempty"`
	TaxTotal       string   `json:"taxTotal,omitempty"`
	AllowedActions []string `json:"allowedActions"`
	ArtifactStatus string   `json:"artifactStatus,omitempty"`
	LastErrorCode  string   `json:"lastErrorCode,omitempty"`
	LastErrorSafe  string   `json:"lastErrorMessage,omitempty"`
	Readiness      []string `json:"readinessIssues,omitempty"`
}

// FiscalizationSummariesResponse documents GET /invoices/fiscalization-summaries.
type FiscalizationSummariesResponse struct {
	Items []FiscalizationSummaryItem `json:"items"`
}

// FiscalizationSummaryItem is one batch summary badge.
type FiscalizationSummaryItem struct {
	InvoiceID      string   `json:"invoiceId"`
	Status         string   `json:"status"`
	Lifecycle      string   `json:"lifecycle,omitempty"`
	AllowedActions []string `json:"allowedActions,omitempty"`
}

// FiscalFieldError documents 422 field failures on fiscal endpoints.
type FiscalFieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// FiscalValidationError documents fiscal validation failures.
type FiscalValidationError struct {
	Error  string             `json:"error"`
	Fields []FiscalFieldError `json:"fields"`
}

// RepairAPIModel documenta el JSON de `domain.Repair` en respuestas Gin (POST/GET/PUT /repairs).
type RepairAPIModel struct {
	ID           string  `json:"id"`
	CarID        string  `json:"car_id"`
	TechnicianID string  `json:"technician_id"`
	Description  string  `json:"description"`
	Status       string  `json:"status"`
	Cost         float64 `json:"cost"`
	StartedAt    *string `json:"started_at,omitempty"`
	CompletedAt  *string `json:"completed_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	DeletedAt    *string `json:"deleted_at,omitempty"`
}
