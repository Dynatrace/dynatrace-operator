package kubesystem

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetUID(t *testing.T) {
	const testUID = types.UID("test-uid")

	fakeClient := fake.NewFakeClientWithScheme(
		scheme.Scheme,
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: Namespace,
				UID:  testUID,
			},
		},
	)
	uid, err := GetUID(fakeClient)

	assert.NoError(t, err)
	assert.NotEmpty(t, uid)
	assert.Equal(t, testUID, uid)
}
