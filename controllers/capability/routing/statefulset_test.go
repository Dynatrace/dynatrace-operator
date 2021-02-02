package routing

import (
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
)

func TestNewStatefulSetBuilder(t *testing.T) {
	stsBuilder := newStatefulSetProperties(&v1alpha1.DynaKube{})
	assert.NotNil(t, stsBuilder)
	assert.NotNil(t, stsBuilder.instance)
}

func TestStatefulSetBuilder_Build(t *testing.T) {
	instance := &v1alpha1.DynaKube{
		ObjectMeta: v1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		}}
	t.Run(`is not nil`, func(t *testing.T) {
		sts, err := createStatefulSet(newStatefulSetProperties(instance))
		assert.NoError(t, err)
		assert.NotNil(t, sts)
	})
	t.Run(`name is instance name plus correct suffix`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance))
		assert.Equal(t, instance.Name+StatefulSetSuffix, sts.Name)
	})
	t.Run(`namespace is instance namespace`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance))
		assert.Equal(t, instance.Namespace, sts.Namespace)
	})
}
