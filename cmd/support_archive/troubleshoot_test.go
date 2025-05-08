package support_archive

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestTroubleshootCollector(t *testing.T) {
	logBuffer := bytes.Buffer{}
	log := newSupportArchiveLogger(&logBuffer)

	clt := fake.NewClientWithIndex(
		&appsv1.Deployment{
			TypeMeta:   typeMeta("Deployment"),
			ObjectMeta: objectMeta("deployment1"),
		},
		&corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "core/v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "random",
			},
		},
		&dynakube.DynaKube{
			TypeMeta:   typeMeta("DynaKube"),
			ObjectMeta: objectMeta("dynakube1"),
		},
	)

	buffer := bytes.Buffer{}
	supportArchive := newZipArchive(bufio.NewWriter(&buffer))
	ctx := context.TODO()
	require.NoError(t, newTroubleshootCollector(ctx, log, supportArchive, testOperatorNamespace, clt, rest.Config{}).Do())

	assertNoErrorOnClose(t, supportArchive)

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))

	require.NoError(t, err)
	assert.Len(t, zipReader.File, 1)

	file := zipReader.File[0]
	assert.Equal(t, TroublshootOutputFileName, file.Name)

	size := file.FileInfo().Size()
	troubleshootFile := make([]byte, size)
	reader, err := file.Open()
	bytesRead, _ := reader.Read(troubleshootFile)

	if !errors.Is(err, io.EOF) {
		require.NoError(t, err)
	}

	assert.Equal(t, size, int64(bytesRead))
}
