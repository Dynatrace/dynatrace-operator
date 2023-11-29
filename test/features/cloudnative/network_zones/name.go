package network_zones

import (
	"crypto/rand"
)

const prefix = "op-e2e-"
const defaultName = "test-network-zone"

// Use a randomized network zone names, to avoid problems with improper or slow cleanup of network zones on DT cluster side.
// With randomized names a fresh start is guaranteed for each test run.
func getNetworkZoneName() string {
	length := 8

	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return prefix + defaultName
	}
	return prefix + encode(b)
}

func encode(randBytes []byte) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

	result := make([]rune, len(randBytes))
	for _, b := range randBytes {
		result = append(result, letterRunes[b%byte(len(letterRunes))])
	}
	return string(result)
}
