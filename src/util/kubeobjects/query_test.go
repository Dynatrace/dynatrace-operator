package kubeobjects

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/src/api/scheme/fake"
	"testing"
)

func TestKubeQuery(t *testing.T) {
	fakeClient := fake.NewClient()
	_ = newKubeQuery(context.TODO(), fakeClient, fakeClient, configMapLog)
}
