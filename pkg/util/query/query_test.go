package query

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/consts"
)

func TestKubeQuery(t *testing.T) {
	fakeClient := fake.NewClient()
	_ = New(context.TODO(), fakeClient, fakeClient, consts.ConfigMapLog)
}
