package metadata

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	testutil "github.com/Dynatrace/dynatrace-operator/pkg/util/testing"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestTenantConfig(index int) *TenantConfig {
	return &TenantConfig{
		Name:                        fmt.Sprintf("dk%d", index),
		TenantUUID:                  fmt.Sprintf("asc%d", index),
		DownloadedCodeModuleVersion: fmt.Sprintf("%d", 123*index),
		MaxFailedMountAttempts:      int64(index),
		TimeStampedModel:            TimeStampedModel{},
	}
}

func createTestAppMount(index int) *AppMount {
	return &AppMount{
		VolumeMeta:        VolumeMeta{ID: fmt.Sprintf("vol-%d", index), PodName: fmt.Sprintf("pod%d", index)},
		CodeModuleVersion: strconv.Itoa(123 * index),
		CodeModule:        CodeModule{Version: strconv.Itoa(123 * index)},
		VolumeMetaID:      fmt.Sprintf("vol-%d", index),
		MountAttempts:     int64(index),
		TimeStampedModel:  TimeStampedModel{},
	}
}

func TestCorrectCSI(t *testing.T) {
	diffOptsAppMount := cmpopts.IgnoreFields(AppMount{}, "TimeStampedModel")
	diffOptsTenantConfig := cmpopts.IgnoreFields(TenantConfig{}, "TimeStampedModel")

	t.Run("error on no db or missing tables", func(t *testing.T) {
		db := emptyMemoryDB()

		checker := NewCorrectnessChecker(nil, db, dtcsi.CSIOptions{})

		err := checker.CorrectCSI(context.Background())

		require.Error(t, err)
	})
	t.Run("no error on empty db", func(t *testing.T) {
		db := FakeMemoryDB()

		checker := NewCorrectnessChecker(nil, db, dtcsi.CSIOptions{})

		err := checker.CorrectCSI(context.Background())

		require.NoError(t, err)
	})

	t.Run("no error on nil apiReader, database is not cleaned", func(t *testing.T) {
		testAppMount1 := createTestAppMount(1)
		testTenantConfig1 := createTestTenantConfig(1)
		db := FakeMemoryDB()
		db.CreateAppMount(testAppMount1)
		db.CreateTenantConfig(testTenantConfig1)

		checker := NewCorrectnessChecker(nil, db, dtcsi.CSIOptions{})

		err := checker.CorrectCSI(context.Background())

		require.NoError(t, err)
		appMount, err := db.ReadAppMount(*testAppMount1)
		require.NoError(t, err)
		testutil.PartialEqual(t, testAppMount1, appMount, diffOptsAppMount)

		tenantConfig, err := db.ReadTenantConfig(TenantConfig{Name: testTenantConfig1.Name})
		require.NoError(t, err)
		testutil.PartialEqual(t, testTenantConfig1, tenantConfig, diffOptsTenantConfig)
	})

	t.Run("nothing to remove, everything is still correct", func(t *testing.T) {
		ctx := context.Background()
		testAppMount1 := createTestAppMount(1)
		testTenantConfig1 := createTestTenantConfig(1)
		db := FakeMemoryDB()
		db.CreateAppMount(testAppMount1)
		db.CreateTenantConfig(testTenantConfig1)
		client := fake.NewClient(
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testAppMount1.VolumeMeta.PodName}},
			&dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testTenantConfig1.Name}},
		)

		checker := NewCorrectnessChecker(client, db, dtcsi.CSIOptions{})

		err := checker.CorrectCSI(ctx)
		require.NoError(t, err)

		appMount, err := db.ReadAppMount(*testAppMount1)
		require.NoError(t, err)
		testutil.PartialEqual(t, testAppMount1, appMount, diffOptsAppMount)

		require.NoError(t, err)
		tenantConfig, err := db.ReadTenantConfig(TenantConfig{Name: testTenantConfig1.Name})
		require.NoError(t, err)

		testutil.PartialEqual(t, testTenantConfig1, tenantConfig, diffOptsTenantConfig)
	})
	t.Run("remove unnecessary entries in the filesystem", func(t *testing.T) {
		testAppMount1 := createTestAppMount(1)
		testAppMount2 := createTestAppMount(2)
		testAppMount3 := createTestAppMount(3)

		testTenantConfig1 := createTestTenantConfig(1)
		testTenantConfig2 := createTestTenantConfig(2)
		testTenantConfig3 := createTestTenantConfig(3)

		db := FakeMemoryDB()
		db.CreateAppMount(testAppMount1)
		db.CreateAppMount(testAppMount2)
		db.CreateAppMount(testAppMount3)
		db.CreateTenantConfig(testTenantConfig1)
		db.CreateTenantConfig(testTenantConfig2)
		db.CreateTenantConfig(testTenantConfig3)

		client := fake.NewClient(
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testAppMount1.VolumeMeta.PodName}},
			&dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testTenantConfig1.Name}},
		)

		checker := NewCorrectnessChecker(client, db, dtcsi.CSIOptions{})

		err := checker.CorrectCSI(context.Background())
		require.NoError(t, err)

		testAppMount1.TimeStampedModel = TimeStampedModel{}
		appMount, err := db.ReadAppMount(*testAppMount1)
		require.NoError(t, err)
		testutil.PartialEqual(t, &testAppMount1, &appMount, diffOptsAppMount)

		tenantConfig, err := db.ReadTenantConfig(TenantConfig{Name: testTenantConfig1.Name})
		require.NoError(t, err)
		testutil.PartialEqual(t, &testTenantConfig1, &tenantConfig, diffOptsTenantConfig)

		// PURGED
		appMount, err = db.ReadAppMount(AppMount{VolumeMetaID: testAppMount2.VolumeMetaID})
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, appMount)

		// PURGED
		testAppMount3.TimeStampedModel = TimeStampedModel{}
		appMount, err = db.ReadAppMount(*testAppMount3)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, appMount)

		// PURGED
		tenantConfig, err = db.ReadTenantConfig(TenantConfig{Name: testTenantConfig2.TenantUUID})
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, tenantConfig)

		// PURGED
		tenantConfig, err = db.ReadTenantConfig(TenantConfig{Name: testTenantConfig3.TenantUUID})
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.Nil(t, tenantConfig)
	})
}
