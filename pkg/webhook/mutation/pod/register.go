package pod

import (
	"context"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/system"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler/injection"
	otlphandler "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	otlpexporter "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/exporter"
	otlpresourceattributes "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/resourceattributes"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	webhooks "sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func registerInjectEndpoint(ctx context.Context, mgr manager.Manager, webhookNamespace string, webhookPodName string, isOpenShift bool) error {
	eventRecorder := events.NewRecorder(
		mgr.GetEventRecorderFor("dynatrace-webhook"), //
	)
	kubeConfig := mgr.GetConfig()
	kubeClient := mgr.GetClient()
	apiReader := mgr.GetAPIReader()

	webhookPod, err := k8spod.Get(ctx, apiReader, webhookPodName, webhookNamespace)
	if err != nil {
		return err
	}

	// the injected podMutator.client doesn't have permissions to Get(sth) from a different namespace
	metaClient, err := client.New(kubeConfig, client.Options{})
	if err != nil {
		return errors.WithStack(err)
	}

	wh, err := newWebhook(
		kubeClient,
		metaClient,
		apiReader,
		eventRecorder,
		admission.NewDecoder(mgr.GetScheme()),
		*webhookPod,
		isOpenShift,
	)
	if err != nil {
		return err
	}

	mgr.GetWebhookServer().Register("/inject", &webhooks.Admission{Handler: wh})
	log.Info("registered /inject endpoint")

	return nil
}

func newWebhook( //nolint:revive
	kubeClient,
	metaClient client.Client,
	apiReader client.Reader,
	eventRecorder events.EventRecorder,
	decoder admission.Decoder,
	webhookPod corev1.Pod,
	isOpenshift bool) (*webhook, error) {
	webhookPodImage, err := getWebhookContainerImage(webhookPod)
	if err != nil {
		return nil, err
	}

	return &webhook{
		injectionHandler: injection.New(
			kubeClient,
			apiReader,
			eventRecorder,
			webhookPodImage,
			isOpenshift,
			metadata.NewMutator(metaClient),
			oneagent.NewMutator(),
		),
		otlpHandler: otlphandler.New(
			kubeClient,
			apiReader,
			otlpexporter.New(),
			otlpresourceattributes.New(metaClient),
		),
		apiReader:        apiReader,
		recorder:         eventRecorder,
		webhookNamespace: webhookPod.Namespace,
		deployedViaOLM:   system.IsDeployedViaOlm(webhookPod),
		decoder:          decoder,
	}, nil
}

func registerLivezEndpoint(mgr manager.Manager) {
	mgr.GetWebhookServer().Register("/livez", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	log.Info("registered /livez endpoint")
}

func getWebhookContainerImage(webhookPod corev1.Pod) (string, error) {
	webhookContainer, err := k8scontainer.FindInPod(webhookPod, dtwebhook.WebhookContainerName)
	if err != nil {
		return "", errors.WithStack(err)
	}

	log.Info("got webhook's image", "image", webhookContainer.Image)

	return webhookContainer.Image, nil
}
