package oneagent

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/require"
)

// TestGetProcessGroupingConfig_Integration is an integration test that connects to a real
// Dynatrace cluster. It is skipped by default unless DT_API_URL and DT_PAAS_TOKEN are set.
//
// Set the following environment variables before running:
//
//	DT_API_URL    - Dynatrace API base URL (e.g. https://<env-id>.live.dynatrace.com/api)
//	DT_API_TOKEN  - API token with the appropriate permissions
//	DT_CLUSTER_ID - (optional) Kubernetes cluster ID to scope the config
//
// Run with: go test -v -run TestGetProcessGroupingConfig_Integration ./pkg/clients/dynatrace/oneagent/
func TestGetProcessGroupingConfig_Integration(t *testing.T) {
	// TODO: This test is currently intended to be used by devs for trying the new API until it is used in
	//       the product. Reconsider if it should be kept, once the API is used in the product.
	apiURL := os.Getenv("DT_API_URL")
	paasToken := os.Getenv("DT_API_TOKEN")

	if apiURL == "" || paasToken == "" {
		t.Skip("Skipping integration test: DT_API_URL and DT_PAAS_TOKEN must be set")
	}

	clusterID := os.Getenv("DT_CLUSTER_ID")

	parsedURL, err := url.Parse(apiURL)
	require.NoError(t, err, "failed to parse DT_API_URL")

	apiClient := core.NewClient(core.Config{
		BaseURL:    parsedURL,
		HTTPClient: http.DefaultClient,
		APIToken:   paasToken,
	})

	client := NewClient(apiClient, "", "")

	var buf bytes.Buffer

	etag, err := client.GetProcessGroupingConfig(t.Context(), clusterID, "", &buf)
	require.NoError(t, err)

	t.Logf("Response size: %d bytes", buf.Len())
	t.Logf("ETag: %s", etag)

	require.NotZero(t, buf.Len(), "expected non-empty response body")

	t.Logf("Decoded response:\n%s", decodeCBOR(t, buf.Bytes()))

	// Test conditional request with ETag (should return 304)
	// Skip for now, something seems to still be fishy with eTag handling on cluster side
	if etag != "" {
		var buf2 bytes.Buffer

		etag2, err := client.GetProcessGroupingConfig(t.Context(), clusterID, etag, &buf2)
		if err != nil {
			require.ErrorIs(t, err, ErrNotModified)
			require.Equal(t, etag, etag2)
			require.Zero(t, buf2.Len(), "body should be empty on 304")

			t.Log("Second request returned 304 Not Modified as expected")
		} else {
			t.Log("Second request returned new content (config changed between requests)")
		}
	}
}

// decodeCBOR decodes raw CBOR bytes into a generic Go structure and returns its
// pretty-printed JSON representation for human-readable logging.
// This is a minimal, self-contained CBOR decoder that covers the most common
// types (unsigned/negative ints, booleans, null, byte/text strings, arrays, maps, floats).
func decodeCBOR(t *testing.T, data []byte) string {
	t.Helper()

	decoded, _, err := cborDecodeItem(data)
	require.NoError(t, err, "failed to decode CBOR response")

	jsonBytes, err := json.MarshalIndent(decoded, "", "  ")
	require.NoError(t, err, "failed to marshal decoded CBOR to JSON")

	return string(jsonBytes)
}

// cborDecodeItem decodes one CBOR data item from the front of data and returns
// the decoded value, the remaining unconsumed bytes, and any error.
func cborDecodeItem(data []byte) (any, []byte, error) {
	if len(data) == 0 {
		return nil, nil, errors.New("unexpected end of CBOR data")
	}

	major := data[0] >> 5
	additional := data[0] & 0x1f

	switch major {
	case 0: // unsigned integer
		return cborReadUint(data)
	case 1: // negative integer
		return cborDecodeNegInt(data)
	case 2: // byte string
		return cborDecodeByteString(data)
	case 3: // text string
		return cborDecodeTextString(data)
	case 4: // array
		return cborDecodeArray(data)
	case 5: // map
		return cborDecodeMap(data)
	case 7: // simple values and floats
		return cborDecodeSimple(additional, data)
	default:
		return nil, nil, fmt.Errorf("unsupported CBOR major type %d", major)
	}
}

func cborDecodeNegInt(data []byte) (any, []byte, error) {
	val, rest, err := cborReadUint(data)
	if err != nil {
		return nil, nil, err
	}

	return int64(-1) - int64(val), rest, nil //nolint:gosec
}

func cborDecodeByteString(data []byte) (any, []byte, error) {
	length, rest, err := cborReadUint(data)
	if err != nil {
		return nil, nil, err
	}

	if uint64(len(rest)) < length {
		return nil, nil, fmt.Errorf("CBOR byte string: need %d bytes, have %d", length, len(rest))
	}

	bs := make([]byte, length)
	copy(bs, rest[:length])

	return bs, rest[length:], nil
}

