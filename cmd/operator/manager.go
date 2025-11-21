package operator

import (
	"os"
	"strconv"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/nodes"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/envvars"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // important for running operator locally
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

type controllerSetupFunc func(manager.Manager, string, *corev1.Pod) error

func getControllerAddFuncs(isOLM bool) []controllerSetupFunc {
	funcs := []controllerSetupFunc{
		dynakube.Add,
		edgeconnect.Add,
	}

	if envvars.GetBool(consts.HostAvailabilityDetectionEnvVar, true) {
		funcs = append(funcs, nodes.Add)
	}

	if !isOLM {
		funcs = append(funcs, certificates.Add)
	}

	return funcs
}

func createOperatorManager(cfg *rest.Config, namespace string, isOLM bool, operatorPod *corev1.Pod) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(cfg, createOptions(namespace))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = mgr.AddHealthzCheck(livezEndpointName, healthz.Ping)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	addFuncs := getControllerAddFuncs(isOLM)

	for _, add := range addFuncs {
		err = add(mgr, namespace, operatorPod)
		if err != nil {
			return nil, err
		}
	}

	return mgr, nil
}

func createOptions(namespace string) ctrl.Options {
	return ctrl.Options{
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
		Scheme: scheme.Scheme,
		Metrics: server.Options{
			BindAddress: metricsBindAddress,
		},
		LeaderElection:             true,
		LeaderElectionID:           leaderElectionID,
		LeaderElectionResourceLock: leaderElectionResourceLock,
		LeaderElectionNamespace:    namespace,
		HealthProbeBindAddress:     healthProbeBindAddress,
		LivenessEndpointName:       livenessEndpointName,
		LeaseDuration:              getTimeFromEnvWithDefault(leaderElectionEnvVarLeaseDuration, defaultLeaseDuration),
		RenewDeadline:              getTimeFromEnvWithDefault(leaderElectionEnvVarRenewDeadline, defaultRenewDeadline),
		RetryPeriod:                getTimeFromEnvWithDefault(leaderElectionEnvVarRetryPeriod, defaultRetryPeriod),
	}
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
