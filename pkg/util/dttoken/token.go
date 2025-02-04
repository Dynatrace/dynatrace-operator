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

func New(prefix string) (*Token, error) {
	public, err := generateRandom(publicPortionSize)
	if err != nil {
		return nil, err
	}

	private, err := generateRandom(privatePortionSize)
	if err != nil {
		return nil, err
	}

	return &Token{prefix: prefix, public: public, private: private}, nil
}

// generate base32 encoded random string using base32.StdEncoding
func generateRandom(size int) (string, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)

	if err != nil {
		return "", err
	}

	return base32.StdEncoding.EncodeToString(b)[:size], nil
}
