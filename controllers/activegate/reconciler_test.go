package activegate

import (
	"context"
	"reflect"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/customproperties"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestNewReconiler(t *testing.T) {
	createDefaultReconciler(t)
}

func createDefaultReconciler(t *testing.T) *Reconciler {
	log := logger.NewDTLogger()
	clt := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUID,
			},
		}).
		Build()
	dtc := &dtclient.MockDynatraceClient{}
	instance := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
		}}
	imgVerProvider := func(img string, dockerConfig *dtversion.DockerConfig) (dtversion.ImageVersion, error) {
		return dtversion.ImageVersion{}, nil
	}

	r := NewReconciler(clt, clt, scheme.Scheme, dtc, log, instance, imgVerProvider,
		&instance.Spec.RoutingSpec.CapabilityProperties, "router", "MSGrouter", "")
	require.NotNil(t, r)
	require.NotNil(t, r.Client)
	require.NotNil(t, r.scheme)
	require.NotNil(t, r.dtc)
	require.NotNil(t, r.log)
	require.NotNil(t, r.Instance)
	require.NotNil(t, r.imageVersionProvider)

	return r
}

func TestReconcile(t *testing.T) {
	t.Run(`reconcile custom properties`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		r.Instance.Spec.RoutingSpec.CustomProperties = &dynatracev1alpha1.DynaKubeValueSource{
			Value: testValue,
		}
		_, err := r.Reconcile()

		assert.NoError(t, err)

		var customProperties corev1.Secret
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + "-" + r.feature + "-" + customproperties.Suffix, Namespace: r.Instance.Namespace}, &customProperties)
		assert.NoError(t, err)
		assert.NotNil(t, customProperties)
		assert.Contains(t, customProperties.Data, customproperties.DataKey)
		assert.Equal(t, testValue, string(customProperties.Data[customproperties.DataKey]))
	})
	t.Run(`create stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		update, err := r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + "-" + r.feature, Namespace: r.Instance.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)
	})
	t.Run(`update stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		update, err := r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + "-" + r.feature, Namespace: r.Instance.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		r.Instance.Spec.Proxy = &dynatracev1alpha1.DynaKubeProxy{Value: testValue}
		update, err = r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		newStatefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + "-" + r.feature, Namespace: r.Instance.Namespace}, newStatefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		found := false
		for _, env := range newStatefulSet.Spec.Template.Spec.Containers[0].Env {
			if env.Name == DTInternalProxy {
				found = true
				assert.Equal(t, testValue, env.Value)
			}
		}
		assert.True(t, found)
	})
}

func TestReconcile_GetStatefulSet(t *testing.T) {
	r := createDefaultReconciler(t)
	update, err := r.Reconcile()
	assert.True(t, update)
	assert.NoError(t, err)

	desiredSts, err := r.buildDesiredStatefulSet()
	assert.NoError(t, err)
	assert.NotNil(t, desiredSts)

	desiredSts.Kind = "StatefulSet"
	desiredSts.APIVersion = "apps/v1"
	desiredSts.ResourceVersion = "1"
	err = controllerutil.SetControllerReference(r.Instance, desiredSts, r.scheme)
	require.NoError(t, err)

	sts, err := r.getStatefulSet(desiredSts)
	assert.NoError(t, err)
	assert.Equal(t, *desiredSts, *sts)
}

func TestReconcile_CreateStatefulSetIfNotExists(t *testing.T) {
	r := createDefaultReconciler(t)
	desiredSts, err := r.buildDesiredStatefulSet()
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	created, err := r.createStatefulSetIfNotExists(desiredSts)
	assert.NoError(t, err)
	assert.True(t, created)

	created, err = r.createStatefulSetIfNotExists(desiredSts)
	assert.NoError(t, err)
	assert.False(t, created)
}

func TestReconcile_UpdateStatefulSetIfOutdated(t *testing.T) {
	r := createDefaultReconciler(t)
	desiredSts, err := r.buildDesiredStatefulSet()
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	updated, err := r.updateStatefulSetIfOutdated(desiredSts)
	assert.Error(t, err)
	assert.False(t, updated)
	assert.True(t, k8serrors.IsNotFound(errors.Cause(err)))

	created, err := r.createStatefulSetIfNotExists(desiredSts)
	require.True(t, created)
	require.NoError(t, err)

	updated, err = r.updateStatefulSetIfOutdated(desiredSts)
	assert.NoError(t, err)
	assert.False(t, updated)

	r.Instance.Spec.Proxy = &dynatracev1alpha1.DynaKubeProxy{Value: testValue}
	desiredSts, err = r.buildDesiredStatefulSet()
	require.NoError(t, err)

	updated, err = r.updateStatefulSetIfOutdated(desiredSts)
	assert.NoError(t, err)
	assert.True(t, updated)
}

