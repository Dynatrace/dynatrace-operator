package otlpconfig

import (
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log = logd.Get().WithName("otlp-config")
)

// SecretGenerator manages the otlp secret generation for the user namespaces.
type SecretGenerator struct {
	client       client.Client
	dtClient     dtclient.Client
	apiReader    client.Reader
	timeProvider *timeprovider.Provider
	secrets      k8ssecret.QueryObject
}

func NewSecretGenerator(client client.Client, apiReader client.Reader, dtClient dtclient.Client) *SecretGenerator {
	return &SecretGenerator{
		client:       client,
		dtClient:     dtClient,
		apiReader:    apiReader,
		timeProvider: timeprovider.New(),
		secrets:      k8ssecret.Query(client, apiReader, log),
	}
}

func (s SecretGenerator) GenerateForDynakube() {

}
