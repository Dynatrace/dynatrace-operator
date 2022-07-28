package troubleshoot

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkDTClusterConnection(troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("[dtcluster ] ")

	logNewTestf("checking if tenant is accessible ...")

	dk := dynatracev1beta1.DynaKube{}
	if err := troubleshootCtx.apiReader.Get(context.TODO(), client.ObjectKey{Name: troubleshootCtx.dynakubeName, Namespace: troubleshootCtx.namespaceName}, &dk); err != nil {
		logErrorf("Selected Dynakube does not exist '%s' (%s)", troubleshootCtx.dynakubeName, err.Error())
		return err
	}

	dynatraceClientProperties, err := dynakube.NewDynatraceClientProperties(context.TODO(), troubleshootCtx.apiReader, dk)
	if err != nil {
		logWithErrorf(err, "failed to configure DynatraceAPI client")
		return err
	}

	dtc, err := dynakube.BuildDynatraceClient(*dynatraceClientProperties)
	if err != nil {
		logWithErrorf(err, "failed to build DynatraceAPI client")
		return err
	}

	_, err = dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		logWithErrorf(err, "failed to connect to DynatraceAPI")
		return err
	}

	logOkf("tenant is accessible")
	return nil
}
