package k8ssecret

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetDataFromSecretName(t *testing.T) {
	fakeClient := fake.NewClient()
	fakeClient.Create(context.Background(), getTestSecret())
	ctx := context.Background()

	t.Run("get secret data", func(t *testing.T) {
		data, _ := GetDataFromSecretName(ctx, fakeClient, types.NamespacedName{Name: testSecretName, Namespace: testNamespace}, testSecretDataKey, logd.Logger{})
		assert.Equal(t, string(dataValue), data)
	})
}
