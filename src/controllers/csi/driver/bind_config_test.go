package csidriver

import (
	"context"
	"path/filepath"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dkName       = "a-dynakube"
	tenantUUID   = "a-tenant-uuid"
	agentVersion = "1.2-3"
)

func TestCSIDriverServer_NewBindConfig(t *testing.T) {
	t.Run(`no namespace`, func(t *testing.T) {
		clt := fake.NewClient()
		srv := &CSIDriverServer{
			client: clt,
			db:     metadata.FakeMemoryDB(),
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`no dynakube instance label`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})
		srv := &CSIDriverServer{
			client: clt,
			db:     metadata.FakeMemoryDB(),
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`no dynakube in storage`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: dkName},
			},
		)
		srv := &CSIDriverServer{
			client: clt,
			db:     metadata.FakeMemoryDB(),
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg)

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
			db:     metadata.FakeMemoryDB(),
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to read version file`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{webhook.LabelInstance: dkName}}})
		srv := &CSIDriverServer{
			client: clt,
			fs:     afero.Afero{Fs: afero.NewMemMapFs()},
			db:     metadata.FakeMemoryDB(),
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
		}

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`create correct bind config`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{webhook.LabelInstance: dkName}}},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: dkName},
			},
		)
		opts := dtcsi.CSIOptions{RootDir: "/"}
		srv := &CSIDriverServer{
			client: clt,
			opts:   opts,
			fs:     afero.Afero{Fs: afero.NewMemMapFs()},
			db:     metadata.FakeMemoryDB(),
			path:   metadata.PathResolver{RootDir: opts.RootDir},
		}
		volumeCfg := &volumeConfig{
			namespace: namespace,
		}

		srv.db.InsertDynakube(metadata.NewDynakube(dkName, tenantUUID, agentVersion))

		bindCfg, err := newBindConfig(context.TODO(), srv, volumeCfg)

		assert.NoError(t, err)
		assert.NotNil(t, bindCfg)
		assert.Equal(t, filepath.Join(srv.opts.RootDir, tenantUUID, "bin", agentVersion), srv.path.AgentBinaryDirForVersion(tenantUUID, agentVersion))
		assert.Equal(t, filepath.Join(srv.opts.RootDir, tenantUUID), srv.path.EnvDir(tenantUUID))
	})
}
