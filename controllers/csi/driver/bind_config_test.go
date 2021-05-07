package csidriver

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName   = "test-name"
	testTenant = "test-tenant"
)

func TestCSIDriverServer_NewBindConfig(t *testing.T) {
	t.Run(`no namespace`, func(t *testing.T) {
		clt := fake.NewClient()
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: testNamespace,
			podUID:    testUid,
			flavor:    dtclient.FlavorMUSL,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				return []byte(""), nil
			},
			func(path string, perm fs.FileMode) error {
				return nil
			})

		assert.EqualError(t, err, "rpc error: code = FailedPrecondition desc = Failed to query namespace test-namespace: namespaces \"test-namespace\" not found")
		assert.Nil(t, bindCfg)
	})
	t.Run(`no dynakube instance label`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}})
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: testNamespace,
			podUID:    testUid,
			flavor:    dtclient.FlavorMUSL,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				return []byte(""), nil
			},
			func(path string, perm fs.FileMode) error {
				return nil
			})

		assert.EqualError(t, err, "rpc error: code = FailedPrecondition desc = Namespace 'test-namespace' doesn't have DynaKube assigned")
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to extract tenant from file`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace, Labels: map[string]string{webhook.LabelInstance: testName}}})
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: testNamespace,
			podUID:    testUid,
			flavor:    dtclient.FlavorMUSL,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				return []byte(""), fmt.Errorf(testError)
			},
			func(path string, perm fs.FileMode) error {
				return nil
			})

		assert.EqualError(t, err, "rpc error: code = Unavailable desc = Failed to extract tenant for DynaKube test-name: test error message")
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to create directories`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace, Labels: map[string]string{webhook.LabelInstance: testName}}})
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: testNamespace,
			podUID:    testUid,
			flavor:    dtclient.FlavorMUSL,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				return []byte(testTenant), nil
			},
			func(path string, perm fs.FileMode) error {
				return fmt.Errorf(testError)
			})

		assert.EqualError(t, err, "rpc error: code = Internal desc = test error message")
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to read version file`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace, Labels: map[string]string{webhook.LabelInstance: testName}}})
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: testNamespace,
			podUID:    testUid,
			flavor:    dtclient.FlavorMUSL,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				if strings.HasSuffix(filename, "version") {
					return []byte(""), fmt.Errorf(testError)
				}
				return []byte(testTenant), nil
			},
			func(path string, perm fs.FileMode) error {
				return nil
			})

		assert.EqualError(t, err, "rpc error: code = Internal desc = Failed to query agent directory for DynaKube test-name: test error message")
		assert.Nil(t, bindCfg)
	})
	t.Run(`create correct bind config`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace, Labels: map[string]string{webhook.LabelInstance: testName}}},
			&dynatracev1alpha1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: testName},
			},
		)
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: testNamespace,
			podUID:    testUid,
			flavor:    dtclient.FlavorMUSL,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				return []byte(testTenant), nil
			},
			func(path string, perm fs.FileMode) error {
				return nil
			})

		assert.NoError(t, err)
		assert.NotNil(t, bindCfg)
		assert.Equal(t, path.Join(dtcsi.DataPath, testTenant, "bin", fmt.Sprintf("%s-musl", testTenant)), bindCfg.agentDir)
		assert.Equal(t, path.Join(dtcsi.DataPath, testTenant), bindCfg.envDir)
	})
}
