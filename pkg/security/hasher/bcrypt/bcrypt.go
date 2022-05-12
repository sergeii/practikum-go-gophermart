package bcrypt

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type Hasher struct{}

func New() Hasher {
	return Hasher{}
}

// Hash hashes a plaintext password using the Go's bcrypt package with the default cost
func (h Hasher) Hash(plainPassword string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// Check compares a plaintext password with its possible hashed equivalent
// Returns the result of the comparison
func (h Hasher) Check(plainPassword, hashedPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
