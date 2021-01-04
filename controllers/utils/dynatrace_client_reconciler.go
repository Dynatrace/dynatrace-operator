package utils

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"net/http"
	"strings"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DynatraceClientReconciler struct {
	Client              client.Client
	DynatraceClientFunc DynatraceClientFunc
	Now                 metav1.Time
	UpdatePaaSToken     bool
	UpdateAPIToken      bool
}

type tokenConfig struct {
	Type              string
	Key, Value, Scope string
	Timestamp         **metav1.Time
}

func (r *DynatraceClientReconciler) Reconcile(ctx context.Context, instance *dynatracev1alpha1.DynaKube, logger logr.Logger) (dtclient.Client, bool, error) {
	now := r.Now
	if now.IsZero() {
		now = metav1.Now()
	}

	dtf := r.DynatraceClientFunc
	if dtf == nil {
		dtf = BuildDynatraceClient
	}

	sts := instance.Status
	ns := instance.GetNamespace()
	secretName := GetTokensName(*instance)

	var tokens []*tokenConfig

	if r.UpdatePaaSToken {
		tokens = append(tokens, &tokenConfig{
			Type:      dynatracev1alpha1.PaaSTokenConditionType,
			Key:       DynatracePaasToken,
			Scope:     dtclient.TokenScopeInstallerDownload,
			Timestamp: &sts.LastPaaSTokenProbeTimestamp,
		})
	}

	if r.UpdateAPIToken {
		tokens = append(tokens, &tokenConfig{
			Type:      dynatracev1alpha1.APITokenConditionType,
			Key:       DynatraceApiToken,
			Scope:     dtclient.TokenScopeDataExport,
			Timestamp: &sts.LastAPITokenProbeTimestamp,
		})
	}

	updateCR := false

	// To migrate from our implementation for conditions on Operator v0.6-0.7 to operator-sdk's implementation.
	for i := range sts.BaseOneAgentStatus.Conditions {
		condition, err := sts.BaseOneAgentStatus.Conditions[i].GetCondition(ctx)
		if err != nil {
			logger.Error(err, "Failed to get condition")
			continue
		}
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = now
			updateCR = true
		}
	}

	secretKey := ns + ":" + secretName
	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ns}, secret); k8serrors.IsNotFound(err) {
		message := fmt.Sprintf("Secret '%s' not found", secretKey)

		for _, t := range tokens {
			updateCR = sts.BaseOneAgentStatus.Conditions.SetCondition(conditions.Condition{
				Type:    t.Type,
				Status:  corev1.ConditionFalse,
				Reason:  dynatracev1alpha1.ReasonTokenSecretNotFound,
				Message: message,
			}) || updateCR
		}

		return nil, updateCR, fmt.Errorf(message)
	} else if err != nil {
		return nil, updateCR, err
	}

	valid := true

	for _, t := range tokens {
		v := secret.Data[t.Key]
		if len(v) == 0 {
			updateCR = sts.BaseOneAgentStatus.Conditions.SetCondition(dynatracev1alpha1.Condition{
				Type:    t.Type,
				Status:  corev1.ConditionFalse,
				Reason:  dynatracev1alpha1.ReasonTokenMissing,
				Message: fmt.Sprintf("Token %s on secret %s missing", t.Key, secretKey),
			}) || updateCR
			valid = false
		}
		t.Value = string(v)
	}

	if !valid {
		return nil, updateCR, fmt.Errorf("issues found with tokens, see status")
	}

	dtc, err := dtf(r.Client, instance, r.UpdateAPIToken, r.UpdatePaaSToken)
	if err != nil {
		message := fmt.Sprintf("Failed to create Dynatrace API Client: %s", err)

		for _, t := range tokens {
			updateCR = sts.BaseOneAgentStatus.Conditions.SetCondition(dynatracev1alpha1.Condition{
				Type:    t.Type,
				Status:  corev1.ConditionFalse,
				Reason:  dynatracev1alpha1.ReasonTokenError,
				Message: message,
			}) || updateCR
		}

		return nil, updateCR, err
	}

	for _, t := range tokens {
		if strings.TrimSpace(t.Value) != t.Value {
			updateCR = sts.BaseOneAgentStatus.Conditions.SetCondition(dynatracev1alpha1.Condition{
				Type:    t.Type,
				Status:  corev1.ConditionFalse,
				Reason:  dynatracev1alpha1.ReasonTokenUnauthorized,
				Message: fmt.Sprintf("Token on secret %s has leading and/or trailing spaces", secretKey),
			}) || updateCR
			continue
		}

		// At this point, we can query the Dynatrace API to verify whether our tokens are correct. To avoid excessive requests,
		// we wait at least 5 mins between proves.
		if *t.Timestamp != nil && now.Time.Before((*t.Timestamp).Add(5*time.Minute)) {
			continue
		}

		nowCopy := now
		*t.Timestamp = &nowCopy
		updateCR = true
		ss, err := dtc.GetTokenScopes(t.Value)

		var serr dtclient.ServerError
		if ok := errors.As(err, &serr); ok && serr.Code == http.StatusUnauthorized {
			sts.BaseOneAgentStatus.Conditions.SetCondition(dynatracev1alpha1.Condition{
				Type:    t.Type,
				Status:  corev1.ConditionFalse,
				Reason:  dynatracev1alpha1.ReasonTokenUnauthorized,
				Message: fmt.Sprintf("Token on secret %s unauthorized", secretKey),
			})
			continue
		}

		if err != nil {
			sts.BaseOneAgentStatus.Conditions.SetCondition(dynatracev1alpha1.Condition{
				Type:    t.Type,
				Status:  corev1.ConditionFalse,
				Reason:  dynatracev1alpha1.ReasonTokenError,
				Message: fmt.Sprintf("error when querying token on secret %s: %v", secretKey, err),
			})
			continue
		}

		if !ss.Contains(t.Scope) {
			sts.BaseOneAgentStatus.Conditions.SetCondition(dynatracev1alpha1.Condition{
				Type:    t.Type,
				Status:  corev1.ConditionFalse,
				Reason:  dynatracev1alpha1.ReasonTokenScopeMissing,
				Message: fmt.Sprintf("Token on secret %s missing scope %s", secretKey, t.Scope),
			})
			continue
		}

		if t.Key == DynatracePaasToken {
			ci, err := dtc.GetConnectionInfo()
			if err != nil {
				sts.BaseOneAgentStatus.Conditions.SetCondition(dynatracev1alpha1.Condition{
					Type:    t.Type,
					Status:  corev1.ConditionFalse,
					Reason:  dynatracev1alpha1.ReasonTokenError,
					Message: fmt.Sprintf("error when connection info with token on secret %s: %v", secretKey, err),
				})
				continue
			}

			sts.OneAgentStatus.EnvironmentID = ci.TenantUUID
		}

		sts.BaseOneAgentStatus.Conditions.SetCondition(dynatracev1alpha1.Condition{
			Type:    t.Type,
			Status:  corev1.ConditionTrue,
			Reason:  dynatracev1alpha1.ReasonTokenReady,
			Message: "Ready",
		})
	}

	return dtc, updateCR, nil
}
