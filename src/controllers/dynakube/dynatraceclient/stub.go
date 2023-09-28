package dynatraceclient

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StubBuilder can be used for unit tests where a full builder.go implementation is not needed
type StubBuilder struct {
	DynatraceClient dtclient.Client
	Err             error
}

func (stubBuilder StubBuilder) SetContext(context.Context) Builder {
	return stubBuilder
}

func (stubBuilder StubBuilder) SetDynakube(dynatracev1beta1.DynaKube) Builder {
	return stubBuilder
}

func (stubBuilder StubBuilder) SetTokens(token.Tokens) Builder {
	return stubBuilder
}

func (stubBuilder StubBuilder) LastApiProbeTimestamp() *metav1.Time {
	return nil
}

func (stubBuilder StubBuilder) Build() (dtclient.Client, error) {
	return stubBuilder.DynatraceClient, stubBuilder.Err
}

func (stubBuilder StubBuilder) BuildWithTokenVerification(*dynatracev1beta1.DynaKubeStatus) (dtclient.Client, error) {
	return stubBuilder.DynatraceClient, stubBuilder.Err
}
