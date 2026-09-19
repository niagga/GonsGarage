package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

var (
	// ErrInvalidDecimalString is returned when a decimal string is not strict base-10.
	ErrInvalidDecimalString = errors.New("invalid decimal string")
	// ErrInvalidRoundingMode is returned when a rounding mode is unknown.
	ErrInvalidRoundingMode = errors.New("invalid rounding mode")
)

var strictDecimalPattern = regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)

var halfDecimal = decimal.New(5, -1)

// Decimal is a strict exact decimal wrapper.
//
// It accepts only non-exponent decimal strings, never uses float64, and
// normalizes negative zero to plain zero.
type Decimal struct {
	value decimal.Decimal
}

// ParseDecimal parses a strict base-10 decimal string.
func ParseDecimal(raw string) (Decimal, error) {
	if raw == "" || !strictDecimalPattern.MatchString(raw) {
		return Decimal{}, fmt.Errorf("%w: %q", ErrInvalidDecimalString, raw)
	}

	parsed, err := decimal.NewFromString(raw)
	if err != nil {
		return Decimal{}, fmt.Errorf("%w: %q", ErrInvalidDecimalString, raw)
	}
	if parsed.Equal(decimal.Zero) {
		return Decimal{value: decimal.Zero}, nil
	}
	return Decimal{value: parsed}, nil
}

// MustParseDecimal parses a decimal string and panics on error.
func MustParseDecimal(raw string) Decimal {
	d, err := ParseDecimal(raw)
	if err != nil {
		panic(err)
	}
	return d
}

// ZeroDecimal returns the canonical zero value.
func ZeroDecimal() Decimal {
	return Decimal{value: decimal.Zero}
}

// NewDecimalFromInt64 constructs a decimal from an integer.
func NewDecimalFromInt64(value int64) Decimal {
	return Decimal{value: decimal.NewFromInt(value)}
}

// String returns the canonical non-exponent string form.
func (d Decimal) String() string {
	if d.value.Equal(decimal.Zero) {
		return "0"
	}

	text := d.value.String()
	if text == "-0" {
		return "0"
	}
	return strings.TrimSpace(text)
}

// Scale returns the number of digits after the decimal point.
func (d Decimal) Scale() int32 {
	exponent := d.value.Exponent()
	if exponent < 0 {
		return -exponent
	}
	return 0
}

// Sign reports the sign of the decimal.
func (d Decimal) Sign() int {
	return d.value.Sign()
}

// IsZero reports whether the decimal equals zero.
func (d Decimal) IsZero() bool {
	return d.value.Equal(decimal.Zero)
}

// Abs returns the absolute value.
func (d Decimal) Abs() Decimal {
	return Decimal{value: d.value.Abs()}
}

// Add returns the exact sum.
func (d Decimal) Add(other Decimal) Decimal {
	return Decimal{value: d.value.Add(other.value)}
}

// Sub returns the exact difference.
func (d Decimal) Sub(other Decimal) Decimal {
	return Decimal{value: d.value.Sub(other.value)}
}

// Mul returns the exact product.
func (d Decimal) Mul(other Decimal) Decimal {
	return Decimal{value: d.value.Mul(other.value)}
}

// Div returns the exact quotient.
func (d Decimal) Div(other Decimal) Decimal {
	return Decimal{value: d.value.Div(other.value)}
}

// Cmp compares two decimals.
func (d Decimal) Cmp(other Decimal) int {
	return d.value.Cmp(other.value)
}

// Equal reports whether two decimals are numerically equal.
func (d Decimal) Equal(other Decimal) bool {
	return d.Cmp(other) == 0
}

// Shift shifts the decimal exponent by the supplied amount.
func (d Decimal) Shift(places int32) Decimal {
	return Decimal{value: d.value.Shift(places)}
}

// Truncate truncates the decimal to the requested number of fractional digits.
func (d Decimal) Truncate(places int32) Decimal {
	return Decimal{value: d.value.Truncate(places)}
}

// Round rounds the decimal to the requested scale using the supplied mode.
func (d Decimal) Round(scale int32, mode RoundingMode) (Decimal, error) {
	if scale < 0 {
		return Decimal{}, fmt.Errorf("%w: negative scale %d", ErrInvalidDecimalString, scale)
	}
	if !mode.IsValid() {
		return Decimal{}, fmt.Errorf("%w: %q", ErrInvalidRoundingMode, mode)
	}
	if d.IsZero() {
		return ZeroDecimal(), nil
	}

	shifted := d.value.Shift(scale)
	integerPart := shifted.Truncate(0)
	remainder := shifted.Sub(integerPart)
	if remainder.IsZero() {
		return Decimal{value: integerPart.Shift(-scale)}, nil
	}

	sign := shifted.Sign()
	adjust := false
	switch mode {
	case RoundingModeDown:
		adjust = false
	case RoundingModeUp:
		adjust = true
	case RoundingModeFloor:
		adjust = sign < 0
	case RoundingModeCeiling:
		adjust = sign > 0
	case RoundingModeHalfUp:
		adjust = remainder.Abs().Cmp(halfDecimal) >= 0
	case RoundingModeHalfEven:
		comparison := remainder.Abs().Cmp(halfDecimal)
		if comparison > 0 {
			adjust = true
		} else if comparison == 0 {
			adjust = integerPart.Abs().Coefficient().Bit(0) == 1
		}
	}

	if adjust {
		if sign > 0 {
			integerPart = integerPart.Add(decimal.NewFromInt(1))
		} else {
			integerPart = integerPart.Sub(decimal.NewFromInt(1))
		}
	}

	return Decimal{value: integerPart.Shift(-scale)}, nil
}

// MarshalJSON writes the canonical string representation.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON accepts only strict decimal strings.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("%w: decimal JSON must be a string", ErrInvalidDecimalString)
	}

	parsed, err := ParseDecimal(raw)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// MarshalText writes the canonical string representation.
func (d Decimal) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText accepts only strict decimal strings.
func (d *Decimal) UnmarshalText(text []byte) error {
	parsed, err := ParseDecimal(string(text))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
