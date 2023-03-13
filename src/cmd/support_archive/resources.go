package support_archive

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const k8sResourceCollectorName = "k8sResourceCollector"

type k8sResourceCollector struct {
	collectorCommon
	context   context.Context
	namespace string
	apiReader client.Reader
}

func newK8sObjectCollector(context context.Context, log logr.Logger, supportArchive tarball, namespace string, apiReader client.Reader) collector { //nolint:revive // argument-limit doesn't apply to constructors
	return k8sResourceCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		context:   context,
		namespace: namespace,
		apiReader: apiReader,
	}
}

func (collector k8sResourceCollector) Do() error {
	logInfof(collector.log, "Starting K8S resource collection")

	for _, query := range getQueries(collector.namespace) {
		resourceList, err := collector.readObjectsList(query.groupVersionKind, query.filters)
		if err != nil {
			logErrorf(collector.log, err, "could not get manifest for %s", query.groupVersionKind.String())
			continue
		}
		for _, resource := range resourceList.Items {
			collector.storeObject(resource)
		}
	}
	return nil
}

func (collector k8sResourceCollector) Name() string {
	return k8sResourceCollectorName
}

func (collector k8sResourceCollector) readObjectsList(groupVersionKind schema.GroupVersionKind, listOptions []client.ListOption) (*unstructured.UnstructuredList, error) {
	resourceList := &unstructured.UnstructuredList{}
	resourceList.SetGroupVersionKind(groupVersionKind)

	err := collector.apiReader.List(collector.context, resourceList, listOptions...)
	if err != nil {
		return nil, err
	}
	return resourceList, nil
}

func (collector k8sResourceCollector) storeObject(resource unstructured.Unstructured) {
	yamlManifest, err := yaml.Marshal(resource)
	if err != nil {
		logErrorf(collector.log, err, "Failed to marshal %s %s/%s", resource.GetKind(), collector.namespace, resource.GetName())
		return
	}
	fileName := collector.createFileName(resource.GetKind(), resource)

	err = collector.supportArchive.addFile(fileName, bytes.NewBuffer(yamlManifest))
	if err != nil {
		logErrorf(collector.log, err, "Failed to add %s to support archive", fileName)
		return
	}

	logInfof(collector.log, "Collected manifest for %s", fileName)
}

func (collector k8sResourceCollector) createFileName(kind string, resourceMeta unstructured.Unstructured) string {
	kind = strings.ToLower(kind)
	switch {
	case resourceMeta.GetNamespace() != "":
		return fmt.Sprintf("%s/%s/%s/%s%s", ManifestsDirectoryName, resourceMeta.GetNamespace(), kind, resourceMeta.GetName(), ManifestsFileExtension)

	case resourceMeta.GetName() == collector.namespace:
		return fmt.Sprintf("%s/%s/%s-%s%s", ManifestsDirectoryName, collector.namespace, kind, resourceMeta.GetName(), ManifestsFileExtension)

	default:
		return fmt.Sprintf("%s/%s/%s-%s%s", ManifestsDirectoryName, InjectedNamespacesManifestsDirectoryName, kind, resourceMeta.GetName(), ManifestsFileExtension)
	}
}
