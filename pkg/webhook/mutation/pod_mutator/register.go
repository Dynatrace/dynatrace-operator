package pod_mutator

import (
	"context"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oneagentapm"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod_mutator/dataingest_mutation"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod_mutator/oneagent_mutation"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func registerInjectEndpoint(mgr manager.Manager, webhookNamespace string, webhookPodName string) error {
	// Don't use mgr.GetClient() on this function, or other cache-dependent functions from the manager. The cache may
	// not be ready at this point, and queries for Kubernetes objects may fail. mgr.GetAPIReader() doesn't depend on the
	// cache and is safe to use.

	eventRecorder := newPodMutatorEventRecorder(mgr.GetEventRecorderFor("dynatrace-webhook"))
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
		eventRecorder.sendOneAgentAPMWarningEvent(webhookPod)
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

	clusterID, err := getClusterID(context.Background(), apiReader)
	if err != nil {
		return err
	}

	otelMeter := otel.Meter(otelName)
	requestCounter, err := otelMeter.Int64Counter("handledPodMutationRequests")
	if err != nil {
		return errors.WithStack(err)
	}

	mgr.GetWebhookServer().Register("/inject", &webhook.Admission{Handler: &podMutatorWebhook{
		apiReader:        apiReader,
		webhookNamespace: webhookNamespace,
		webhookImage:     webhookPodImage,
		deployedViaOLM:   kubesystem.IsDeployedViaOlm(*webhookPod),
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
		decoder:    *admission.NewDecoder(mgr.GetScheme()),
		spanTracer: otel.Tracer(otelName),
		otelMeter:  otel.Meter(otelName),

		requestCounter: requestCounter,
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
	var clusterUID types.UID
	var err error
	if clusterUID, err = kubesystem.GetUID(ctx, apiReader); err != nil {
		return "", errors.WithStack(err)
	}
	log.Info("got cluster UID", "clusterUID", clusterUID)
	return string(clusterUID), nil
}
