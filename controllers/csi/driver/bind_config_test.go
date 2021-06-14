package csidriver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dkName       = "a-dynakube"
	tenantUuid   = "a-tenant-uuid"
	agentVersion = "1.2-3"
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

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg, afero.Afero{})

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

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg, afero.Afero{})

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to extract tenant from file`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{webhook.LabelInstance: dkName}}})
		srv := &CSIDriverServer{
			client: clt,
			fs:     afero.Afero{Fs: afero.NewMemMapFs()},
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
			podUID:    podUid,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg, srv.fs)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to create directories`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{webhook.LabelInstance: dkName}}})
		srv := &CSIDriverServer{
			client: clt,
			opts:   dtcsi.CSIOptions{RootDir: "/"},
			fs:     afero.Afero{Fs: afero.NewMemMapFs()},
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
			podUID:    podUid,
		}

		_ = srv.fs.WriteFile(filepath.Join(srv.opts.RootDir, "tenant-"+dkName), []byte(tenantUuid), os.ModePerm)

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg, srv.fs)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to read version file`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{webhook.LabelInstance: dkName}}})
		srv := &CSIDriverServer{
			client: clt,
			fs:     afero.Afero{Fs: afero.NewMemMapFs()},
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
			podUID:    podUid,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg, srv.fs)

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
			opts:   dtcsi.CSIOptions{RootDir: "/"},
			fs:     afero.Afero{Fs: afero.NewMemMapFs()},
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
			podUID:    podUid,
		}

		_ = srv.fs.WriteFile(filepath.Join(srv.opts.RootDir, "tenant-"+dkName), []byte(tenantUuid), os.ModePerm)
		_ = srv.fs.WriteFile(filepath.Join(srv.opts.RootDir, tenantUuid, "version"), []byte(agentVersion), os.ModePerm)

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg, srv.fs)

		assert.NoError(t, err)
		assert.NotNil(t, bindCfg)
		assert.Equal(t, filepath.Join(srv.opts.RootDir, tenantUuid, "bin", agentVersion), bindCfg.agentDir)
		assert.Equal(t, filepath.Join(srv.opts.RootDir, tenantUuid), bindCfg.envDir)
	})
}
