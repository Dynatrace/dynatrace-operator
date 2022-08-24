package metadata

import (
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
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

func TestCreateTestDynakube(t *testing.T) {
	// Check instantiation
	dynakube0 := createTestDynakube(0)

	assert.Equal(t, "asc0", dynakube0.TenantUUID)
	assert.Equal(t, "0", dynakube0.LatestVersion)
	assert.Equal(t, "dk0", dynakube0.Name)
	assert.Equal(t, "sha256:0", dynakube0.ImageDigest)

	dynakube1 := createTestDynakube(1)

	assert.Equal(t, "asc1", dynakube1.TenantUUID)
	assert.Equal(t, "123", dynakube1.LatestVersion)
	assert.Equal(t, "dk1", dynakube1.Name)
	assert.Equal(t, "sha256:123", dynakube1.ImageDigest)

	dynakube2 := createTestDynakube(2)

	assert.Equal(t, "asc2", dynakube2.TenantUUID)
	assert.Equal(t, "246", dynakube2.LatestVersion)
	assert.Equal(t, "dk2", dynakube2.Name)
	assert.Equal(t, "sha256:246", dynakube2.ImageDigest)

	// Check that they are not references of each other
	dynakube0.Name = "new-name"
	newDynakube0 := createTestDynakube(0)

	assert.NotEqual(t, dynakube0.Name, newDynakube0.Name)
	assert.Equal(t, "dk0", newDynakube0.Name)

	newDynakube0 = createTestDynakube(0)
	dynakube0.Name = "new-name"

	assert.NotEqual(t, dynakube0.Name, newDynakube0.Name)
	assert.Equal(t, "dk0", newDynakube0.Name)
}

func TestCreateTestVolume(t *testing.T) {
	// Check instantiation
	volume0 := createTestVolume(0)

	assert.Equal(t, "vol-0", volume0.VolumeID)
	assert.Equal(t, "pod0", volume0.PodName)
	assert.Equal(t, "0", volume0.Version)
	assert.Equal(t, "asc0", volume0.TenantUUID)

	volume1 := createTestVolume(1)

	assert.Equal(t, "vol-1", volume1.VolumeID)
	assert.Equal(t, "pod1", volume1.PodName)
	assert.Equal(t, "123", volume1.Version)
	assert.Equal(t, "asc1", volume1.TenantUUID)

	volume2 := createTestVolume(2)

	assert.Equal(t, "vol-2", volume2.VolumeID)
	assert.Equal(t, "pod2", volume2.PodName)
	assert.Equal(t, "246", volume2.Version)
	assert.Equal(t, "asc2", volume2.TenantUUID)

	// Check that they are not references of each other
	volume0.PodName = "new-name"
	newVolume0 := createTestVolume(0)

	assert.NotEqual(t, volume0.PodName, newVolume0.PodName)
	assert.Equal(t, "pod0", newVolume0.PodName)

	newVolume0 = createTestVolume(0)
	volume0.PodName = "new-name"

	assert.NotEqual(t, volume0.PodName, newVolume0.PodName)
	assert.Equal(t, "pod0", newVolume0.PodName)
}

func TestCheckStorageCorrectness_FreshDB(t *testing.T) {
	// db without tables
	db := emptyMemoryDB()

	err := CorrectMetadata(nil, db)

	assert.Error(t, err)
}

func TestCheckStorageCorrectness_EmptyDB(t *testing.T) {
	// db with tables but empty
	db := FakeMemoryDB()

	err := CorrectMetadata(nil, db)

	assert.NoError(t, err)
}

func TestCheckStorageCorrectness_DoNothing(t *testing.T) {
	testVolume1 := createTestVolume(1)
	testDynakube1 := createTestDynakube(1)
	db := FakeMemoryDB()
	db.InsertVolume(&testVolume1)
	client := fake.NewClient(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testVolume1.PodName}},
		&dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDynakube1.Name}},
	)

	err := CorrectMetadata(client, db)

	assert.NoError(t, err)
	vol, err := db.GetVolume(testVolume1.VolumeID)
	assert.NoError(t, err)
	assert.Equal(t, &testVolume1, vol)
}

func TestCheckStorageCorrectness_PURGE(t *testing.T) {
	testVolume1 := createTestVolume(1)
	testVolume2 := createTestVolume(2)
	testVolume3 := createTestVolume(3)

	testDynakube1 := createTestDynakube(1)
	testDynakube2 := createTestDynakube(2)
	testDynakube3 := createTestDynakube(3)

	db := FakeMemoryDB()
	db.InsertVolume(&testVolume1)
	db.InsertVolume(&testVolume2)
	db.InsertVolume(&testVolume3)
	db.InsertDynakube(&testDynakube1)
	db.InsertDynakube(&testDynakube2)
	db.InsertDynakube(&testDynakube3)
	client := fake.NewClient(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testVolume1.PodName}},
		&dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDynakube1.Name}},
	)

	err := CorrectMetadata(client, db)
	require.NoError(t, err)

	vol, err := db.GetVolume(testVolume1.VolumeID)
	require.NoError(t, err)
	assert.Equal(t, &testVolume1, vol)

	ten, err := db.GetDynakube(testDynakube1.Name)
	require.NoError(t, err)
	assert.Equal(t, &testDynakube1, ten)

	// PURGED
	vol, err = db.GetVolume(testVolume2.VolumeID)
	require.NoError(t, err)
	assert.Nil(t, vol)

	// PURGED
	vol, err = db.GetVolume(testVolume3.VolumeID)
	require.NoError(t, err)
	assert.Nil(t, vol)

	// PURGED
	ten, err = db.GetDynakube(testDynakube2.TenantUUID)
	require.NoError(t, err)
	assert.Nil(t, ten)

	// PURGED
	ten, err = db.GetDynakube(testDynakube3.TenantUUID)
	require.NoError(t, err)
	assert.Nil(t, ten)
}
