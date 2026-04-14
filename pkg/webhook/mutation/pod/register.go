package pod

import (
	"net/http"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/system"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/events"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler/injection"
	otlphandler "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/handler/otlp"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	otlpexporter "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/exporter"
	otlpresourceattributes "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/otlp/resourceattributes"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	webhooks "sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func registerInjectEndpoint(mgr manager.Manager, webhookNamespace string, isOpenShift bool) error {
	eventRecorder := mgr.GetEventRecorder("dynatrace-webhook")
	kubeConfig := mgr.GetConfig()
	kubeClient := mgr.GetClient()
	apiReader := mgr.GetAPIReader()

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
		webhookNamespace,
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
	webhookNamespace string,
	isOpenshift bool) (*webhook, error) {
	webhookImage := os.Getenv(k8senv.DTOperatorImageEnvName)
	if webhookImage == "" {
		return nil, errors.New("DT_OPERATOR_IMAGE env var is not set, cannot determine webhook container image")
	}

	log.Info("got webhook's image from env", "image", webhookImage)

	return &webhook{
		injectionHandler: injection.New(
			kubeClient,
			apiReader,
			eventRecorder,
			webhookImage,
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
		webhookNamespace: webhookNamespace,
		deployedViaOLM:   system.IsDeployedViaOLM(),
		decoder:          decoder,
	}, nil
}

func registerLivezEndpoint(mgr manager.Manager) {
	mgr.GetWebhookServer().Register("/livez", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	log.Info("registered /livez endpoint")
}
