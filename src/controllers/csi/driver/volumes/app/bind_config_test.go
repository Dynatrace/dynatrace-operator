package appvolumes

import (
	"context"
	"path/filepath"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testDkName       = "a-dynakube"
	testTenantUUID   = "a-tenant-uuid"
	testAgentVersion = "1.2-3"
	testNamespace    = "test"
)

func TestNewBindConfig(t *testing.T) {
	t.Run(`no namespace`, func(t *testing.T) {
		clt := fake.NewClient()
		pub := &Publisher{
			client: clt,
			db:     metadata.FakeMemoryDB(),
		}
		volumeCfg := &csivolumes.VolumeConfig{
			Namespace: testNamespace,
		}

		bindCfg, err := newBindConfig(context.TODO(), pub, volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`no dynakube instance label`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}})
		pub := &Publisher{
			client: clt,
			db:     metadata.FakeMemoryDB(),
		}
		volumeCfg := &csivolumes.VolumeConfig{
			Namespace: testNamespace,
		}

		bindCfg, err := newBindConfig(context.TODO(), pub, volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`no dynakube in storage`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: testDkName},
			},
		)
		pub := &Publisher{
			client: clt,
			db:     metadata.FakeMemoryDB(),
		}
		volumeCfg := &csivolumes.VolumeConfig{
			Namespace: testNamespace,
		}

		bindCfg, err := newBindConfig(context.TODO(), pub, volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to create directories`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace, Labels: map[string]string{webhook.LabelInstance: testDkName}}})
		pub := &Publisher{
			client: clt,
			fs:     afero.Afero{Fs: afero.NewMemMapFs()},
			db:     metadata.FakeMemoryDB(),
		}
		volumeCfg := &csivolumes.VolumeConfig{
			Namespace: testNamespace,
		}

		bindCfg, err := newBindConfig(context.TODO(), pub, volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`failed to read version file`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace, Labels: map[string]string{webhook.LabelInstance: testDkName}}})
		pub := &Publisher{
			client: clt,
			fs:     afero.Afero{Fs: afero.NewMemMapFs()},
			db:     metadata.FakeMemoryDB(),
		}
		volumeCfg := &csivolumes.VolumeConfig{
			Namespace: testNamespace,
		}

		bindCfg, err := newBindConfig(context.TODO(), pub, volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`create correct bind config`, func(t *testing.T) {
		clt := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace, Labels: map[string]string{webhook.LabelInstance: testDkName}}},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: testDkName},
			},
		)
		opts := dtcsi.CSIOptions{RootDir: "/"}
		pub := &Publisher{
			client: clt,
			fs:     afero.Afero{Fs: afero.NewMemMapFs()},
			db:     metadata.FakeMemoryDB(),
			path:   metadata.PathResolver{RootDir: opts.RootDir},
		}
		volumeCfg := &csivolumes.VolumeConfig{
			Namespace: testNamespace,
		}

		pub.db.InsertDynakube(metadata.NewDynakube(testDkName, testTenantUUID, testAgentVersion))

		bindCfg, err := newBindConfig(context.TODO(), pub, volumeCfg)

		assert.NoError(t, err)
		assert.NotNil(t, bindCfg)
		assert.Equal(t, filepath.Join(opts.RootDir, testTenantUUID, "bin", testAgentVersion), pub.path.AgentBinaryDirForVersion(testTenantUUID, testAgentVersion))
		assert.Equal(t, filepath.Join(opts.RootDir, testTenantUUID), pub.path.EnvDir(testTenantUUID))
	})
}
