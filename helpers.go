package main

import (
	"crypto/rand"
	"math/big"
)

func randomHex(lenght int) string {
	hex := []rune("0123456789abcdef")
	b := make([]rune, lenght)
	for i := range b {
		r, err := rand.Int(rand.Reader, big.NewInt(16))
		if err != nil {
			return ""
		}
		b[i] = hex[r.Int64()]
	}
	return string(b)
}
