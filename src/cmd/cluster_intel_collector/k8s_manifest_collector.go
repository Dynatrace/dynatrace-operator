package cluster_intel_collector

import (
	"bytes"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func collectManifests(ctx *intelCollectorContext, tarBall *intelTarball) error {
	for _, manifest := range getManifestList(ctx) {
		objectList, err := readManifestList(ctx, manifest)
		if err != nil {
			logErrorf("could not get manifest for %s: %v", manifest.gvk.Kind, err)
			continue
		}
		for _, object := range objectList.Items {
			marshallManifest(ctx, tarBall, object)
		}
	}

	return nil
}

func readManifestList(ctx *intelCollectorContext, manifest manifestSpec) (*unstructured.UnstructuredList, error) {
	objectList := &unstructured.UnstructuredList{}
	objectList.SetGroupVersionKind(manifest.gvk)

	err := ctx.apiReader.List(ctx.ctx, objectList, manifest.listOptions...)
	if err != nil {
		return nil, err
	}
	return objectList, nil
}

func marshallManifest(ctx *intelCollectorContext, tarBall *intelTarball, object unstructured.Unstructured) {
	if document, err := object.MarshalJSON(); err == nil {
		fileName := createFileName(object.GetKind(), object)

		err = tarBall.addFile(fileName, bytes.NewReader(document))
		if err != nil {
			logErrorf("Failed to add %s to tarball", fileName)
		} else {
			logInfof("Collected manifest for %s", fileName)
		}
	} else {
		logErrorf("Failed to marshal deployment %s/%s", ctx.namespaceName, object.GetName())
	}
}

func createFileName(kind string, objectMeta unstructured.Unstructured) string {
	if len(objectMeta.GetNamespace()) > 0 {
		return fmt.Sprintf("%s-%s-%s.yaml", kind, objectMeta.GetNamespace(), objectMeta.GetName())
	} else {
		return fmt.Sprintf("%s-%s.yaml", kind, objectMeta.GetName())
	}
}
