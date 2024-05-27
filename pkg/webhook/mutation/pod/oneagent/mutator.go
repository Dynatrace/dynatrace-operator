package oneagent

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	webhookotel "github.com/Dynatrace/dynatrace-operator/pkg/webhook/internal/otel"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mutator struct {
	client           client.Client
	apiReader        client.Reader
	image            string
	clusterID        string
	webhookNamespace string
}

var _ dtwebhook.PodMutator = &Mutator{}

func NewMutator(image, clusterID, webhookNamespace string, client client.Client, apiReader client.Reader) *Mutator {
	return &Mutator{
		image:            image,
		clusterID:        clusterID,
		webhookNamespace: webhookNamespace,
		client:           client,
		apiReader:        apiReader,
	}
}

func (mut *Mutator) Enabled(request *dtwebhook.BaseRequest) bool {
	enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationOneAgentInject, request.DynaKube.FeatureAutomaticInjection())
	enabledOnDynakube := request.DynaKube.OneAgentNamespaceSelector() != nil

	matchesNamespaceSelector := true // if no namespace selector is configured, we just pass set this to true

	if request.DynaKube.OneAgentNamespaceSelector().Size() > 0 {
		selector, _ := metav1.LabelSelectorAsSelector(request.DynaKube.OneAgentNamespaceSelector())

		matchesNamespaceSelector = selector.Matches(labels.Set(request.Namespace.Labels))
	}

	return matchesNamespaceSelector && enabledOnPod && enabledOnDynakube
}

func (mut *Mutator) Injected(request *dtwebhook.BaseRequest) bool {
	return maputils.GetFieldBool(request.Pod.Annotations, dtwebhook.AnnotationOneAgentInjected, false)
}

func (mut *Mutator) Mutate(ctx context.Context, request *dtwebhook.MutationRequest) error {
	_, span := dtotel.StartSpan(ctx, webhookotel.Tracer())
	defer span.End()

	if !request.DynaKube.IsOneAgentCommunicationRouteClear() {
		log.Info("OneAgent were not yet able to communicate with tenant, no direct route or ready ActiveGate available, code modules have not been injected.")
		setNotInjectedAnnotations(request.Pod, dtwebhook.EmptyConnectionInfoReason)

		return nil
	}

	log.Info("injecting OneAgent into pod", "podName", request.PodName())

	if err := mut.ensureInitSecret(request); err != nil {
		span.RecordError(err)

		return err
	}

	installerInfo := getInstallerInfo(request.Pod, request.DynaKube)
	mut.addVolumes(request.Pod, request.DynaKube)
	mut.configureInitContainer(request, installerInfo)
	injectedContainers := mut.mutateUserContainers(request)
	mut.setContainerCount(request.InstallContainer, injectedContainers)
	addInjectionConfigVolumeMount(request.InstallContainer)
	setInjectedAnnotation(request.Pod)

	return nil
}

func (mut *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
	if !mut.Injected(request.BaseRequest) {
		return false
	}

	log.Info("reinvoking", "podName", request.PodName())

	return mut.reinvokeUserContainers(request)
}

func (mut *Mutator) ensureInitSecret(request *dtwebhook.MutationRequest) error {
	var initSecret corev1.Secret

	secretObjectKey := client.ObjectKey{Name: consts.AgentInitSecretName, Namespace: request.Namespace.Name}
	if err := mut.apiReader.Get(request.Context, secretObjectKey, &initSecret); k8serrors.IsNotFound(err) {
		initGenerator := initgeneration.NewInitGenerator(mut.client, mut.apiReader, mut.webhookNamespace)

		err := initGenerator.GenerateForNamespace(request.Context, request.DynaKube, request.Namespace.Name)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			log.Info("failed to create the init secret before oneagent pod injection")

			return err
		}

		log.Info("ensured that the init secret is present before oneagent pod injection")
	} else if err != nil {
		log.Info("failed to query the init secret before oneagent pod injection")

		return errors.WithStack(err)
	}

	return nil
}

func containerIsInjected(container *corev1.Container) bool {
	for _, e := range container.Env {
		if e.Name == dynatraceMetadataEnv {
			return true
		}
	}

	return false
}
