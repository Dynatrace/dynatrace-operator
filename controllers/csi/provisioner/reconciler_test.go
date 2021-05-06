package csiprovisioner

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	"time"
)

const (
	testName  = "test-name"
	testError = "test-error"
)

func TestOneAgentProvisioner_Reconcile(t *testing.T) {
	t.Run(`no dynakube instance`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(),
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`code modules not enabled`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: false,
						},
					},
				},
			),
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{RequeueAfter: 30 * time.Minute}, result)
	})
	t.Run(`no tokens`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
						},
					},
				},
			),
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})

		assert.EqualError(t, err, `failed to query tokens: secrets "`+testName+`" not found`)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error when creating dynatrace client`, func(t *testing.T) {
		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
				},
			),
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return nil, fmt.Errorf(testError)
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})

		assert.EqualError(t, err, "failed to create Dynatrace client: "+testError)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	t.Run(`error when querying dynatrace client for connection info`, func(t *testing.T) {
		mockClient := &dtclient.MockDynatraceClient{}
		mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{}, fmt.Errorf(testError))

		r := &OneAgentProvisioner{
			client: fake.NewClient(
				&v1alpha1.DynaKube{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
					Spec: v1alpha1.DynaKubeSpec{
						CodeModules: v1alpha1.CodeModulesSpec{
							Enabled: true,
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: testName,
					},
				},
			),
			dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
				return mockClient, nil
			},
		}
		result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})

		assert.EqualError(t, err, "failed to fetch connection info: "+testError)
		assert.NotNil(t, result)
		assert.Equal(t, reconcile.Result{}, result)
	})
	//t.Run(`error when querying dynatrace client for connection info`, func(t *testing.T) {
	//	mockClient := &dtclient.MockDynatraceClient{}
	//	mockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{}, nil)
	//
	//	r := &OneAgentProvisioner{
	//		client: fake.NewClient(
	//			&v1alpha1.DynaKube{
	//				ObjectMeta: metav1.ObjectMeta{
	//					Name: testName,
	//				},
	//				Spec: v1alpha1.DynaKubeSpec{
	//					CodeModules: v1alpha1.CodeModulesSpec{
	//						Enabled: true,
	//					},
	//				},
	//			},
	//			&v1.Secret{
	//				ObjectMeta: metav1.ObjectMeta{
	//					Name: testName,
	//				},
	//			},
	//		),
	//		dtcBuildFunc: func(rtc client.Client, instance *v1alpha1.DynaKube, secret *v1.Secret) (dtclient.Client, error) {
	//			return mockClient, nil
	//		},
	//	}
	//	result, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: testName}})
	//
	//	assert.EqualError(t, err, "failed to fetch connection info: "+testError)
	//	assert.NotNil(t, result)
	//	assert.Equal(t, reconcile.Result{}, result)
	//})
}
