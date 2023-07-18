package namespaces

import (
	"os"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/version"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	updateInterval = 5 * time.Minute
)

func Add(mgr manager.Manager, _ string) error {
	kubeSysUID, err := kubesystem.GetUID(mgr.GetAPIReader())
	if err != nil {
		return errors.WithStack(err)
	}
	return NewController(mgr, string(kubeSysUID)).SetupWithManager(mgr)
}

// NewController returns a new ReconcileNamespace
func NewController(mgr manager.Manager, clusterID string) *Controller {
	return NewNamespaceController(mgr.GetClient(), mgr.GetAPIReader(), mgr.GetScheme(), mgr.GetConfig(), clusterID)
}

func NewNamespaceController(kubeClient client.Client, apiReader client.Reader, scheme *runtime.Scheme, config *rest.Config, clusterID string) *Controller { //nolint:revive
	return &Controller{
		client:                 kubeClient,
		apiReader:              apiReader,
		scheme:                 scheme,
		fs:                     afero.Afero{Fs: afero.NewOsFs()},
		dynatraceClientBuilder: dynatraceclient.NewBuilder(apiReader),
		config:                 config,
		operatorNamespace:      os.Getenv(kubeobjects.EnvPodNamespace),
		clusterID:              clusterID,
		versionProvider:        version.GetImageVersion,
	}
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Owns(&corev1.Secret{}).
		Complete(controller)
}

// Controller reconciles a Namespace object
type Controller struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the api-server
	client                 client.Client
	apiReader              client.Reader
	scheme                 *runtime.Scheme
	fs                     afero.Afero
	dynatraceClientBuilder dynatraceclient.Builder
	config                 *rest.Config
	operatorNamespace      string
	clusterID              string
	versionProvider        version.ImageVersionFunc
}

// Reconcile reads that state of the cluster for a DynaKube object and makes changes based on the state read
// and what is in the DynaKube.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (controller *Controller) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling Namespace", "namespace", request.Namespace, "name", request.Name)

	namespace, err := controller.getNamespace(ctx, request.Name)
	if err != nil {
		return reconcile.Result{RequeueAfter: updateInterval}, err
	}

	err = ensureSecrets(ctx, controller.client, controller.apiReader, namespace)
	if err != nil {
		return reconcile.Result{RequeueAfter: updateInterval}, err
	}

	log.Info("reconciling Namespace - done", "namespace", request.Namespace, "name", request.Name, "requeueAfter", updateInterval)
	return reconcile.Result{RequeueAfter: updateInterval}, nil
}

func (controller *Controller) getNamespace(ctx context.Context, namespaceName string) (*corev1.Namespace, error) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespaceName,
			Namespace: "",
		},
	}
	err := controller.apiReader.Get(ctx, client.ObjectKey{Name: namespace.Name, Namespace: namespace.Namespace}, namespace)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return namespace, nil
}

func ensureSecrets(ctx context.Context, clt client.Client, apiReader client.Reader, namespace *corev1.Namespace) error {
	dynakube, err := mapper.GetDynakubeForNamespace(ctx, clt, namespace)
	if err != nil {
		return err
	}

	if dynakube != nil {
		err = ensureInitSecret(ctx, clt, apiReader, dynakube, namespace.Name)
		if err != nil {
			return err
		}

		err = ensureDataIngestSecret(ctx, clt, apiReader, dynakube, namespace.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func ensureInitSecret(ctx context.Context, clt client.Client, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube, namespaceName string) error {
	var initSecret corev1.Secret
	secretObjectKey := client.ObjectKey{Name: config.AgentInitSecretName, Namespace: namespaceName}
	if err := apiReader.Get(ctx, secretObjectKey, &initSecret); k8serrors.IsNotFound(err) {
		initGenerator := initgeneration.NewInitGenerator(clt, apiReader, dynakube.Namespace)
		err := initGenerator.GenerateForNamespace(context.TODO(), *dynakube, namespaceName)
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

func ensureDataIngestSecret(ctx context.Context, clt client.Client, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube, namespaceName string) error {
	endpointGenerator := dtingestendpoint.NewEndpointSecretGenerator(clt, apiReader, dynakube.Namespace)

	var endpointSecret corev1.Secret
	err := apiReader.Get(
		ctx,
		client.ObjectKey{
			Name:      config.EnrichmentEndpointSecretName,
			Namespace: namespaceName,
		},
		&endpointSecret)
	if k8serrors.IsNotFound(err) {
		err := endpointGenerator.GenerateForNamespace(ctx, dynakube.Name, namespaceName)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			log.Info("failed to create the data-ingest endpoint secret before pod injection")
			return err
		}
		log.Info("ensured that the data-ingest endpoint secret is present before pod injection")
	} else if err != nil {
		log.Info("failed to query the data-ingest endpoint secret before pod injection")
		return err
	}

	return nil
}
