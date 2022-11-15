package troubleshoot

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

func checkDtClusterConnection(results ChecksResults, troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("[dtcluster ] ")

	logNewCheckf("checking if tenant is accessible ...")

	checks := getConnectionChecks()

	err := runChecks(results, troubleshootCtx, checks)
	if err != nil {
		return errors.Wrap(err, "tenant isn't  accessible")
	}

	logOkf("tenant is accessible")
	return nil
}

func getConnectionChecks() []*Check {
	return []*Check{{Do: checkConnection}}
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
		return errors.Wrap(err, "failed to build DynatraceAPI client")
	}

	_, err = dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return errors.Wrap(err, "failed to connect to DynatraceAPI")
	}
	return nil
}
