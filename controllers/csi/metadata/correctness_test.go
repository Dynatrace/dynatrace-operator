package metadata

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckStorageCorrectness_FreshDB(t *testing.T) {
	// db without tables
	db := emptyMemoryDB()
	log := logger.NewDTLogger()

	err := CorrectMetadata(nil, db, log)

	assert.Error(t, err)
}

func TestCheckStorageCorrectness_EmptyDB(t *testing.T) {
	// db with tables but empty
	db := FakeMemoryDB()
	log := logger.NewDTLogger()

	err := CorrectMetadata(nil, db, log)

	assert.NoError(t, err)
}

func TestCheckStorageCorrectness_DoNothing(t *testing.T) {
	db := FakeMemoryDB()
	db.InsertVolume(&testVolume1)
	log := logger.NewDTLogger()
	client := fake.NewClient(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testVolume1.PodName}},
		&dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDynakube1.Name}},
	)

	err := CorrectMetadata(client, db, log)

	assert.NoError(t, err)
	vol, err := db.GetVolume(testVolume1.VolumeID)
	assert.NoError(t, err)
	assert.Equal(t, &testVolume1, vol)
}

func TestCheckStorageCorrectness_PURGE(t *testing.T) {
	db := FakeMemoryDB()
	db.InsertVolume(&testVolume1)
	db.InsertVolume(&testVolume2)
	db.InsertVolume(&testVolume3)
	db.InsertDynakube(&testDynakube1)
	db.InsertDynakube(&testDynakube2)
	db.InsertDynakube(&testDynakube3)
	log := logger.NewDTLogger()
	client := fake.NewClient(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testVolume1.PodName}},
		&dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: testDynakube1.Name}},
	)

	err := CorrectMetadata(client, db, log)
	assert.NoError(t, err)

	vol, err := db.GetVolume(testVolume1.VolumeID)
	assert.NoError(t, err)
	assert.Equal(t, &testVolume1, vol)

	ten, err := db.GetDynakube(testDynakube1.Name)
	assert.NoError(t, err)
	assert.Equal(t, &testDynakube1, ten)

	// PURGED
	vol, err = db.GetVolume(testVolume2.VolumeID)
	assert.NoError(t, err)
	assert.Nil(t, vol)

	// PURGED
	vol, err = db.GetVolume(testVolume3.VolumeID)
	assert.NoError(t, err)
	assert.Nil(t, vol)

	// PURGED
	ten, err = db.GetDynakube(testDynakube2.TenantUUID)
	assert.NoError(t, err)
	assert.Nil(t, ten)

	// PURGED
	ten, err = db.GetDynakube(testDynakube3.TenantUUID)
	assert.NoError(t, err)
	assert.Nil(t, ten)
}
