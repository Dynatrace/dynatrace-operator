package routing

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/deprecated/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func init() {
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme.Scheme))
}

func TestNewReconiler(t *testing.T) {
	createDefaultReconciler(t)
}

func createDefaultReconciler(t *testing.T) *ReconcileRouting {
	log := logger.NewDTLogger()
	clt := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		Build()
	dtc := &dtclient.MockDynatraceClient{}
	instance := &v1alpha1.DynaKube{}
	imgVerProvider := func(img string, dockerConfig *dtversion.DockerConfig) (dtversion.ImageVersion, error) {
		return dtversion.ImageVersion{}, nil
	}

	r := NewReconciler(clt, scheme.Scheme, dtc, log, instance, imgVerProvider)
	require.NotNil(t, r)
	require.NotNil(t, r.client)
	require.NotNil(t, r.scheme)
	require.NotNil(t, r.dtc)
	require.NotNil(t, r.log)
	require.NotNil(t, r.instance)
	require.NotNil(t, r.imageVersionProvider)

	return r
}

func TestReconcile(t *testing.T) {
	r := createDefaultReconciler(t)
	update, err := r.Reconcile()

	assert.True(t, update)
	assert.NoError(t, err)

	statefulSet := &v1.StatefulSet{}
	err = r.client.Get(context.TODO(), client.ObjectKey{Name: r.instance.Name + StatefulSetSuffix, Namespace: r.instance.Namespace}, statefulSet)

	assert.NotNil(t, statefulSet)
	assert.NoError(t, err)
}
