package mock

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// RenderMockPDF returns deterministic PDF bytes labeled as non-legal mock evidence.
func RenderMockPDF(operationKey, canonicalSHA256, documentKind string) ([]byte, string) {
	body := fmt.Sprintf(
		"BT /F1 18 Tf 72 720 Td (%s) Tj 0 -28 Td (operation: %s) Tj 0 -24 Td (kind: %s) Tj 0 -24 Td (sha: %s) Tj ET",
		pdfEscape(MockPDFLabel),
		pdfEscape(operationKey),
		pdfEscape(documentKind),
		pdfEscape(canonicalSHA256),
	)
	objects := []string{
		"1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj\n",
		"2 0 obj<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj\n",
		"3 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources<< /Font<< /F1 5 0 R >> >> >>endobj\n",
		fmt.Sprintf("4 0 obj<< /Length %d >>stream\n%s\nendstream\nendobj\n", len(body), body),
		"5 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj\n",
	}

	var b strings.Builder
	b.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	offsets := make([]int, 0, len(objects)+1)
	offsets = append(offsets, 0)
	for _, obj := range objects {
		offsets = append(offsets, b.Len())
		b.WriteString(obj)
	}
	xrefStart := b.Len()
	b.WriteString("xref\n0 ")
	b.WriteString(strconv.Itoa(len(offsets)))
	b.WriteString("\n0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	b.WriteString("trailer<< /Size ")
	b.WriteString(strconv.Itoa(len(offsets)))
	b.WriteString(" /Root 1 0 R >>\nstartxref\n")
	b.WriteString(strconv.Itoa(xrefStart))
	b.WriteString("\n%%EOF\n")

	content := []byte(b.String())
	sum := sha256.Sum256(content)
	return content, hex.EncodeToString(sum[:])
}

func pdfEscape(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)")
	return replacer.Replace(value)
}
