package testutils

import (
	"github.com/sergeii/practikum-go-gophermart/pkg/random"
)

func NewLuhnNumber(sz int) string {
	_ = random.Seed()
	if sz < 2 {
		panic("invalid size")
	}
	digits := make([]uint8, sz)
	for i := 0; i < sz; i++ {
		digits[i] = uint8(random.Int(1, 9))
	}
	digits[sz-1] = calcLuhnDigit(digits[:sz-1])
	// convert digits to string
	for i, digit := range digits {
		digits[i] = '0' + digit
	}
	return string(digits)
}

func calcLuhnDigit(digits []uint8) uint8 {
	luhn := 0
	odd := false
	for i := len(digits) - 1; i >= 0; i-- {
		digit := digits[i]
		if !odd {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		luhn += int(digit)
		odd = !odd
	}
	return uint8((luhn * 9) % 10)
}
