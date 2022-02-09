package csivolumes

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testDynakubeName = "a-dynakube"
	testTenantUUID   = "a-tenant-uuid"
	testAgentVersion = "1.2-3"
)

func TestNewBindConfig(t *testing.T) {
	t.Run(`no namespace`, func(t *testing.T) {
		client := fake.NewClient()
		volumeCfg := &VolumeConfig{
			Namespace: testNamespace,
		}

		bindCfg, err := NewBindConfig(context.TODO(), client, metadata.FakeMemoryDB(), volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`no dynakube instance label`, func(t *testing.T) {
		client := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}})
		volumeCfg := &VolumeConfig{
			Namespace: testNamespace,
		}

		bindCfg, err := NewBindConfig(context.TODO(), client, metadata.FakeMemoryDB(), volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`no dynakube in storage`, func(t *testing.T) {
		client := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: testDynakubeName},
			},
		)
		volumeCfg := &VolumeConfig{
			Namespace: testNamespace,
		}

		bindCfg, err := NewBindConfig(context.TODO(), client, metadata.FakeMemoryDB(), volumeCfg)

		assert.Error(t, err)
		assert.Nil(t, bindCfg)
	})
	t.Run(`create correct bind config`, func(t *testing.T) {
		client := fake.NewClient(
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace, Labels: map[string]string{webhook.LabelInstance: testDynakubeName}}},
			&dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: testDynakubeName},
			},
		)
		volumeCfg := &VolumeConfig{
			Namespace: testNamespace,
		}

		db := metadata.FakeMemoryDB()

		db.InsertDynakube(metadata.NewDynakube(testDynakubeName, testTenantUUID, testAgentVersion))

		bindCfg, err := NewBindConfig(context.TODO(), client, db, volumeCfg)

		expected := BindConfig{
			TenantUUID: testTenantUUID,
			Version:    testAgentVersion,
		}
		assert.NoError(t, err)
		assert.NotNil(t, bindCfg)
		assert.Equal(t, expected, *bindCfg)
	})
}
