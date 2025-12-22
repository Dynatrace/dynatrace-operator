package supportarchive

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	k8sResourceCollectorName = "k8sResourceCollector"
	webhookValidatorName     = "dynatrace-webhook"
	crdNameSuffix            = "dynatrace.com"
)

type k8sResourceCollector struct {
	collectorCommon
	discoveryClient discovery.DiscoveryInterface
	context         context.Context
	apiReader       client.Reader
	namespace       string
	appName         string
}

func newK8sObjectCollector(context context.Context, log logd.Logger, supportArchive archiver, namespace string, appName string, apiReader client.Reader, discoveryClient discovery.DiscoveryInterface) collector { //nolint:revive // argument-limit doesn't apply to constructors
	return k8sResourceCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		context:         context,
		namespace:       namespace,
		appName:         appName,
		apiReader:       apiReader,
		discoveryClient: discoveryClient,
	}
}

func (collector k8sResourceCollector) Do() error {
	logInfof(collector.log, "Starting K8S resource collection")

	numberOfStorages := 0

	for _, query := range getQueries(collector.namespace, collector.appName) {
		resourceList, err := collector.readObjectsList(query.groupVersionKind, query.filters)
		if err != nil {
			logErrorf(collector.log, err, "could not get manifest for %s", query.groupVersionKind.String())

			continue
		}

		for _, resource := range resourceList.Items {
			numberOfStorages++

			collector.storeObject(resource)
		}
	}

	if numberOfStorages > 0 {
		webhookConfigurations, err := collector.readWebhookConfigurations()
		if err != nil {
			logErrorf(collector.log, err, "could not read webhook configurations")

			return err
		}

		for _, resource := range webhookConfigurations.Items {
			collector.storeObject(resource)
		}

		customResourceDefinitions, err := collector.readCustomResourceDefinitions()
		if err != nil {
			logErrorf(collector.log, err, "could not read custom resource definitions")

			return err
		}

		for _, resource := range customResourceDefinitions.Items {
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

func (collector k8sResourceCollector) readWebhookConfigurations() (*unstructured.UnstructuredList, error) {
	resourceList := &unstructured.UnstructuredList{}
	resourceList.SetGroupVersionKind(toGroupVersionKind(admissionregistrationv1.SchemeGroupVersion, admissionregistrationv1.MutatingWebhookConfiguration{}))
	resourceList.SetGroupVersionKind(toGroupVersionKind(admissionregistrationv1.SchemeGroupVersion, admissionregistrationv1.ValidatingWebhookConfiguration{}))

	var mutatingWebhookConfiguration admissionregistrationv1.MutatingWebhookConfiguration

	err := collector.apiReader.Get(collector.context, client.ObjectKey{Name: webhookValidatorName}, &mutatingWebhookConfiguration)
	if err != nil {
		return nil, err
	}

	resourceList.Items = append(resourceList.Items, collector.getMutatingWebhookConfiguration(mutatingWebhookConfiguration))

	var validatingWebhookConfiguration admissionregistrationv1.ValidatingWebhookConfiguration

	err = collector.apiReader.Get(collector.context, client.ObjectKey{Name: webhookValidatorName}, &validatingWebhookConfiguration)
	if err != nil {
		return nil, err
	}

	resourceList.Items = append(resourceList.Items, collector.getValidatingWebhookConfiguration(validatingWebhookConfiguration))

	return resourceList, nil
}

func (collector k8sResourceCollector) readCustomResourceDefinitions() (*unstructured.UnstructuredList, error) {
	resourceList := &unstructured.UnstructuredList{}
	resourceList.SetGroupVersionKind(toGroupVersionKind(apiextensionsv1.SchemeGroupVersion, apiextensionsv1.CustomResourceDefinition{}))

	var dynaKube apiextensionsv1.CustomResourceDefinition
	if err := collector.apiReader.Get(collector.context, client.ObjectKey{Name: "dynakubes.dynatrace.com"}, &dynaKube); err != nil {
		return nil, err
	}

	var edgeConnect apiextensionsv1.CustomResourceDefinition
	if err := collector.apiReader.Get(collector.context, client.ObjectKey{Name: "edgeconnects.dynatrace.com"}, &edgeConnect); err != nil {
		return nil, err
	}

	resourceList.Items = append(resourceList.Items, collector.getCRD(dynaKube), collector.getCRD(edgeConnect))

	return resourceList, nil
}

func (collector k8sResourceCollector) getCRD(customResourceDefinition apiextensionsv1.CustomResourceDefinition) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": apiextensionsv1.GroupName,
			"kind":       CRDKindName,
			"metadata":   customResourceDefinition.ObjectMeta,
			"spec":       customResourceDefinition.Spec,
			"status":     customResourceDefinition.Status,
		},
	}
}

