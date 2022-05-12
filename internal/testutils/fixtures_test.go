package testutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sergeii/practikum-go-gophermart/internal/testutils"
	"github.com/sergeii/practikum-go-gophermart/pkg/validation"
)

func TestNewLuhnNumber(t *testing.T) {
	for n := 2; n <= 16; n++ {
		maybeLuhnNumber := testutils.NewLuhnNumber(n)
		require.Len(t, maybeLuhnNumber, n)
		assert.True(t, validation.CheckLuhnNumber(maybeLuhnNumber), maybeLuhnNumber)
	}
}
