package troubleshoot

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkDTClusterConnection(apiReader client.Reader, troubleshootContext *TestData) error {
	tslog.SetPrefix("[dtcluster ] ")
	tslog.NewTestf("checking if tenant is accessible ...")

	dk := dynatracev1beta1.DynaKube{}
	if err := apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootContext.dynakubeName, Namespace: troubleshootContext.namespaceName}, &dk); err != nil {
		tslog.Errorf("Selected Dynakube does not exist '%s' (%s)", troubleshootContext.dynakubeName, err.Error())
		return err
	}

	dynatraceClientProperties, err := dynakube.NewDynatraceClientProperties(context.TODO(), apiReader, dk)
	if err != nil {
		tslog.WithErrorf(err, "failed to configure DynatraceAPI client")
		return err
	}

	dtc, err := dynakube.BuildDynatraceClient(*dynatraceClientProperties)
	if err != nil {
		tslog.WithErrorf(err, "failed to configure DynatraceAPI client")
		return err
	}

	_, err = dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		tslog.WithErrorf(err, "failed to connect to DynatraceAPI")
		return err
	}

	tslog.Okf("tenant is accessible")
	return nil
}
