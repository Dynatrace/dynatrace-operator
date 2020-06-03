package activegate

import (
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := apis.AddToScheme(scheme.Scheme); err != nil {
		log.Error(err, err.Error())
	}
	if err := os.Setenv(k8sutil.WatchNamespaceEnvVar, "dynatrace"); err != nil {
		log.Error(err, err.Error())
	}
	m.Run()
}
