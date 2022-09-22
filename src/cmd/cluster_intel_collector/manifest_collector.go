package cluster_intel_collector

import (
	"bytes"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func collectManifests(cicCtx *intelCollectorContext, tarBall *intelTarball) error {
	for _, manifest := range getRelevantManifests(cicCtx) {
		objectList, err := readManifestList(cicCtx, manifest)
		if err != nil {
			logErrorf("could not get manifest for %s: %v", manifest.gvk.Kind, err)
			continue
		}
		for _, object := range objectList.Items {
			marshallManifest(cicCtx, tarBall, object)
		}
	}
	return nil
}

func readManifestList(cicCtx *intelCollectorContext, manifest manifestSpec) (*unstructured.UnstructuredList, error) {
	objectList := &unstructured.UnstructuredList{}
	objectList.SetGroupVersionKind(manifest.gvk)

	err := cicCtx.apiReader.List(cicCtx.ctx, objectList, manifest.listOptions...)
	if err != nil {
		return nil, err
	}
	return objectList, nil
}

func marshallManifest(cicCtx *intelCollectorContext, tarBall *intelTarball, object unstructured.Unstructured) {
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
