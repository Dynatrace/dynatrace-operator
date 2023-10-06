package metadata

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestDynakube(index int) Dynakube {
	return Dynakube{
		TenantUUID:             fmt.Sprintf("asc%d", index),
		LatestVersion:          fmt.Sprintf("%d", 123*index),
		Name:                   fmt.Sprintf("dk%d", index),
		ImageDigest:            fmt.Sprintf("sha256:%d", 123*index),
		MaxFailedMountAttempts: index,
	}
}

func createTestVolume(index int) Volume {
	return Volume{
		VolumeID:      fmt.Sprintf("vol-%d", index),
		PodName:       fmt.Sprintf("pod%d", index),
		Version:       createTestDynakube(index).LatestVersion,
		TenantUUID:    createTestDynakube(index).TenantUUID,
		MountAttempts: index,
	}
}

func TestCorrectCSI(t *testing.T) {
	t.Run("error on no db or missing tables", func(t *testing.T) {
		db := emptyMemoryDB()

		checker := NewCorrectnessChecker(nil, db, dtcsi.CSIOptions{})

		err := checker.CorrectCSI(context.TODO())

		assert.Error(t, err)
	})
	t.Run("no error on empty db", func(t *testing.T) {
		db := FakeMemoryDB()

		checker := NewCorrectnessChecker(nil, db, dtcsi.CSIOptions{})

		err := checker.CorrectCSI(context.TODO())

		assert.NoError(t, err)
	})

	t.Run("no error on nil apiReader, database is not cleaned", func(t *testing.T) {
		ctx := context.TODO()
		testVolume1 := createTestVolume(1)
		testDynakube1 := createTestDynakube(1)
		db := FakeMemoryDB()
		db.InsertVolume(ctx, &testVolume1)
		db.InsertDynakube(ctx, &testDynakube1)

		checker := NewCorrectnessChecker(nil, db, dtcsi.CSIOptions{})

		err := checker.CorrectCSI(context.TODO())

		assert.NoError(t, err)
		vol, err := db.GetVolume(ctx, testVolume1.VolumeID)
		assert.NoError(t, err)
		assert.Equal(t, &testVolume1, vol)

		assert.NoError(t, err)
		dk, err := db.GetDynakube(ctx, testDynakube1.Name)
		assert.NoError(t, err)
		assert.Equal(t, &testDynakube1, dk)
	})

	t.Run("nothing to remove, everything is still correct", func(t *testing.T) {
		ctx := context.TODO()
		testVolume1 := createTestVolume(1)
		testDynakube1 := createTestDynakube(1)
		db := FakeMemoryDB()
		db.InsertVolume(ctx, &testVolume1)
		db.InsertDynakube(ctx, &testDynakube1)
		client := fake.NewClient(
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testVolume1.PodName}},
			&dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDynakube1.Name}},
		)

		checker := NewCorrectnessChecker(client, db, dtcsi.CSIOptions{})

		err := checker.CorrectCSI(ctx)

		assert.NoError(t, err)
		vol, err := db.GetVolume(ctx, testVolume1.VolumeID)
		assert.NoError(t, err)
		assert.Equal(t, &testVolume1, vol)

		assert.NoError(t, err)
		dk, err := db.GetDynakube(ctx, testDynakube1.Name)
		assert.NoError(t, err)
		assert.Equal(t, &testDynakube1, dk)
	})
	t.Run("remove unnecessary entries in the filesystem", func(t *testing.T) {
		ctx := context.TODO()
		testVolume1 := createTestVolume(1)
		testVolume2 := createTestVolume(2)
		testVolume3 := createTestVolume(3)

		testDynakube1 := createTestDynakube(1)
		testDynakube2 := createTestDynakube(2)
		testDynakube3 := createTestDynakube(3)

		db := FakeMemoryDB()
		db.InsertVolume(ctx, &testVolume1)
		db.InsertVolume(ctx, &testVolume2)
		db.InsertVolume(ctx, &testVolume3)
		db.InsertDynakube(ctx, &testDynakube1)
		db.InsertDynakube(ctx, &testDynakube2)
		db.InsertDynakube(ctx, &testDynakube3)
		client := fake.NewClient(
			&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testVolume1.PodName}},
			&dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDynakube1.Name}},
		)

		checker := NewCorrectnessChecker(client, db, dtcsi.CSIOptions{})

		err := checker.CorrectCSI(ctx)
		require.NoError(t, err)

		vol, err := db.GetVolume(ctx, testVolume1.VolumeID)
		require.NoError(t, err)
		assert.Equal(t, &testVolume1, vol)

		ten, err := db.GetDynakube(ctx, testDynakube1.Name)
		require.NoError(t, err)
		assert.Equal(t, &testDynakube1, ten)

		// PURGED
		vol, err = db.GetVolume(ctx, testVolume2.VolumeID)
		require.NoError(t, err)
		assert.Nil(t, vol)

		// PURGED
		vol, err = db.GetVolume(ctx, testVolume3.VolumeID)
		require.NoError(t, err)
		assert.Nil(t, vol)

		// PURGED
		ten, err = db.GetDynakube(ctx, testDynakube2.TenantUUID)
		require.NoError(t, err)
		assert.Nil(t, ten)

		// PURGED
		ten, err = db.GetDynakube(ctx, testDynakube3.TenantUUID)
		require.NoError(t, err)
		assert.Nil(t, ten)
	})
}
