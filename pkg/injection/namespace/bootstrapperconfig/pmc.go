package bootstrapperconfig

import (
	"context"
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"k8s.io/apimachinery/pkg/types"
)

func (s *SecretGenerator) preparePMC(ctx context.Context, dk dynakube.DynaKube) ([]byte, error) {
	pmc, err := s.dtClient.GetProcessModuleConfig(ctx, 0)
	if err != nil {
		conditions.SetDynatraceApiError(dk.Conditions(), processmoduleconfigsecret.PMCConditionType, err)

		return nil, err
	}

	tenantToken, err := k8ssecret.GetDataFromSecretName(ctx, s.apiReader, types.NamespacedName{
		Name:      dk.OneAgent().GetTenantSecret(),
		Namespace: dk.Namespace,
	}, connectioninfo.TenantTokenKey, log)
	if err != nil {
		conditions.SetKubeApiError(dk.Conditions(), processmoduleconfigsecret.PMCConditionType, err)

		return nil, err
	}

	pmc = pmc.
		AddHostGroup(dk.OneAgent().GetHostGroup()).
		AddConnectionInfo(dk.Status.OneAgent.ConnectionInfoStatus, tenantToken).
		// set proxy explicitly empty, so old proxy settings get deleted where necessary
		AddProxy("")

	if dk.NeedsOneAgentProxy() {
		proxy, err := dk.Proxy(ctx, s.apiReader)
		if err != nil {
			conditions.SetKubeApiError(dk.Conditions(), processmoduleconfigsecret.PMCConditionType, err)

			return nil, err
		}

		pmc.AddProxy(proxy)

		multiCap := capability.NewMultiCapability(&dk)
		dnsEntry := capability.BuildDNSEntryPointWithoutEnvVars(dk.Name, dk.Namespace, multiCap)

		if dk.FF().GetNoProxy() != "" {
			dnsEntry += "," + dk.FF().GetNoProxy()
		}

		pmc.AddNoProxy(dnsEntry)
	}

	marshaled, err := json.Marshal(pmc)
	if err != nil {
		log.Info("could not marshal process module config")

		return nil, err
	}

	return marshaled, err
}
