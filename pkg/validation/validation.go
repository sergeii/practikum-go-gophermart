package validation

func CheckLuhnNumber(num string) bool {
	size := len(num)
	if size == 0 {
		return false
	}
	// make sure that all characters in the string are digits
	for _, n := range num {
		if n < '0' || n > '9' {
			// not a digit, cannot validate
			return false
		}
	}
	sum := num[size-1] - '0'
	parity := (size - 2) % 2
	for i := 0; i <= size-2; i++ {
		digit := num[i] - '0'
		if i%2 == parity {
			digit *= 2
		}
		if digit > 9 {
			digit -= 9
		}
		sum += digit
	}
	return sum%10 == 0
}
