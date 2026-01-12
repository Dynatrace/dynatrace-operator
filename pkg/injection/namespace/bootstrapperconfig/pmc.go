package bootstrapperconfig

import (
	"context"
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"k8s.io/apimachinery/pkg/types"
)

func (s *SecretGenerator) preparePMC(ctx context.Context, dk *dynakube.DynaKube) ([]byte, error) {
	log.Debug("calling the Dynatrace API for ruxitagentproc content")

	conditions.SetSecretOutdated(dk.Conditions(), ConfigConditionType, "secret is outdated, update in progress")

	pmc, err := s.dtClient.GetProcessModuleConfig(ctx, 0)
	if err != nil {
		conditions.SetDynatraceAPIError(dk.Conditions(), ConfigConditionType, err)

		return nil, err
	}

	tenantToken, err := k8ssecret.GetDataFromSecretName(ctx, s.apiReader, types.NamespacedName{
		Name:      dk.OneAgent().GetTenantSecret(),
		Namespace: dk.Namespace,
	}, connectioninfo.TenantTokenKey, log)
	if err != nil {
		conditions.SetKubeAPIError(dk.Conditions(), ConfigConditionType, err)

		return nil, err
	}

	pmc = pmc.
		AddHostGroup(dk.OneAgent().GetHostGroup()).
		AddConnectionInfo(dk.Status.OneAgent.ConnectionInfo, tenantToken).
		// set proxy explicitly empty, so old proxy settings get deleted where necessary
		AddProxy("")

	if dk.NeedsOneAgentProxy() {
		log.Debug("proxy is needed")

		proxy, err := dk.Proxy(ctx, s.apiReader)
		if err != nil {
			conditions.SetKubeAPIError(dk.Conditions(), ConfigConditionType, err)

			return nil, err
		}

		pmc.AddProxy(proxy)

		dnsEntry := capability.BuildHostEntries(*dk)

		if dk.FF().GetNoProxy() != "" {
			dnsEntry += "," + dk.FF().GetNoProxy()
		}

		pmc.AddNoProxy(dnsEntry)
	}

	pmc.SortPropertiesByKey()

	marshaled, err := json.Marshal(pmc)
	if err != nil {
		conditions.SetSecretGenFailed(dk.Conditions(), ConfigConditionType, err)

		log.Info("could not marshal process module config")

		return nil, err
	}

	return marshaled, nil
}