func TestReconcile_DeleteStatefulSetIfOldLabelsAreUsed(t *testing.T) {
	r := createDefaultReconciler(t)
	desiredSts, err := r.buildDesiredStatefulSet()
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	deleted, err := r.deleteStatefulSetIfOldLabelsAreUsed(desiredSts)
	assert.Error(t, err)
	assert.False(t, deleted)
	assert.True(t, k8serrors.IsNotFound(errors.Cause(err)))

	created, err := r.createStatefulSetIfNotExists(desiredSts)
	require.True(t, created)
	require.NoError(t, err)

	deleted, err = r.deleteStatefulSetIfOldLabelsAreUsed(desiredSts)
	assert.NoError(t, err)
	assert.False(t, deleted)

	r.Instance.Spec.Proxy = &dynatracev1alpha1.DynaKubeProxy{Value: testValue}
	desiredSts, err = r.buildDesiredStatefulSet()
	require.NoError(t, err)
	correctLabels := desiredSts.Labels
	desiredSts.Labels = map[string]string{"activegate": "dynakube"}
	err = r.Update(context.TODO(), desiredSts)
	assert.NoError(t, err)

	desiredSts.Labels = correctLabels
	deleted, err = r.deleteStatefulSetIfOldLabelsAreUsed(desiredSts)
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestReconcile_GetCustomPropertyHash(t *testing.T) {
	r := createDefaultReconciler(t)
	hash, err := r.calculateCustomPropertyHash()
	assert.NoError(t, err)
	assert.Empty(t, hash)

	r.Instance.Spec.RoutingSpec.CustomProperties = &dynatracev1alpha1.DynaKubeValueSource{Value: testValue}
	hash, err = r.calculateCustomPropertyHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	r.Instance.Spec.RoutingSpec.CustomProperties = &dynatracev1alpha1.DynaKubeValueSource{ValueFrom: testName}
	hash, err = r.calculateCustomPropertyHash()
	assert.Error(t, err)
	assert.Empty(t, hash)

	err = r.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			customproperties.DataKey: []byte(testValue),
		},
	})
	require.NoError(t, err)

	hash, err = r.calculateCustomPropertyHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestNewReconciler(t *testing.T) {
	type args struct {
		clt                  client.Client
		apiReader            client.Reader
		scheme               *runtime.Scheme
		dtc                  dtclient.Client
		log                  logr.Logger
		instance             *v1alpha1.DynaKube
		imageVersionProvider dtversion.ImageVersionProvider
		capability           *v1alpha1.CapabilityProperties
		feature              string
		capabilityName       string
		serviceAccountOwner  string
	}
	tests := []struct {
		name string
		args args
		want *Reconciler
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewReconciler(tt.args.clt, tt.args.apiReader, tt.args.scheme, tt.args.dtc, tt.args.log, tt.args.instance, tt.args.imageVersionProvider, tt.args.capability, tt.args.feature, tt.args.capabilityName, tt.args.serviceAccountOwner); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewReconciler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_AddOnAfterStatefulSetCreateListener(t *testing.T) {
	type args struct {
		event StatefulSetEvent
	}
	tests := []struct {
		name string
		r    *Reconciler
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.AddOnAfterStatefulSetCreateListener(tt.args.event)
		})
	}
}

func TestReconciler_Reconcile(t *testing.T) {
	tests := []struct {
		name       string
		r          *Reconciler
		wantUpdate bool
		wantErr    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUpdate, err := tt.r.Reconcile()
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUpdate != tt.wantUpdate {
				t.Errorf("Reconciler.Reconcile() = %v, want %v", gotUpdate, tt.wantUpdate)
			}
		})
	}
}

func TestReconciler_manageStatefulSet(t *testing.T) {
	tests := []struct {
		name    string
		r       *Reconciler
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.manageStatefulSet()
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconciler.manageStatefulSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Reconciler.manageStatefulSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_buildDesiredStatefulSet(t *testing.T) {
	tests := []struct {
		name    string
		r       *Reconciler
		want    *appsv1.StatefulSet
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.buildDesiredStatefulSet()
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconciler.buildDesiredStatefulSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reconciler.buildDesiredStatefulSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_getStatefulSet(t *testing.T) {
	type args struct {
		desiredSts *appsv1.StatefulSet
	}
	tests := []struct {
		name    string
		r       *Reconciler
		args    args
		want    *appsv1.StatefulSet
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.getStatefulSet(tt.args.desiredSts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconciler.getStatefulSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reconciler.getStatefulSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_createStatefulSetIfNotExists(t *testing.T) {
	type args struct {
		desiredSts *appsv1.StatefulSet
	}
	tests := []struct {
		name    string
		r       *Reconciler
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.createStatefulSetIfNotExists(tt.args.desiredSts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconciler.createStatefulSetIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Reconciler.createStatefulSetIfNotExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_updateStatefulSetIfOutdated(t *testing.T) {
	type args struct {
		desiredSts *appsv1.StatefulSet
	}
	tests := []struct {
		name    string
		r       *Reconciler
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.updateStatefulSetIfOutdated(tt.args.desiredSts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconciler.updateStatefulSetIfOutdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Reconciler.updateStatefulSetIfOutdated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_deleteStatefulSetIfOldLabelsAreUsed(t *testing.T) {
	type args struct {
		desiredSts *appsv1.StatefulSet
	}
	tests := []struct {
		name    string
		r       *Reconciler
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.deleteStatefulSetIfOldLabelsAreUsed(tt.args.desiredSts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconciler.deleteStatefulSetIfOldLabelsAreUsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Reconciler.deleteStatefulSetIfOldLabelsAreUsed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_calculateCustomPropertyHash(t *testing.T) {
	tests := []struct {
		name    string
		r       *Reconciler
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.calculateCustomPropertyHash()
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconciler.calculateCustomPropertyHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Reconciler.calculateCustomPropertyHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_getDataFromCustomProperty(t *testing.T) {
	type args struct {
		customProperties *v1alpha1.DynaKubeValueSource
	}
	tests := []struct {
		name    string
		r       *Reconciler
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.getDataFromCustomProperty(tt.args.customProperties)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconciler.getDataFromCustomProperty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Reconciler.getDataFromCustomProperty() = %v, want %v", got, tt.want)
			}
		})
	}
}
