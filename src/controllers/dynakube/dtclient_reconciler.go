package dynakube

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DynatraceClientReconciler struct {
	Client                               client.Client
	DynatraceClientFunc                  DynatraceClientFunc
	Now                                  metav1.Time
	ApiToken, PaasToken, DataIngestToken string
	ValidTokens                          bool
	dkName, ns, secretKey                string
	status                               *dynatracev1beta1.DynaKubeStatus
}

type tokenConfig struct {
	Type           string
	Key, Value     string
	Scopes         []string
	OptionalScopes []string
	Timestamp      **metav1.Time
}

func (r *DynatraceClientReconciler) Reconcile(ctx context.Context, instance *dynatracev1beta1.DynaKube) (dtclient.Client, bool, error) {
	r.ValidTokens = true
	if r.Now.IsZero() {
		r.Now = metav1.Now()
	}

	dtf := r.DynatraceClientFunc
	if dtf == nil {
		dtf = BuildDynatraceClient
	}

	r.status = &instance.Status
	r.ns = instance.GetNamespace()
	r.dkName = instance.GetName()
	secretName := instance.Tokens()
	updateCR := false

	r.secretKey = r.ns + ":" + secretName
	secret, err := r.getSecret(ctx, instance)
	r.setTokens(secret)
	if k8serrors.IsNotFound(err) {
		message := fmt.Sprintf("Secret '%s' not found", r.secretKey)

		updateCR = r.setAndLogCondition(&r.status.Conditions, metav1.Condition{
			Type:    dynatracev1beta1.APITokenConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  dynatracev1beta1.ReasonTokenSecretNotFound,
			Message: message,
		}) || updateCR
		updateCR = r.removePaaSTokenCondition() || updateCR

		return nil, updateCR, nil
	} else if err != nil {
		return nil, updateCR, err
	}

	if r.ApiToken == "" {
		msg := fmt.Sprintf("Token %s on secret %s missing", dtclient.DynatraceApiToken, r.secretKey)
		updateCR = r.setAndLogCondition(&r.status.Conditions, metav1.Condition{
			Type:    dynatracev1beta1.APITokenConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  dynatracev1beta1.ReasonTokenMissing,
			Message: msg,
		}) || updateCR
		updateCR = r.removePaaSTokenCondition() || updateCR

		return nil, updateCR, nil
	}

	dtc, err := dtf(DynatraceClientProperties{
		ApiReader:           r.Client,
		Secret:              secret,
		Proxy:               convertProxy(instance.Spec.Proxy),
		ApiUrl:              instance.Spec.APIURL,
		Namespace:           r.ns,
		NetworkZone:         instance.Spec.NetworkZone,
		TrustedCerts:        instance.Spec.TrustedCAs,
		SkipCertCheck:       instance.Spec.SkipCertCheck,
		DisableHostRequests: instance.FeatureDisableHostsRequests(),
	})

	if err != nil {
		message := fmt.Sprintf("Failed to create Dynatrace API Client: %s", err)

		updateCR = r.setAndLogCondition(&r.status.Conditions, metav1.Condition{
			Type:    dynatracev1beta1.APITokenConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  dynatracev1beta1.ReasonTokenMissing,
			Message: message,
		}) || updateCR
		updateCR = r.removePaaSTokenCondition() || updateCR

		return nil, updateCR, err
	}

	var tokens []tokenConfig
	if r.PaasToken == "" {
		tokens = []tokenConfig{{
			Type:      dynatracev1beta1.APITokenConditionType,
			Key:       dtclient.DynatraceApiToken,
			Value:     r.ApiToken,
			Scopes:    []string{dtclient.TokenScopeInstallerDownload},
			Timestamp: &r.status.LastAPITokenProbeTimestamp,
		}}
		updateCR = r.removePaaSTokenCondition() || updateCR
	} else {
		tokens = []tokenConfig{
			{
				Type:      dynatracev1beta1.APITokenConditionType,
				Key:       dtclient.DynatraceApiToken,
				Value:     r.ApiToken,
				Scopes:    []string{},
				Timestamp: &r.status.LastAPITokenProbeTimestamp,
			},
			{
				Type:      dynatracev1beta1.PaaSTokenConditionType,
				Key:       dtclient.DynatracePaasToken,
				Value:     r.PaasToken,
				Scopes:    []string{dtclient.TokenScopeInstallerDownload},
				Timestamp: &r.status.LastPaaSTokenProbeTimestamp,
			}}
	}
	if !instance.FeatureDisableHostsRequests() {
		tokens[0].Scopes = append(tokens[0].Scopes, dtclient.TokenScopeDataExport)
	}

	if instance.KubernetesMonitoringMode() &&
		instance.FeatureAutomaticKubernetesApiMonitoring() {
		tokens[0].Scopes = append(tokens[0].Scopes,
			dtclient.TokenScopeEntitiesRead,
			dtclient.TokenScopeSettingsRead,
			dtclient.TokenScopeSettingsWrite)
	}

	if r.DataIngestToken != "" {
		tokens = append(tokens, tokenConfig{
			Type:      dynatracev1beta1.DataIngestTokenConditionType,
			Key:       dtclient.DynatraceDataIngestToken,
			Value:     r.DataIngestToken,
			Scopes:    []string{dtclient.TokenScopeMetricsIngest},
			Timestamp: &r.status.LastDataIngestTokenProbeTimestamp,
		})
	}

	tokens[0].OptionalScopes = append(tokens[0].Scopes, dtclient.TokenScopeActiveGateTokenCreate)

	for _, token := range tokens {
		updateCR = r.CheckToken(dtc, token) || updateCR
	}

	return dtc, updateCR, nil
}

