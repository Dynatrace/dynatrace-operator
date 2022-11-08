package troubleshoot

import (
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

func checkDTClusterConnection(troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("[dtcluster ] ", true)

	logNewTestf("checking if tenant is accessible ...")

	tests := []troubleshootFunc{
		checkConnection,
	}

	for _, test := range tests {
		err := test(troubleshootCtx)

		if err != nil {
			logErrorf(err.Error())
			return errors.New("tenant isn't  accessible")
		}
	}

	logOkf("tenant is accessible")
	return nil
}

func checkConnection(troubleshootCtx *troubleshootContext) error {
	dynatraceClientProperties, err := dynakube.NewDynatraceClientProperties(troubleshootCtx.context, troubleshootCtx.apiReader, troubleshootCtx.dynakube)
	if err != nil {
		return errorWithMessagef(err, "failed to configure DynatraceAPI client")
	}

	dtc, err := dynakube.BuildDynatraceClient(*dynatraceClientProperties)
	if err != nil {
		return errorWithMessagef(err, "failed to build DynatraceAPI client")
	}

	_, err = dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return errorWithMessagef(err, "failed to connect to DynatraceAPI")
	}
	return nil
}
