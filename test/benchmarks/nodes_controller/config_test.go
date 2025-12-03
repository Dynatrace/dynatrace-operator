package nodes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace = "dynatrace"
	testAPIToken  = "test-api-token"
)

// benchmarkConfig holds the configurable parameters for the benchmark
type benchmarkConfig struct {
	// Number of nodes to simulate in the benchmark
	NumNodes int
	// Number of Dynakube instances to simulate in the benchmark
	NumDynakubes int
	// Number of host entities (in the Dynatrace environment) to simulate in the benchmark
	NumEntities int
}

// SetupDTServerMock sets up a mock Dynatrace server that responds to host entity requests
// It is not a dtclient Mock, but a real HTTP server that responds to requests made by the dtclient.
// This allows for more realistic benchmarking of the NodesController's interaction with the Dynatrace API.
func (bc benchmarkConfig) SetupDTServerMock(b *testing.B) *httptest.Server {
	b.Helper()

	mockHostEntityAPI := func(expectedHostInfo []dtclient.HostInfoResponse) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.FormValue("Api-Token") == "" && r.Header.Get("Authorization") == "" {
				b.Fatal()
			}

			switch r.URL.Path {
			case "/v1/entity/infrastructure/hosts":
				getResponseBytes, err := json.Marshal(expectedHostInfo)
				if err != nil {
					return
				}

				w.WriteHeader(http.StatusOK)
				w.Write(getResponseBytes)

			case "/v1/events":
				w.WriteHeader(http.StatusOK)
			default:
				b.Fatal()
			}
		}
	}

	responses := []dtclient.HostInfoResponse{}
	for i := range bc.NumEntities {
		responses = append(responses, dtclient.HostInfoResponse{
			EntityID:    generateEntityID(i),
			IPAddresses: []string{generateNodeIP(i)},
		})
	}

	return httptest.NewServer(mockHostEntityAPI(responses))
}

// SetupDKs creates the specified number of Dynakube instances in the test namespace.
// Also populates each Dynakube with the specified number of node instances.
//   - This is just for simplicity; the instances would be populated by the Dynatrace Operator in a real scenario.
func (bc benchmarkConfig) SetupDKs(b *testing.B, clt client.Client, dtURL string) {
	b.Helper()

	require.NoError(b, clt.Create(b.Context(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}))

	for i := range bc.NumDynakubes {
		// Create secret first
		createSecret(b, clt, i)

		instances := make(map[string]oneagent.Instance)
		for j := range bc.NumNodes {
			nodeName := generateNodeName(j)
			instances[nodeName] = oneagent.Instance{
				IPAddress: generateNodeIP(i),
			}
		}

		createDynakube(b, clt, dtURL, i, instances)
	}
}

func (bc benchmarkConfig) SetupNodes(b *testing.B, clt client.Client) {
	for i := range bc.NumNodes {
		createNode(b, clt, i)
	}
}

func (bc benchmarkConfig) RemoveNodes(b *testing.B, clt client.Client) {
	for i := range bc.NumNodes {
		clt.Delete(b.Context(), genNode(i))
	}
}

func (bc benchmarkConfig) ReportMetrics(b *testing.B) {
	b.ReportMetric(float64(bc.NumNodes), "nodes")
	b.ReportMetric(float64(bc.NumDynakubes), "dynakubes")
	b.ReportMetric(float64(bc.NumEntities), "host-entities")
}
