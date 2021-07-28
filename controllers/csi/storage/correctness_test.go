package storage

import (
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testTen1 = Tenant{
		UUID:          "asc1",
		LatestVersion: "123",
		Dynakube:      "dk1",
	}
	testTen2 = Tenant{
		UUID:          "asc2",
		LatestVersion: "223",
		Dynakube:      "dk2",
	}
	testTen3 = Tenant{
		UUID:          "asc3",
		LatestVersion: "323",
		Dynakube:      "dk3",
	}

	testVol1 = Volume{
		ID:         "vol-1",
		PodName:    "pod1",
		Version:    testTen1.LatestVersion,
		TenantUUID: testTen1.Dynakube,
	}
	testVol2 = Volume{
		ID:         "vol-2",
		PodName:    "pod2",
		Version:    testTen2.LatestVersion,
		TenantUUID: testTen2.Dynakube,
	}
	testVol3 = Volume{
		ID:         "vol-3",
		PodName:    "pod3",
		Version:    testTen3.LatestVersion,
		TenantUUID: testTen3.Dynakube,
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
	client := fake.NewClient(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testVol1.PodName}},
		&dynatracev1alpha1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testTen1.Dynakube}},
	)

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
	db.InsertTenant(&testTen1)
	db.InsertTenant(&testTen2)
	db.InsertTenant(&testTen3)
	log := logger.NewDTLogger()
	client := fake.NewClient(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testVol1.PodName}},
		&dynatracev1alpha1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testTen1.Dynakube}},
	)

	err := CheckStorageCorrectness(client, db, log)
	assert.NoError(t, err)

	vol, err := db.GetVolumeInfo(testVol1.ID)
	assert.NoError(t, err)
	assert.Equal(t, &testVol1, vol)

	ten, err := db.GetTenant(testTen1.UUID)
	assert.NoError(t, err)
	assert.Equal(t, &testTen1, ten)

	// PURGED
	vol, err = db.GetVolumeInfo(testVol2.ID)
	assert.NoError(t, err)
	assert.Nil(t, vol)

	// PURGED
	vol, err = db.GetVolumeInfo(testVol3.ID)
	assert.NoError(t, err)
	assert.Nil(t, vol)

	// PURGED
	ten, err = db.GetTenant(testTen2.UUID)
	assert.NoError(t, err)
	assert.Nil(t, ten)

	// PURGED
	ten, err = db.GetTenant(testTen3.UUID)
	assert.NoError(t, err)
	assert.Nil(t, ten)
}
