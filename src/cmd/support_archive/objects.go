package support_archive

import (
	"bytes"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type objectQuery struct {
	groupVersionKind schema.GroupVersionKind
	listOptions      []client.ListOption
}

func collectManifests(cicCtx *supportArchiveContext, tarBall *tarball) error {
	for _, manifest := range getObjectsQuery(cicCtx) {
		objectList, err := readObjectsList(cicCtx, manifest)
		if err != nil {
			logErrorf("could not get manifest for %s: %v", manifest.groupVersionKind.Kind, err)
			continue
		}
		for _, object := range objectList.Items {
			marshallObjects(cicCtx, tarBall, object)
		}
	}
	return nil
}

func readObjectsList(cicCtx *supportArchiveContext, manifest objectQuery) (*unstructured.UnstructuredList, error) {
	objectList := &unstructured.UnstructuredList{}
	objectList.SetGroupVersionKind(manifest.groupVersionKind)

	err := cicCtx.apiReader.List(cicCtx.ctx, objectList, manifest.listOptions...)
	if err != nil {
		return nil, err
	}
	return objectList, nil
}

func marshallObjects(cicCtx *supportArchiveContext, tarBall *tarball, object unstructured.Unstructured) {
	if document, err := object.MarshalJSON(); err == nil {
		fileName := createFileName(object.GetKind(), object)

		err = tarBall.addFile(fileName, bytes.NewReader(document))
		if err != nil {
			logErrorf("Failed to add %s to tarball", fileName)
		} else {
			logInfof("Collected manifest for %s", fileName)
		}
	} else {
		logErrorf("Failed to marshal deployment %s/%s", cicCtx.namespaceName, object.GetName())
	}
}

func createFileName(kind string, objectMeta unstructured.Unstructured) string {
	if len(objectMeta.GetNamespace()) > 0 {
		return fmt.Sprintf("%s-%s-%s.yaml", kind, objectMeta.GetNamespace(), objectMeta.GetName())
	} else {
		return fmt.Sprintf("%s-%s.yaml", kind, objectMeta.GetName())
	}
}