func (r *DynatraceClientReconciler) CheckToken(dtc dtclient.Client, token tokenConfig) bool {
	if strings.TrimSpace(token.Value) != token.Value {
		return r.setAndLogCondition(&r.status.Conditions, metav1.Condition{
			Type:    token.Type,
			Status:  metav1.ConditionFalse,
			Reason:  dynatracev1beta1.ReasonTokenUnauthorized,
			Message: fmt.Sprintf("Token on secret %s has leading and/or trailing spaces", r.secretKey),
		})
	}

	// At this point, we can query the Dynatrace API to verify whether our tokens are correct. To avoid excessive requests,
	// we wait at least 5 mins between proves.
	if *token.Timestamp != nil && r.Now.Time.Before((*token.Timestamp).Add(5*time.Minute)) {
		oldCondition := meta.FindStatusCondition(r.status.Conditions, token.Type)
		if oldCondition.Reason != dynatracev1beta1.ReasonTokenReady {
			r.ValidTokens = false
		}
		return false
	}

	nowCopy := r.Now
	*token.Timestamp = &nowCopy
	ss, err := dtc.GetTokenScopes(token.Value)

	var serr dtclient.ServerError
	if ok := errors.As(err, &serr); ok && serr.Code == http.StatusUnauthorized {
		r.setAndLogCondition(&r.status.Conditions, metav1.Condition{
			Type:    token.Type,
			Status:  metav1.ConditionFalse,
			Reason:  dynatracev1beta1.ReasonTokenUnauthorized,
			Message: fmt.Sprintf("Token on secret %s unauthorized", r.secretKey),
		})
		return true
	}

	if err != nil {
		r.setAndLogCondition(&r.status.Conditions, metav1.Condition{
			Type:    token.Type,
			Status:  metav1.ConditionFalse,
			Reason:  dynatracev1beta1.ReasonTokenError,
			Message: fmt.Sprintf("error when querying token on secret %s: %v", r.secretKey, err),
		})
		return true
	}

	missingScopes := make([]string, 0)
	for _, s := range token.Scopes {
		if !ss.Contains(s) {
			missingScopes = append(missingScopes, s)
		}
	}

	if len(missingScopes) > 0 {
		r.setAndLogCondition(&r.status.Conditions, metav1.Condition{
			Type:    token.Type,
			Status:  metav1.ConditionFalse,
			Reason:  dynatracev1beta1.ReasonTokenScopeMissing,
			Message: fmt.Sprintf("Token on secret %s missing scopes [%s]", r.secretKey, strings.Join(missingScopes, ", ")),
		})
		return true
	}

	r.HandleOptionalScope(token.OptionalScopes, ss)

	r.setAndLogCondition(&r.status.Conditions, metav1.Condition{
		Type:    token.Type,
		Status:  metav1.ConditionTrue,
		Reason:  dynatracev1beta1.ReasonTokenReady,
		Message: "Ready",
	})
	return true
}

func (r *DynatraceClientReconciler) removePaaSTokenCondition() bool {
	if meta.FindStatusCondition(r.status.Conditions, dynatracev1beta1.PaaSTokenConditionType) != nil {
		meta.RemoveStatusCondition(&r.status.Conditions, dynatracev1beta1.PaaSTokenConditionType)
		return true
	}
	return false
}

func (r *DynatraceClientReconciler) getSecret(ctx context.Context, instance *dynatracev1beta1.DynaKube) (*corev1.Secret, error) {
	secretName := instance.Tokens()
	ns := instance.GetNamespace()
	secret := corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ns}, &secret); err != nil {
		return nil, err
	}
	return &secret, nil
}

func (r *DynatraceClientReconciler) setTokens(secret *corev1.Secret) {
	if secret != nil {
		r.ApiToken = string(secret.Data[dtclient.DynatraceApiToken])
		r.PaasToken = string(secret.Data[dtclient.DynatracePaasToken])
		r.DataIngestToken = string(secret.Data[dtclient.DynatraceDataIngestToken])
	}
}

func (r *DynatraceClientReconciler) setAndLogCondition(conditions *[]metav1.Condition, condition metav1.Condition) bool {
	c := meta.FindStatusCondition(*conditions, condition.Type)

	if condition.Reason != dynatracev1beta1.ReasonTokenReady {
		r.ValidTokens = false
		log.Info("problem with token detected", "dynakube", r.dkName, "token", condition.Type,
			"msg", condition.Message)
	}

	if c != nil && c.Reason == condition.Reason && c.Message == condition.Message && c.Status == condition.Status {
		return false
	}

	condition.LastTransitionTime = r.Now
	meta.SetStatusCondition(conditions, condition)
	return true
}

func (r *DynatraceClientReconciler) HandleOptionalScope(optionalScopes []string, tokenScopes dtclient.TokenScopes) {
	for _, scope := range optionalScopes {
		switch scope {
		case dtclient.TokenScopeActiveGateTokenCreate:
			r.status.ActiveGate.UseAuthToken = tokenScopes.Contains(scope)
		}
	}
}

func convertProxy(proxy *dynatracev1beta1.DynaKubeProxy) *DynatraceClientProxy {
	if proxy == nil {
		return nil
	}
	return &DynatraceClientProxy{
		Value:     proxy.Value,
		ValueFrom: proxy.ValueFrom,
	}
}
