package valueobject_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/pricing/constant"
	"platform-service/internal/domain/pricing/valueobject"
	"platform-service/internal/testhelper"
)

func TestNewPrice(t *testing.T) {
	valid := map[string]string{
		"0": "0.00", "150000": "150000.00", "150000.5": "150000.50", " 99.99 ": "99.99",
		"9999999999999999.99": "9999999999999999.99",
	}
	for in, want := range valid {
		p, err := valueobject.NewPrice(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, p.String())
	}
	for _, in := range []string{"", "-1", "1.234", "01", "1e5", "abc", "10000000000000000", ".5"} {
		_, err := valueobject.NewPrice(in)
		testhelper.AssertDomainError(t, err, constant.CodePriceInvalid)
	}
}

func TestNewCurrency(t *testing.T) {
	c, err := valueobject.NewCurrency("")
	require.NoError(t, err)
	assert.Equal(t, "IDR", c.String())
	c, err = valueobject.NewCurrency(" usd ")
	require.NoError(t, err)
	assert.Equal(t, "USD", c.String())
	_, err = valueobject.NewCurrency("RP")
	testhelper.AssertDomainError(t, err, constant.CodeCurrencyInvalid)
}

func TestNewBillingCycle(t *testing.T) {
	c, err := valueobject.NewBillingCycle("")
	require.NoError(t, err)
	assert.Equal(t, valueobject.BillingCycleMonthly, c)
	_, err = valueobject.NewBillingCycle("weekly")
	testhelper.AssertDomainError(t, err, constant.CodeBillingCycleInvalid)
}

func TestParsePriceJSON(t *testing.T) {
	cases := map[string]string{`150000`: "150000", `"99.50"`: "99.50", ``: "", `null`: "", ` 1e5 `: "1e5"}
	for in, want := range cases {
		got, err := valueobject.ParsePriceJSON([]byte(in))
		require.NoError(t, err, in)
		assert.Equal(t, want, got)
	}
	_, err := valueobject.ParsePriceJSON([]byte(`"tidak ditutup`))
	testhelper.AssertDomainError(t, err, constant.CodePriceInvalid)
}
