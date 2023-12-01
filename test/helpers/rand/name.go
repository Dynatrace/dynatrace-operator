package rand

import (
	"crypto/rand"
)

const defaultLength = 8

type nameOptions struct {
	length int
	prefix string
}

type Options func(*nameOptions)

func WithLength(length int) Options {
	return func(o *nameOptions) {
		o.length = length
	}
}

func WithPrefix(prefix string) Options {
	return func(o *nameOptions) {
		o.prefix = prefix
	}
}

// Use a randomized network zone names, to avoid problems with improper or slow cleanup of network zones on DT cluster side.
// With randomized names a fresh start is guaranteed for each test run.
func GetRandomName(opts ...Options) (string, error) {
	options := nameOptions{
		length: defaultLength,
	}

	for _, opt := range opts {
		opt(&options)
	}

	b := make([]byte, options.length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return options.prefix + encode(b)[:options.length], nil
}

func encode(randBytes []byte) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

	result := make([]rune, 0, len(randBytes))
	for _, b := range randBytes {
		result = append(result, letterRunes[b%byte(len(letterRunes))])
	}
	return string(result)
}
