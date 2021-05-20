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
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dkName     = "a-dynakube"
	tenantUuid = "a-tenant-uuid"
)

func TestCSIDriverServer_NewBindConfig(t *testing.T) {
	t.Run(`no namespace`, func(t *testing.T) {
		clt := fake.NewClient()
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
			podUID:    podUid,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				return []byte(""), nil
			},
			func(path string, perm fs.FileMode) error {
				return nil
			})

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`no dynakube instance label`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
			podUID:    podUid,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				return []byte(""), nil
			},
			func(path string, perm fs.FileMode) error {
				return nil
			})

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to extract tenant from file`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{webhook.LabelInstance: dkName}}})
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
			podUID:    podUid,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				return []byte(""), fmt.Errorf(testError)
			},
			func(path string, perm fs.FileMode) error {
				return nil
			})

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to create directories`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{webhook.LabelInstance: dkName}}})
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
			podUID:    podUid,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				return []byte(tenantUuid), nil
			},
			func(path string, perm fs.FileMode) error {
				return fmt.Errorf(testError)
			})

		assert.EqualError(t, err, "rpc error: code = Internal desc = test error message")
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to read version file`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{webhook.LabelInstance: dkName}}})
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
			podUID:    podUid,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				if strings.HasSuffix(filename, "version") {
					return []byte(""), fmt.Errorf(testError)
				}
				return []byte(tenantUuid), nil
			},
			func(path string, perm fs.FileMode) error {
				return nil
			})

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`create correct bind config`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{webhook.LabelInstance: dkName}}},
			&dynatracev1alpha1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: dkName},
			},
		)
		srv := &CSIDriverServer{
			client: clt,
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
			podUID:    podUid,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg,
			func(filename string) ([]byte, error) {
				return []byte(tenantUuid), nil
			},
			func(path string, perm fs.FileMode) error {
				return nil
			})

		assert.NoError(t, err)
		assert.NotNil(t, bindCfg)
		assert.Equal(t, path.Join(dtcsi.DataPath, tenantUuid, "bin", tenantUuid), bindCfg.agentDir)
		assert.Equal(t, path.Join(dtcsi.DataPath, tenantUuid), bindCfg.envDir)
	})
}
