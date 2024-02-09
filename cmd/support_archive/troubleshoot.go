package support_archive

import (
	"bytes"
	"context"

	"github.com/Dynatrace/dynatrace-operator/cmd/troubleshoot"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const troubleshootCollectorName = "troubleshoot"
const TroublshootOutputFileName = "troubleshoot.txt"

type troubleshootCollector struct {
	collectorCommon

	context    context.Context
	apiReader  client.Reader
	kubeConfig rest.Config
	namespace  string
}

func newTroubleshootCollector(context context.Context, log logger.DtLogger, supportArchive archiver, namespace string, apiReader client.Reader, kubeConfig rest.Config) collector { //nolint:revive // argument-limit doesn't apply to constructors
	return troubleshootCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		context:    context,
		apiReader:  apiReader,
		kubeConfig: kubeConfig,
		namespace:  namespace,
	}
}

func (t troubleshootCollector) Name() string {
	return troubleshootCollectorName
}

func (t troubleshootCollector) Do() error {
	logInfof(t.log, "Running troubleshoot command and storing output into %s", TroublshootOutputFileName)

	troubleshootCmdOutput := bytes.Buffer{}
	log := troubleshoot.NewTroubleshootLoggerToWriter(&troubleshootCmdOutput)

	troubleshoot.RunTroubleshootCmd(context.Background(), log, t.namespace, &t.kubeConfig)

	t.supportArchive.addFile(TroublshootOutputFileName, &troubleshootCmdOutput)

	return nil
}
