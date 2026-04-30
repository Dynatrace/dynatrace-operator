package logmonsettings

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/tenant/optionalscope"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
)

type Reconciler struct {
	timeProvider *timeprovider.Provider
}

func NewReconciler() *Reconciler {
	return &Reconciler{
		timeProvider: timeprovider.New(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dtClient settings.Client, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "logmonitoring-settings")

	if !dk.LogMonitoring().IsEnabled() {
		_ = meta.RemoveStatusCondition(dk.Conditions(), ConditionType)

		return nil
	}

	if !k8sconditions.IsOutdated(r.timeProvider, dk, ConditionType) {
		return nil
	}

	_ = meta.RemoveStatusCondition(dk.Conditions(), ConditionType)

	if !dk.Status.APIToken.Platform {
		hasReadScope := optionalscope.IsAvailable(dk, token.ScopeSettingsRead)
		hasWriteScope := optionalscope.IsAvailable(dk, token.ScopeSettingsWrite)

		var missingScopes []string
		if !hasReadScope {
			missingScopes = append(missingScopes, token.ScopeSettingsRead)
		}

		if !hasWriteScope {
			missingScopes = append(missingScopes, token.ScopeSettingsWrite)
		}

		if len(missingScopes) > 0 {
			message := strings.Join(missingScopes, ", ") + " scope(s) missing: cannot query existing log monitoring setting and/or safely create new one."
			k8sconditions.SetOptionalScopeMissing(dk.Conditions(), ConditionType, message)
			log.Info(message)

			return nil
		}

		log.Info("necessary scopes for logmonitoring settings creation is available, proceeding with reconciliation")
	}

	err := r.checkLogMonitoringSettings(ctx, dtClient, dk)
	if err != nil {
		if !core.IsForbidden(err) {
			return err
		}

		log.Info("skipping reconciliation: tenant requires additional scopes for managing log monitoring settings")

		if dk.Status.APIToken.Platform {
			message := "platform token scope(s) missing: cannot query existing log monitoring monitoring setting and/or safely create new one."
			k8sconditions.SetOptionalScopeMissing(dk.Conditions(), ConditionType, message)
		}
	}

	return nil
}

func (r *Reconciler) checkLogMonitoringSettings(ctx context.Context, dtClient settings.Client, dk *dynakube.DynaKube) error {
	log := logd.FromContext(ctx)
	log.Info("start reconciling log monitoring settings")

	if dk.Status.KubernetesClusterMEID == "" {
		msg := "kubernetesClusterMEID is not available, which is needed for logmonitoring settings creation, will skip it for now"
		log.Info(msg)

		setSkippedCondition(dk.Conditions(), msg)

		return nil
	}

	logMonitoringSettings, err := dtClient.GetSettingsForLogModule(ctx, dk.Status.KubernetesClusterMEID)
	if err != nil {
		setErrorCondition(dk.Conditions())

		return errors.WithMessage(err, "error trying to check if setting exists")
	}

	if logMonitoringSettings.TotalCount > 0 {
		log.Info("there are already settings", "settings", logMonitoringSettings)

		setExistsCondition(dk.Conditions())

		return nil
	}

	matchers := []logmonitoring.IngestRuleMatchers{}
	if len(dk.LogMonitoring().IngestRuleMatchers) > 0 {
		matchers = dk.LogMonitoring().IngestRuleMatchers
	}

	objectID, err := dtClient.CreateLogMonitoringSetting(ctx, dk.Status.KubernetesClusterMEID, dk.Status.KubernetesClusterName, matchers)
	if err != nil {
		setErrorCondition(dk.Conditions())

		return err
	}

	setExistsCondition(dk.Conditions())
	log.Info("log monitoring setting created", "settings", objectID)

	return nil
}