func cborDecodeTextString(data []byte) (any, []byte, error) {
	length, rest, err := cborReadUint(data)
	if err != nil {
		return nil, nil, err
	}

	if uint64(len(rest)) < length {
		return nil, nil, fmt.Errorf("CBOR text string: need %d bytes, have %d", length, len(rest))
	}

	return string(rest[:length]), rest[length:], nil
}

func cborDecodeArray(data []byte) (any, []byte, error) {
	count, rest, err := cborReadUint(data)
	if err != nil {
		return nil, nil, err
	}

	arr := make([]any, 0, count)

	for range count {
		var item any

		item, rest, err = cborDecodeItem(rest)
		if err != nil {
			return nil, nil, err
		}

		arr = append(arr, item)
	}

	return arr, rest, nil
}

func cborDecodeMap(data []byte) (any, []byte, error) {
	count, rest, err := cborReadUint(data)
	if err != nil {
		return nil, nil, err
	}

	m := make(map[string]any, count)

	for range count {
		var key, val any

		key, rest, err = cborDecodeItem(rest)
		if err != nil {
			return nil, nil, err
		}

		val, rest, err = cborDecodeItem(rest)
		if err != nil {
			return nil, nil, err
		}

		m[fmt.Sprint(key)] = val
	}

	return m, rest, nil
}

func cborDecodeSimple(additional byte, data []byte) (any, []byte, error) {
	switch additional {
	case 20: // false
		return false, data[1:], nil
	case 21: // true
		return true, data[1:], nil
	case 22: // null
		return nil, data[1:], nil
	case 25: // float16 — decode as float64
		if len(data) < 3 {
			return nil, nil, errors.New("CBOR float16: need 3 bytes")
		}

		return float64(math.Float32frombits(cborHalfToFloat32Bits(binary.BigEndian.Uint16(data[1:3])))), data[3:], nil
	case 26: // float32
		if len(data) < 5 {
			return nil, nil, errors.New("CBOR float32: need 5 bytes")
		}

		return float64(math.Float32frombits(binary.BigEndian.Uint32(data[1:5]))), data[5:], nil
	case 27: // float64
		if len(data) < 9 {
			return nil, nil, errors.New("CBOR float64: need 9 bytes")
		}

		return math.Float64frombits(binary.BigEndian.Uint64(data[1:9])), data[9:], nil
	default:
		return nil, nil, fmt.Errorf("unsupported CBOR simple value %d", additional)
	}
}

// cborReadUint reads the unsigned integer argument encoded after the major type byte.
// It returns the value and the remaining data after the argument bytes.
func cborReadUint(data []byte) (uint64, []byte, error) {
	additional := data[0] & 0x1f
	data = data[1:] // consume initial byte

	switch {
	case additional < 24:
		return uint64(additional), data, nil
	case additional == 24:
		if len(data) < 1 {
			return 0, nil, errors.New("CBOR uint: need 1 byte")
		}

		return uint64(data[0]), data[1:], nil
	case additional == 25:
		if len(data) < 2 {
			return 0, nil, errors.New("CBOR uint: need 2 bytes")
		}

		return uint64(binary.BigEndian.Uint16(data[:2])), data[2:], nil
	case additional == 26:
		if len(data) < 4 {
			return 0, nil, errors.New("CBOR uint: need 4 bytes")
		}

		return uint64(binary.BigEndian.Uint32(data[:4])), data[4:], nil
	case additional == 27:
		if len(data) < 8 {
			return 0, nil, errors.New("CBOR uint: need 8 bytes")
		}

		return binary.BigEndian.Uint64(data[:8]), data[8:], nil
	default:
		return 0, nil, fmt.Errorf("CBOR uint: unsupported additional info %d", additional)
	}
}

// cborHalfToFloat32Bits converts an IEEE 754 half-precision (16-bit) value to float32 bits.
func cborHalfToFloat32Bits(h uint16) uint32 {
	sign := uint32(h>>15) << 31
	exp := uint32(h>>10) & 0x1f
	mant := uint32(h) & 0x3ff

	switch exp {
	case 0: // subnormal
		if mant == 0 {
			return sign
		}

		for mant&0x400 == 0 {
			mant <<= 1
			exp--
		}

		exp++
		mant &= 0x3ff

		return sign | ((exp + 127 - 15) << 23) | (mant << 13)
	case 0x1f: // inf / nan
		return sign | (0xff << 23) | (mant << 13)
	default:
		return sign | ((exp + 127 - 15) << 23) | (mant << 13)
	}
}
