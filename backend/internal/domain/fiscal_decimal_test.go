package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecimal_ParseStrictAndCanonical(t *testing.T) {
	t.Parallel()

	d, err := ParseDecimal("12.3400")
	require.NoError(t, err)
	assert.Equal(t, "12.34", d.String())
	assert.Equal(t, int32(4), d.Scale())

	negZero, err := ParseDecimal("-0.00")
	require.NoError(t, err)
	assert.True(t, negZero.IsZero())
	assert.Equal(t, "0", negZero.String())

	_, err = ParseDecimal("1e3")
	assert.Error(t, err)

	_, err = ParseDecimal("+1")
	assert.Error(t, err)

	var fromJSON Decimal
	require.NoError(t, json.Unmarshal([]byte(`"3.500"`), &fromJSON))
	assert.Equal(t, "3.5", fromJSON.String())

	err = json.Unmarshal([]byte(`1.25`), &fromJSON)
	assert.Error(t, err)
}

func TestDecimal_ExactArithmeticAndRounding(t *testing.T) {
	t.Parallel()

	product := MustParseDecimal("1.20").Mul(MustParseDecimal("2.5"))
	assert.True(t, product.Equal(MustParseDecimal("3")))

	halfUp, err := MustParseDecimal("1.005").Round(2, RoundingModeHalfUp)
	require.NoError(t, err)
	assert.Equal(t, "1.01", halfUp.String())

	halfEven, err := MustParseDecimal("1.005").Round(2, RoundingModeHalfEven)
	require.NoError(t, err)
	assert.Equal(t, "1", halfEven.String())

	negativeHalfUp, err := MustParseDecimal("-1.005").Round(2, RoundingModeHalfUp)
	require.NoError(t, err)
	assert.Equal(t, "-1.01", negativeHalfUp.String())
}
