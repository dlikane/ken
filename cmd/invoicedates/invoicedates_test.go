package main

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestToCurrencyString(t *testing.T) {
	tests := []struct {
		qty      decimal.Decimal
		price    decimal.Decimal
		expected string
	}{
		{decimal.NewFromFloat(10.5), decimal.NewFromFloat(70.85), "743.93"},
		{decimal.NewFromFloat(1.0), decimal.NewFromFloat(0.99), "0.99"},
	}

	for _, test := range tests {
		result := ToCurrencyString(test.qty, test.price)
		if result != test.expected {
			t.Errorf("ToCurrencyString(%s, %s) = %s; expected %s", test.qty, test.price, result, test.expected)
		}
	}
}
