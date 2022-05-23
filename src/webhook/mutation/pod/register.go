package pod

import (
	"context"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	webhook2 "github.com/Dynatrace/dynatrace-operator/src/webhook"
	dataingest_mutation "github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod/dataingest"
	oneagent_mutation "github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod/oneagent"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func registerInjectEndpoint(mgr manager.Manager, webhookNamespace string, webhookPodName string) error {
	// Don't use mgr.GetClient() on this function, or other cache-dependent functions from the manager. The cache may
	// not be ready at this point, and queries for Kubernetes objects may fail. mgr.GetAPIReader() doesn't depend on the
	// cache and is safe to use.

	kubeConfig := mgr.GetConfig()
	apmExists, err := kubeobjects.CheckIfOneAgentAPMExists(kubeConfig)
	if err != nil {
		return err
	}
	if apmExists {
		log.Info("OneAgentAPM object detected - DynaKube webhook won't inject until the OneAgent Operator has been uninstalled")
	}

	kubeClient := mgr.GetClient()
	apiReader := mgr.GetAPIReader()
	// the injected podMutator.client doesn't have permissions to Get(sth) from a different namespace
	metaClient, err := client.New(kubeConfig, client.Options{})
	if err != nil {
		return err
	}

	webhookPodImage, err := getWebhookPodImage(apiReader, webhookPodName, webhookNamespace)
	if err != nil {
		return err
	}

	clusterID, err := getClusterID(apiReader)
	if err != nil {
		return err
	}

	mgr.GetWebhookServer().Register("/inject", &webhook.Admission{Handler: &podMutatorWebhook{
		apiReader:        apiReader,
		webhookNamespace: webhookNamespace,
		webhookImage:     webhookPodImage,
		apmExists:        apmExists,
		clusterID:        clusterID,
		recorder:         newBasePodMutatorEventRecorder(mgr.GetEventRecorderFor("Webhook Server")),
		mutators: []webhook2.PodMutator{
			oneagent_mutation.NewOneAgentPodMutator(
				webhookPodImage,
				clusterID,
				webhookNamespace,
				kubeClient,
				apiReader,
			),
			dataingest_mutation.NewDataIngestPodMutator(
				webhookNamespace,
				kubeClient,
				apiReader,
				metaClient,
			),
		},
	}})
	return nil
}

func registerLivezEndpoint(mgr manager.Manager) {
	mgr.GetWebhookServer().Register("/livez", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func getWebhookPodImage(apiReader client.Reader, podName string, namespaceName string) (string, error) {
	var pod v1.Pod
	if err := apiReader.Get(context.TODO(), client.ObjectKey{
		Name:      podName,
		Namespace: namespaceName,
	}, &pod); err != nil {
		return "", err
	}
	return pod.Spec.Containers[0].Image, nil
}

func getClusterID(apiReader client.Reader) (string, error) {
	var clusterUID types.UID
	var err error
	if clusterUID, err = kubesystem.GetUID(apiReader); err != nil {
		return "", err
	}
	return string(clusterUID), nil
}
