package troubleshoot

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

func checkDTClusterConnection(troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("[dtcluster ] ")

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
	tokenReader := token.NewReader(troubleshootCtx.apiReader, &troubleshootCtx.dynakube)
	tokens, err := tokenReader.ReadTokens(context.TODO())

	if err != nil {
		return err
	}

	dtc, err := dynatraceclient.NewBuilder(troubleshootCtx.apiReader).
		SetContext(troubleshootCtx.context).
		SetDynakube(troubleshootCtx.dynakube).
		SetTokens(tokens).
		Build()

	if err != nil {
		return errorWithMessagef(err, "failed to build DynatraceAPI client")
	}

	_, err = dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return errorWithMessagef(err, "failed to connect to DynatraceAPI")
	}
	return nil
}
