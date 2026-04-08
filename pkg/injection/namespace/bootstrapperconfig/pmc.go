package bootstrapperconfig

import (
	"context"
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"k8s.io/apimachinery/pkg/types"
)

func (s *SecretGenerator) preparePMC(ctx context.Context, dk *dynakube.DynaKube) ([]byte, error) {
	pmConfig, err := s.dtClient.GetProcessModuleConfig(ctx)
	if err != nil {
		k8sconditions.SetDynatraceAPIError(dk.Conditions(), ConfigConditionType, err)

		return nil, err
	}

	tenantToken, err := k8ssecret.GetDataFromSecretName(ctx, s.apiReader, types.NamespacedName{
		Name:      dk.OneAgent().GetTenantSecret(),
		Namespace: dk.Namespace,
	}, connectioninfo.TenantTokenKey, log)
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), ConfigConditionType, err)

		return nil, err
	}

	pmConfig = pmConfig.
		AddHostGroup(dk.OneAgent().GetHostGroup()).
		AddConnectionInfo(dk.Status.OneAgent.ConnectionInfo, tenantToken).
		// set proxy explicitly empty, so old proxy settings get deleted where necessary
		AddProxy("")

	if dk.NeedsOneAgentProxy() {
		log.Debug("proxy is needed")

		proxy, err := dk.Proxy(ctx, s.apiReader)
		if err != nil {
			k8sconditions.SetKubeAPIError(dk.Conditions(), ConfigConditionType, err)

			return nil, err
		}

		pmConfig.AddProxy(proxy)

		dnsEntry := capability.BuildHostEntries(*dk)

		if dk.FF().GetNoProxy() != "" {
			dnsEntry += "," + dk.FF().GetNoProxy()
		}

		pmConfig.AddNoProxy(dnsEntry)
	}

	pmConfig.SortPropertiesByKey()

	marshaled, err := json.Marshal(pmConfig)
	if err != nil {
		k8sconditions.SetSecretGenFailed(dk.Conditions(), ConfigConditionType, err)

		log.Info("could not marshal process module config")

		return nil, err
	}

	return marshaled, nil
}
