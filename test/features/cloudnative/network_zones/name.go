package network_zones

import (
	"math/rand"
)

const prefix = "op-e2e-"

// Use a randomized network zone names, to avoid problems with improper or slow cleanup of network zones on DT cluster side.
// With randomized names a fresh start is guaranteed for each test run.
func getNetworkZoneName() string {
	length := 8
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return prefix + string(b)
}
