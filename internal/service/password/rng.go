package password

import (
	"crypto/rand"
	"io"
)

var rng = rand.Reader

func SetRandomReader(r io.Reader) {
	rng = r
}
