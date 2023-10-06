package status

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testUUID = "test-uuid"
)

func TestSetDynakubeStatus(t *testing.T) {
	t.Run(`set status`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{}
		clt := fake.NewClient(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUUID,
			},
		})
		err := SetDynakubeStatus(instance, clt)

		assert.NoError(t, err)
		assert.Equal(t, testUUID, instance.Status.KubeSystemUUID)
	})
	t.Run(`error querying kube system uid`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{}
		clt := fake.NewClient()

		err := SetDynakubeStatus(instance, clt)
		assert.EqualError(t, err, "namespaces \"kube-system\" not found")
	})
}
