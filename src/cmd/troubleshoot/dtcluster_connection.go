package troubleshoot

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
)

func checkDTClusterConnection(troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("[dtcluster ] ")

	logNewTestf("checking if tenant is accessible ...")

	tests := []troubleshootFunc{
		checkConnection,
	}

	for _, test := range tests {
		if err := test(troubleshootCtx); err != nil {
			logErrorf(err.Error())
			return fmt.Errorf("tenant isn't  accessible")
		}
	}

	logOkf("tenant is accessible")
	return nil
}

func checkConnection(troubleshootCtx *troubleshootContext) error {
	tokenReader := token.NewReader(troubleshootCtx.apiReader, &troubleshootCtx.dynakube)
	tokens, err := tokenReader.ReadTokens(context.TODO())

	if err != nil {
		return err
	}

	dynatraceClientProperties := dynakube.NewDynatraceClientProperties(context.TODO(), troubleshootCtx.apiReader, troubleshootCtx.dynakube, tokens)
	if err != nil {
		return errorWithMessagef(err, "failed to configure DynatraceAPI client")
	}

	dtc, err := dynakube.BuildDynatraceClient(dynatraceClientProperties)
	if err != nil {
		return errorWithMessagef(err, "failed to build DynatraceAPI client")
	}

	_, err = dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return errorWithMessagef(err, "failed to connect to DynatraceAPI")
	}
	return nil
}
