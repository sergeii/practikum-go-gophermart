package validation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sergeii/practikum-go-gophermart/pkg/validation"
)

func TestIsLuhnNumber(t *testing.T) {
	tests := []struct {
		num  string
		want bool
	}{
		{
			"79927398713",
			true,
		},
		{
			"79927398714",
			false,
		},
		{
			"4561261212345467",
			true,
		},
		{
			"49927398716",
			true,
		},
		{
			"foo",
			false,
		},
		{
			"79927398713foo",
			false,
		},
		{
			"",
			false,
		},
		{
			"0",
			true,
		},
		{
			"01",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.num, func(t *testing.T) {
			assert.Equal(t, tt.want, validation.CheckLuhnNumber(tt.num))
		})
	}
}
