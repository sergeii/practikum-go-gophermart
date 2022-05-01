package random

import (
	crand "crypto/rand"
	"encoding/binary"
	mrand "math/rand"
)

func Seed() error {
	var buf [8]byte
	if _, err := crand.Read(buf[:]); err != nil {
		return err
	}
	mrand.Seed(int64(binary.LittleEndian.Uint64(buf[:])))
	return nil
}

func String(length int, alphabet string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = alphabet[mrand.Int63()%int64(len(alphabet))] // nolint:gosec
	}
	return string(b)
}
