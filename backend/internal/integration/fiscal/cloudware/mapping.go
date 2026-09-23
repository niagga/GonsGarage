package cloudware

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FrozenDocumentInput is the adapter-local frozen snapshot projection used for mapping.
type FrozenDocumentInput struct {
	Kind              string            `json:"kind"`
	ExternalReference string            `json:"externalReference"`
	SeriesPrefix      string            `json:"seriesPrefix,omitempty"`
	SeriesID          string            `json:"seriesId,omitempty"`
	CustomerNIF       string            `json:"customerNif,omitempty"`
	CustomerName      string            `json:"customerName,omitempty"`
	Lines             []FrozenLineInput `json:"lines"`
	Finalize          bool              `json:"finalize,omitempty"`
	ReturnPDF         bool              `json:"returnPdf,omitempty"`
}

// FrozenLineInput is a manual fiscal line mapped to Cloudware v1 concepts.
type FrozenLineInput struct {
	Description   string `json:"description"`
	Quantity      string `json:"quantity"`
	UnitPrice     string `json:"unitPrice"`
	TaxCode       string `json:"taxCode"`
	TaxPercent    string `json:"taxPercent,omitempty"`
	ExemptionCode string `json:"exemptionCode,omitempty"`
	UnitReference string `json:"unitReference,omitempty"`
	ItemReference string `json:"itemReference,omitempty"`
}

// SalesDocumentRequest is the Cloudware-local DTO (not a generic fiscal API contract).
type SalesDocumentRequest struct {
	DocumentType      string              `json:"document_type"`
	ExternalReference string              `json:"external_reference,omitempty"`
	SeriesPrefix      string              `json:"series_prefix,omitempty"`
	SeriesID          string              `json:"series_id,omitempty"`
	CustomerNIF       string              `json:"customer_tax_registration_number,omitempty"`
	CustomerName      string              `json:"customer_name,omitempty"`
	Lines             []SalesDocumentLine `json:"lines"`
	Finalize          bool                `json:"finalize,omitempty"`
	ReturnPDF         bool                `json:"return_pdf,omitempty"`
	AssociatedReceipt bool                `json:"-"` // FR behavior flag; not a separate receipt API
}

// SalesDocumentLine is adapter-local.
type SalesDocumentLine struct {
	Description   string `json:"description"`
	Quantity      string `json:"quantity"`
	UnitPrice     string `json:"unit_price"`
	TaxCode       string `json:"tax_code,omitempty"`
	TaxPercent    string `json:"tax_percentage,omitempty"`
	ExemptionCode string `json:"exemption_reason,omitempty"`
	UnitReference string `json:"unit,omitempty"`
	ItemReference string `json:"item,omitempty"`
}

// MapFrozenDocument maps only evidenced FT/FR concepts.
func MapFrozenDocument(in FrozenDocumentInput) (SalesDocumentRequest, error) {
	kind := strings.ToUpper(strings.TrimSpace(in.Kind))
	switch kind {
	case "FT", "FR":
	default:
		return SalesDocumentRequest{}, fmt.Errorf("%w: %s", ErrUnsupportedDocumentKind, kind)
	}
	if err := ValidateWorkflow(WorkflowFTFRIssue); err != nil {
		return SalesDocumentRequest{}, err
	}
	if len(in.Lines) == 0 {
		return SalesDocumentRequest{}, fmt.Errorf("%w: lines required", ErrUnsupportedDocumentKind)
	}
	out := SalesDocumentRequest{
		DocumentType:      kind,
		ExternalReference: strings.TrimSpace(in.ExternalReference),
		SeriesPrefix:      strings.TrimSpace(in.SeriesPrefix),
		SeriesID:          strings.TrimSpace(in.SeriesID),
		CustomerNIF:       strings.TrimSpace(in.CustomerNIF),
		CustomerName:      strings.TrimSpace(in.CustomerName),
		Finalize:          in.Finalize,
		ReturnPDF:         in.ReturnPDF,
		AssociatedReceipt: kind == "FR",
		Lines:             make([]SalesDocumentLine, 0, len(in.Lines)),
	}
	for _, line := range in.Lines {
		out.Lines = append(out.Lines, SalesDocumentLine{
			Description:   strings.TrimSpace(line.Description),
			Quantity:      strings.TrimSpace(line.Quantity),
			UnitPrice:     strings.TrimSpace(line.UnitPrice),
			TaxCode:       strings.TrimSpace(line.TaxCode),
			TaxPercent:    strings.TrimSpace(line.TaxPercent),
			ExemptionCode: strings.TrimSpace(line.ExemptionCode),
			UnitReference: strings.TrimSpace(line.UnitReference),
			ItemReference: strings.TrimSpace(line.ItemReference),
		})
	}
	return out, nil
}

// BuildFinalizeRequest builds the finalize contract fixture body.
func BuildFinalizeRequest(providerDocumentID string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"id":       strings.TrimSpace(providerDocumentID),
		"finalize": true,
	})
}

// BuildVoidRequest builds the void contract fixture body.
func BuildVoidRequest(providerDocumentID, reason string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"id":     strings.TrimSpace(providerDocumentID),
		"void":   true,
		"reason": strings.TrimSpace(reason),
	})
}

// BuildPDFRequest builds the PDF materialization fixture body.
func BuildPDFRequest(providerDocumentID string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"id":         strings.TrimSpace(providerDocumentID),
		"return_pdf": true,
	})
}
