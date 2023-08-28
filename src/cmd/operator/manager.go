package operator

import (
	"os"
	"strconv"
	"time"

	cmdManager "github.com/Dynatrace/dynatrace-operator/src/cmd/manager"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/certificates"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/namespaces"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/nodes"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // important for running operator locally
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	metricsBindAddress     = ":8080"
	healthProbeBindAddress = ":10080"
	operatorManagerPort    = 8383

	leaderElectionId                  = "dynatrace-operator-lock"
	leaderElectionResourceLock        = "leases"
	leaderElectionEnvVarRenewDeadline = "LEADER_ELECTION_RENEW_DEADLINE"
	leaderElectionEnvVarRetryPeriod   = "LEADER_ELECTION_RETRY_PERIOD"
	leaderElectionEnvVarLeaseDuration = "LEADER_ELECTION_LEASE_DURATION"

	livezEndpointName    = "livez"
	livenessEndpointName = "/" + livezEndpointName
)

type bootstrapManagerProvider struct {
	managerBuilder
}

func NewBootstrapManagerProvider() cmdManager.Provider {
	return bootstrapManagerProvider{}
}

func (provider bootstrapManagerProvider) CreateManager(namespace string, config *rest.Config) (manager.Manager, error) {
	controlManager, err := provider.getManager(config, ctrl.Options{
		Scheme:                 scheme.Scheme,
		Namespace:              namespace,
		HealthProbeBindAddress: healthProbeBindAddress,
		LivenessEndpointName:   livenessEndpointName,
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = controlManager.AddHealthzCheck(livezEndpointName, healthz.Ping)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return controlManager, errors.WithStack(err)
}

type operatorManagerProvider struct {
	managerBuilder
	deployedViaOlm bool
}

func NewOperatorManagerProvider(deployedViaOlm bool) cmdManager.Provider {
	return operatorManagerProvider{
		deployedViaOlm: deployedViaOlm,
	}
}

func (provider operatorManagerProvider) CreateManager(namespace string, cfg *rest.Config) (manager.Manager, error) {
	mgr, err := provider.getManager(cfg, provider.createOptions(namespace))

	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = mgr.AddHealthzCheck(livezEndpointName, healthz.Ping)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = dynakube.Add(mgr, namespace)
	if err != nil {
		return nil, err
	}

	err = nodes.Add(mgr, namespace)
	if err != nil {
		return nil, err
	}

	err = namespaces.Add(mgr, namespace)
	if err != nil {
		return nil, err
	}

	err = provider.addCertificateController(mgr, namespace)
	if err != nil {
		return nil, err
	}

	err = edgeconnect.Add(mgr, namespace)
	if err != nil {
		return nil, err
	}

	return mgr, nil
}

func (provider operatorManagerProvider) addCertificateController(mgr manager.Manager, namespace string) error {
	if !provider.deployedViaOlm {
		return certificates.Add(mgr, namespace)
	}
	return nil
}

func (provider operatorManagerProvider) createOptions(namespace string) ctrl.Options {
	return ctrl.Options{
		Namespace:                  namespace,
		Scheme:                     scheme.Scheme,
		MetricsBindAddress:         metricsBindAddress,
		Port:                       operatorManagerPort,
		LeaderElection:             true,
		LeaderElectionID:           leaderElectionId,
		LeaderElectionResourceLock: leaderElectionResourceLock,
		LeaderElectionNamespace:    namespace,
		HealthProbeBindAddress:     healthProbeBindAddress,
		LivenessEndpointName:       livenessEndpointName,
		LeaseDuration:              getTimeFromEnvWithDefault(leaderElectionEnvVarLeaseDuration, 30),
		RenewDeadline:              getTimeFromEnvWithDefault(leaderElectionEnvVarRenewDeadline, 20),
		RetryPeriod:                getTimeFromEnvWithDefault(leaderElectionEnvVarRetryPeriod, 6),
	}
}

// managerBuilder is used for testing the createManager functions in the providers
type managerBuilder struct {
	mgr manager.Manager
}

func (builder *managerBuilder) getManager(config *rest.Config, options manager.Options) (manager.Manager, error) {
	var err error
	if builder.mgr == nil {
		builder.mgr, err = ctrl.NewManager(config, options)
	}

	return builder.mgr, err
}

func (builder *managerBuilder) setManager(mgr manager.Manager) {
	builder.mgr = mgr
}

func getTimeFromEnvWithDefault(envName string, defaultValue int64) *time.Duration {
	duration := time.Duration(defaultValue) * time.Second
	envValue := os.Getenv(envName)
	if envValue != "" {
		asInt, err := strconv.ParseInt(envValue, 10, 64)
		if err == nil {
			log.Info("using non-default value for", "env", envName, "value", asInt)
			duration = time.Duration(asInt) * time.Second
			return &duration
		}
		log.Info("failed to convert envvar value to int so default value is used", "env", envName, "default", defaultValue)
	}
	return &duration
}
