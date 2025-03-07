package pod

import (
	"context"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oneagentapm"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/events"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v1"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	webhooks "sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func registerInjectEndpoint(ctx context.Context, mgr manager.Manager, webhookNamespace string, webhookPodName string) error {
	eventRecorder := events.NewRecorder(mgr.GetEventRecorderFor("dynatrace-webhook"))
	kubeConfig := mgr.GetConfig()
	kubeClient := mgr.GetClient()
	apiReader := mgr.GetAPIReader()

	webhookPod, err := pod.Get(context.Background(), apiReader, webhookPodName, webhookNamespace)
	if err != nil {
		return err
	}

	apmExists, err := oneagentapm.Exists(kubeConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	if apmExists {
		eventRecorder.SendOneAgentAPMWarningEvent(webhookPod)

		return errors.New("OneAgentAPM object detected - the Dynatrace webhook will not inject until the deprecated OneAgent Operator has been fully uninstalled")
	}

	// the injected podMutator.client doesn't have permissions to Get(sth) from a different namespace
	metaClient, err := client.New(kubeConfig, client.Options{})
	if err != nil {
		return errors.WithStack(err)
	}

	webhookPodImage, err := getWebhookContainerImage(*webhookPod)
	if err != nil {
		return err
	}

	clusterID, err := getClusterID(ctx, apiReader)
	if err != nil {
		return err
	}

	mgr.GetWebhookServer().Register("/inject", &webhooks.Admission{Handler: &webhook{
		v1:               v1.NewInjector(apiReader, kubeClient, metaClient, eventRecorder, clusterID, webhookPodImage, webhookNamespace),
		v2:               v2.NewInjector(apiReader, eventRecorder),
		apiReader:        apiReader,
		webhookNamespace: webhookNamespace,
		deployedViaOLM:   kubesystem.IsDeployedViaOlm(*webhookPod),
		decoder:          admission.NewDecoder(mgr.GetScheme()),
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
	webhookContainer, err := container.FindContainerInPod(webhookPod, dtwebhook.WebhookContainerName)
	if err != nil {
		return "", errors.WithStack(err)
	}

	log.Info("got webhook's image", "image", webhookContainer.Image)

	return webhookContainer.Image, nil
}

func getClusterID(ctx context.Context, apiReader client.Reader) (string, error) {
	if clusterUID, err := kubesystem.GetUID(ctx, apiReader); err != nil {
		return "", errors.WithStack(err)
	} else {
		log.Info("got cluster UID", "clusterUID", clusterUID)

		return string(clusterUID), nil
	}
}
