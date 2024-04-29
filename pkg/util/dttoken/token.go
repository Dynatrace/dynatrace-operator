package dttoken

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
)

const (
	publicPortionSize  = 24
	privatePortionSize = 64
)

// Token represents <prefix>.<24-character-public-portion>.<64-character-private-portion>
type Token struct {
	prefix  string
	public  string
	private string
}

func (t Token) String() string {
	return fmt.Sprintf("%s.%s.%s", t.prefix, t.public, t.private)
}

func New(prefix string) *Token {
	return &Token{prefix: prefix, public: generateRandom(publicPortionSize), private: generateRandom(privatePortionSize)}
}

// generate base32 encoded random string using base32.StdEncoding
func generateRandom(size int) string {
	b := make([]byte, size)
	_, err := rand.Read(b)

	if err != nil {
		return ""
	}

	return base32.StdEncoding.EncodeToString(b)[:size]
}
