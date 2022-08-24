package pod_mutator

import (
	"context"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod_mutator/dataingest_mutation"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod_mutator/oneagent_mutation"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func registerInjectEndpoint(mgr manager.Manager, webhookNamespace string, webhookPodName string) error {
	// Don't use mgr.GetClient() on this function, or other cache-dependent functions from the manager. The cache may
	// not be ready at this point, and queries for Kubernetes objects may fail. mgr.GetAPIReader() doesn't depend on the
	// cache and is safe to use.

	eventRecorder := newPodMutatorEventRecorder(mgr.GetEventRecorderFor("dynatrace-webhook"))
	kubeConfig := mgr.GetConfig()
	kubeClient := mgr.GetClient()
	apiReader := mgr.GetAPIReader()

	var webhookPod corev1.Pod
	if err := apiReader.Get(context.TODO(), client.ObjectKey{
		Name:      webhookPodName,
		Namespace: webhookNamespace,
	}, &webhookPod); err != nil {
		return errors.WithStack(err)
	}

	apmExists, err := kubeobjects.CheckIfOneAgentAPMExists(kubeConfig)
	if err != nil {
		return errors.WithStack(err)
	}
	if apmExists {
		eventRecorder.sendOneAgentAPMWarningEvent(&webhookPod)
		return errors.New("OneAgentAPM object detected - the dynatrace-webhook won't inject until the deprecated OneAgent Operator has been fully uninstalled")
	}

	// the injected podMutator.client doesn't have permissions to Get(sth) from a different namespace
	metaClient, err := client.New(kubeConfig, client.Options{})
	if err != nil {
		return errors.WithStack(err)
	}

	webhookPodImage, err := getWebhookContainerImage(webhookPod)
	if err != nil {
		return err
	}

	clusterID, err := getClusterID(apiReader)
	if err != nil {
		return err
	}

	deployedViaOLM, err := kubesystem.IsDeployedViaOlm(apiReader, webhookPodName, webhookNamespace)
	if err != nil {
		return err
	}

	mgr.GetWebhookServer().Register("/inject", &webhook.Admission{Handler: &podMutatorWebhook{
		apiReader:        apiReader,
		webhookNamespace: webhookNamespace,
		webhookImage:     webhookPodImage,
		deployedViaOLM:   deployedViaOLM,
		clusterID:        clusterID,
		recorder:         eventRecorder,
		mutators: []dtwebhook.PodMutator{
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
	log.Info("registered /inject endpoint")
	return nil
}

func registerLivezEndpoint(mgr manager.Manager) {
	mgr.GetWebhookServer().Register("/livez", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	log.Info("registered /livez endpoint")
}

func getWebhookContainerImage(webhookPod corev1.Pod) (string, error) {
	webhookContainer, err := kubeobjects.FindContainerInPod(webhookPod, dtwebhook.WebhookContainerName)
	if err != nil {
		return "", errors.WithStack(err)
	}
	log.Info("got webhook's image", "image", webhookContainer.Image)
	return webhookContainer.Image, nil
}

func getClusterID(apiReader client.Reader) (string, error) {
	var clusterUID types.UID
	var err error
	if clusterUID, err = kubesystem.GetUID(apiReader); err != nil {
		return "", errors.WithStack(err)
	}
	log.Info("got cluster UID", "clusterUID", clusterUID)
	return string(clusterUID), nil
}
