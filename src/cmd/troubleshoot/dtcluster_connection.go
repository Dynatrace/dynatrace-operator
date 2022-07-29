package troubleshoot

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"k8s.io/apimachinery/pkg/types"
)

func checkDTClusterConnection(troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("[dtcluster ] ")

	logNewTestf("checking if tenant is accessible ...")

	query := kubeobjects.NewDynakubeQuery(nil, troubleshootCtx.apiReader, troubleshootCtx.namespaceName).WithContext(context.TODO())
	dk, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakubeName})
	if err != nil {
		logWithErrorf(err, "selected '%s:%s' Dynakube does not exist", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
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
