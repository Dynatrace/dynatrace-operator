package storage

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testVol1 = Volume{
		ID:         "vol-1",
		PodName:    "pod1",
		Version:    "123",
		TenantUUID: "asc",
	}
	testVol2 = Volume{
		ID:         "vol-2",
		PodName:    "pod2",
		Version:    "223",
		TenantUUID: "asc",
	}
	testVol3 = Volume{
		ID:         "vol-3",
		PodName:    "pod3",
		Version:    "323",
		TenantUUID: "asc",
	}
)

func TestCheckStorageCorrectness_FreshDB(t *testing.T) {
	// db without tables
	db := emptyMemoryDB()
	log := logger.NewDTLogger()

	err := CheckStorageCorrectness(nil, db, log)

	assert.Error(t, err)
}

func TestCheckStorageCorrectness_EmptyDB(t *testing.T) {
	// db with tables but empty
	db := FakeMemoryDB()
	log := logger.NewDTLogger()

	err := CheckStorageCorrectness(nil, db, log)

	assert.NoError(t, err)
}

func TestCheckStorageCorrectness_DoNothing(t *testing.T) {
	db := FakeMemoryDB()
	db.InsertVolumeInfo(&testVol1)
	log := logger.NewDTLogger()
	client := fake.NewClient(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testVol1.PodName}})

	err := CheckStorageCorrectness(client, db, log)

	assert.NoError(t, err)
	vol, err := db.GetVolumeInfo(testVol1.ID)
	assert.NoError(t, err)
	assert.Equal(t, &testVol1, vol)
}

func TestCheckStorageCorrectness_PURGE(t *testing.T) {
	db := FakeMemoryDB()
	db.InsertVolumeInfo(&testVol1)
	db.InsertVolumeInfo(&testVol2)
	db.InsertVolumeInfo(&testVol3)
	log := logger.NewDTLogger()
	client := fake.NewClient(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testVol1.PodName}})

	err := CheckStorageCorrectness(client, db, log)
	assert.NoError(t, err)

	vol, err := db.GetVolumeInfo(testVol1.ID)
	assert.NoError(t, err)
	assert.Equal(t, &testVol1, vol)

	// PURGED
	vol, err = db.GetVolumeInfo(testVol2.ID)
	assert.NoError(t, err)
	assert.Nil(t, vol)

	// PURGED
	vol, err = db.GetVolumeInfo(testVol3.ID)
	assert.NoError(t, err)
	assert.Nil(t, vol)
}
