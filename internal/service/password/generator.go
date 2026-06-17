package password

import (
	"crypto/rand"
	"math/big"
)

// Character pools for generated passwords. Ambiguous glyphs (0/O/1/l/I) are
// omitted so a temporary password can be read off an email without confusion.
const (
	genLower  = "abcdefghijkmnopqrstuvwxyz"
	genUpper  = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	genDigit  = "23456789"
	genSymbol = "!@#$%^&*-_"
)

// Generate returns a random password that satisfies DefaultPasswordPolicy
// (length >= MinLength, at least one uppercase, lowercase, and digit). A symbol
// is also included for strength. Uses crypto/rand.
func Generate() string {
	const length = 16
	all := genLower + genUpper + genDigit + genSymbol

	out := make([]byte, length)
	// Guarantee one character from each required class.
	out[0] = pick(genLower)
	out[1] = pick(genUpper)
	out[2] = pick(genDigit)
	out[3] = pick(genSymbol)
	for i := 4; i < length; i++ {
		out[i] = pick(all)
	}
	shuffle(out)
	return string(out)
}

func pick(s string) byte {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(s))))
	if err != nil {
		return s[0]
	}
	return s[n.Int64()]
}

func shuffle(b []byte) {
	for i := len(b) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			continue
		}
		k := j.Int64()
		b[i], b[k] = b[k], b[i]
	}
}
