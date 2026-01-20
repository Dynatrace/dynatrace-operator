package bootstrapperconfig

import (
	"context"
	"encoding/json"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *SecretGenerator) preparePMC(ctx context.Context, dk *dynakube.DynaKube) ([]byte, error) {
	var pmConfig *dtclient.ProcessModuleConfig

	if !conditions.IsOutdated(s.timeProvider, dk, ConfigConditionType) {
		log.Info("skipping Dynatrace API call, trying to get ruxitagentproc content from source secret")

		sourceKey := client.ObjectKey{
			Name:      GetSourceConfigSecretName(dk.Name),
			Namespace: dk.Namespace,
		}

		targetKey := client.ObjectKey{
			Name:      consts.BootstrapperInitSecretName,
			Namespace: dk.Namespace,
		}

		source, err := k8ssecret.GetSecretFromSource(ctx, s.secrets, sourceKey, targetKey)
		if err != nil && !k8serrors.IsNotFound(err) {
			conditions.SetKubeAPIError(dk.Conditions(), ConfigConditionType, err)

			return nil, err
		} else if err == nil && source.Data[pmc.InputFileName] != nil {
			pmConfig, err = dtclient.NewProcessModuleConfig(source.Data[pmc.InputFileName])
			if err != nil {
				conditions.SetSecretGenFailed(dk.Conditions(), ConfigConditionType, err)

				log.Info("could not unmarshal process module config from source secret")

				return nil, err
			}
		}
	}

	if pmConfig == nil {
		var err error

		pmConfig, err = s.dtClient.GetProcessModuleConfig(ctx, 0)
		if err != nil {
			conditions.SetDynatraceAPIError(dk.Conditions(), ConfigConditionType, err)

			return nil, err
		}

		log.Debug("calling the Dynatrace API for ruxitagentproc content")

		conditions.SetSecretOutdated(dk.Conditions(), ConfigConditionType, "secret is outdated, update in progress")
	}

	tenantToken, err := k8ssecret.GetDataFromSecretName(ctx, s.apiReader, types.NamespacedName{
		Name:      dk.OneAgent().GetTenantSecret(),
		Namespace: dk.Namespace,
	}, connectioninfo.TenantTokenKey, log)
	if err != nil {
		conditions.SetKubeAPIError(dk.Conditions(), ConfigConditionType, err)

		return nil, err
	}

	pmConfig = pmConfig.
		AddHostGroup(dk.OneAgent().GetHostGroup()).
		AddConnectionInfo(dk.Status.OneAgent.ConnectionInfoStatus, tenantToken).
		// set proxy explicitly empty, so old proxy settings get deleted where necessary
		AddProxy("")

	if dk.NeedsOneAgentProxy() {
		log.Debug("proxy is needed")

		proxy, err := dk.Proxy(ctx, s.apiReader)
		if err != nil {
			conditions.SetKubeAPIError(dk.Conditions(), ConfigConditionType, err)

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
		conditions.SetSecretGenFailed(dk.Conditions(), ConfigConditionType, err)

		log.Info("could not marshal process module config")

		return nil, err
	}

	return marshaled, nil
}
