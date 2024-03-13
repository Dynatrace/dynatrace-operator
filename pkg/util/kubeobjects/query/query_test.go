package query

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var configMapLog = logd.Get().WithName("test-configMap")

func TestKubeQuery(t *testing.T) {
	fakeClient := fake.NewClient()
	_ = New(context.TODO(), fakeClient, fakeClient, configMapLog)
}
