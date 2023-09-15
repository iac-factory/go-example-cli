package checksum

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

func SHA256(filepath string) *string {
	f, e := os.Open(filepath)
	defer f.Close()
	if e != nil {
		panic(e)
	}

	h := sha256.New()
	if _, e := io.Copy(h, f); e != nil {
		panic(e)
	}

	sum := fmt.Sprintf("%x", h.Sum(nil))

	return &sum
}
