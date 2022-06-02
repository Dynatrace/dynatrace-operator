package kubeobjects

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
)

func TestKubeQuery(t *testing.T) {
	fakeClient := fake.NewClient()
	_ = newKubeQuery(context.TODO(), fakeClient, fakeClient, log)
}
