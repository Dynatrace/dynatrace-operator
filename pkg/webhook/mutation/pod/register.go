package pod

import (
	"context"
	"net/http"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oneagentapm"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	webhooks "sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func registerInjectEndpoint(ctx context.Context, mgr manager.Manager, webhookNamespace string, webhookPodName string, isOpenShift bool) error {
	eventRecorder := events.NewRecorder(mgr.GetEventRecorderFor("dynatrace-dtInjectionWebhook"))
	kubeConfig := mgr.GetConfig()
	kubeClient := mgr.GetClient()
	apiReader := mgr.GetAPIReader()

	webhookPod, err := pod.Get(ctx, apiReader, webhookPodName, webhookNamespace)
	if err != nil {
		return err
	}

	apmExists, err := oneagentapm.Exists(kubeConfig)
	if err != nil {
		return errors.WithStack(err)
	}

	if apmExists {
		eventRecorder.SendOneAgentAPMWarningEvent(webhookPod)

		return errors.New("OneAgentAPM object detected - the Dynatrace dtInjectionWebhook will not inject until the deprecated OneAgent Operator has been fully uninstalled")
	}

	// the injected podMutator.client doesn't have permissions to Get(sth) from a different namespace
	metaClient, err := client.New(kubeConfig, client.Options{})
	if err != nil {
		return errors.WithStack(err)
	}

	dtWh, err := newDtInjectionWebhook(
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

	mgr.GetWebhookServer().Register("/inject", &webhooks.Admission{Handler: dtWh})
	log.Info("registered /inject endpoint")

	otlpWh := newOtlpInjectionWebhook(
		kubeClient,
		apiReader,
		eventRecorder,
		admission.NewDecoder(mgr.GetScheme()),
		*webhookPod,
	)

	mgr.GetWebhookServer().Register("/otlp", &webhooks.Admission{Handler: otlpWh})
	log.Info("registered /otlp endpoint")

	return nil
}

func newDtInjectionWebhook( //nolint:revive
	kubeClient,
	metaClient client.Client,
	apiReader client.Reader,
	eventRecorder events.EventRecorder,
	decoder admission.Decoder,
	webhookPod corev1.Pod,
	isOpenshift bool) (*dtInjectionWebhook, error) {
	webhookPodImage, err := getWebhookContainerImage(webhookPod)
	if err != nil {
		return nil, err
	}

	return &dtInjectionWebhook{
		webhookBase: webhookBase{
			kubeClient:       kubeClient,
			decoder:          decoder,
			apiReader:        apiReader,
			webhookNamespace: webhookPod.Namespace,
			deployedViaOLM:   kubesystem.IsDeployedViaOlm(webhookPod),
			recorder:         eventRecorder,
		},
		oaMutator:       oneagent.NewMutator(),
		metaMutator:     metadata.NewMutator(metaClient),
		isOpenShift:     isOpenshift,
		webhookPodImage: webhookPodImage,
	}, nil
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

	log.Info("got dtInjectionWebhook's image", "image", webhookContainer.Image)

	return webhookContainer.Image, nil
}