func (collector k8sResourceCollector) getValidatingWebhookConfiguration(validatingWebhookConfig admissionregistrationv1.ValidatingWebhookConfiguration) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": admissionregistrationv1.GroupName,
			"kind":       ValidatingWebhookConfigurationKind,
			"metadata":   validatingWebhookConfig.ObjectMeta,
			"webhooks":   validatingWebhookConfig.Webhooks,
		},
	}
}

func (collector k8sResourceCollector) getMutatingWebhookConfiguration(mutatingWebhookConfig admissionregistrationv1.MutatingWebhookConfiguration) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": admissionregistrationv1.GroupName,
			"kind":       MutatingWebhookConfigurationKind,
			"metadata":   mutatingWebhookConfig.ObjectMeta,
			"webhooks":   mutatingWebhookConfig.Webhooks,
		},
	}
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

func isWebhookConfiguration(resourceMeta unstructured.Unstructured) bool {
	return resourceMeta.GetKind() == ValidatingWebhookConfigurationKind || resourceMeta.GetKind() == MutatingWebhookConfigurationKind
}

func (collector k8sResourceCollector) getCRDName(resourceMeta unstructured.Unstructured) string {
	field, found, err := unstructured.NestedFieldNoCopy(resourceMeta.Object, "metadata")
	if !found || err != nil {
		logErrorf(collector.log, err, "Could not determine CRD name, setting it to default")

		return "default"
	}

	objectMeta, ok := field.(metav1.ObjectMeta)
	if !ok {
		logErrorf(collector.log, err, "Could not determine CRD name, setting it to default")

		return "default"
	}

	return strings.Split(objectMeta.Name, ".")[0]
}

func (collector k8sResourceCollector) createFileName(kind string, resourceMeta unstructured.Unstructured) string {
	kind = strings.ToLower(kind)

	switch {
	case resourceMeta.GetNamespace() != "":
		return fmt.Sprintf("%s/%s/%s/%s%s", ManifestsDirectoryName, resourceMeta.GetNamespace(), kind, resourceMeta.GetName(), ManifestsFileExtension)

	case resourceMeta.GetName() == collector.namespace:
		return fmt.Sprintf("%s/%s/%s-%s%s", ManifestsDirectoryName, collector.namespace, kind, resourceMeta.GetName(), ManifestsFileExtension)

	case isWebhookConfiguration(resourceMeta):
		return fmt.Sprintf("%s/%s/%s%s", ManifestsDirectoryName, WebhookConfigurationsDirectoryName, strings.ToLower(resourceMeta.GetKind()), ManifestsFileExtension)

	case resourceMeta.GetKind() == CRDKindName:
		return fmt.Sprintf("%s/%s/%s-%s%s", ManifestsDirectoryName, CRDDirectoryName, strings.ToLower(resourceMeta.GetKind()), collector.getCRDName(resourceMeta), ManifestsFileExtension)

	default:
		return fmt.Sprintf("%s/%s/%s-%s%s", ManifestsDirectoryName, InjectedNamespacesManifestsDirectoryName, kind, resourceMeta.GetName(), ManifestsFileExtension)
	}
}
