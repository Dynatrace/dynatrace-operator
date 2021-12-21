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
	Client              client.Client
	DynatraceClientFunc DynatraceClientFunc
	Now                 metav1.Time
	ApiToken, PaasToken string
	ValidTokens         bool
}

type tokenConfig struct {
	Type              string
	Key, Value, Scope string
	Timestamp         **metav1.Time
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

	sts := &instance.Status
	ns := instance.GetNamespace()
	dkName := instance.GetName()
	secretName := instance.Tokens()

	paasTokenConf := &tokenConfig{
		Type:      dynatracev1beta1.PaaSTokenConditionType,
		Key:       dtclient.DynatracePaasToken,
		Scope:     dtclient.TokenScopeInstallerDownload,
		Timestamp: &sts.LastPaaSTokenProbeTimestamp,
	}
	apiTokenConf := &tokenConfig{
		Type:      dynatracev1beta1.APITokenConditionType,
		Key:       dtclient.DynatraceApiToken,
		Scope:     dtclient.TokenScopeDataExport,
		Timestamp: &sts.LastAPITokenProbeTimestamp,
	}
	tokens := []*tokenConfig{apiTokenConf, paasTokenConf}

	updateCR := false

	// To migrate from our implementation for conditions on Operator v0.6-0.7 to operator-sdk's implementation.
	for i := range sts.Conditions {
		if sts.Conditions[i].LastTransitionTime.IsZero() {
			sts.Conditions[i].LastTransitionTime = r.Now
			updateCR = true
		}
	}

	secretKey := ns + ":" + secretName
	secret, err := r.getSecret(ctx, instance)
	r.setTokens(secret)
	if k8serrors.IsNotFound(err) {
		message := fmt.Sprintf("Secret '%s' not found", secretKey)

		for _, t := range tokens {
			updateCR = r.setAndLogCondition(&sts.Conditions, metav1.Condition{
				Type:    t.Type,
				Status:  metav1.ConditionFalse,
				Reason:  dynatracev1beta1.ReasonTokenSecretNotFound,
				Message: message,
			}, dkName) || updateCR
		}

		r.ValidTokens = false
		return nil, updateCR, nil
	} else if err != nil {
		return nil, updateCR, err
	}

	valid := true
	updateSecret := false

	if r.ApiToken == "" {
		updateCR = r.setAndLogCondition(&sts.Conditions, metav1.Condition{
			Type:    apiTokenConf.Type,
			Status:  metav1.ConditionFalse,
			Reason:  dynatracev1beta1.ReasonTokenMissing,
			Message: fmt.Sprintf("Token %s on secret %s missing", apiTokenConf.Key, secretKey),
		}, dkName) || updateCR
		valid = false
	} else if r.PaasToken == "" {
		// if paas token is missing api token is paas token
		r.PaasToken = r.ApiToken
		secret.Data[dtclient.DynatracePaasToken] = []byte(r.PaasToken)
		updateSecret = true
	}

	if r.PaasToken == "" {
		updateCR = r.setAndLogCondition(&sts.Conditions, metav1.Condition{
			Type:    paasTokenConf.Type,
			Status:  metav1.ConditionFalse,
			Reason:  dynatracev1beta1.ReasonTokenMissing,
			Message: fmt.Sprintf("Token %s on secret %s missing", paasTokenConf.Key, secretKey),
		}, dkName) || updateCR
		valid = false
	}

	paasTokenConf.Value = r.PaasToken
	apiTokenConf.Value = r.ApiToken

	if !valid {
		return nil, updateCR, fmt.Errorf("issues found with tokens, see status")
	}

	dtc, err := dtf(DynatraceClientProperties{
		ApiReader:           r.Client,
		Secret:              secret,
		Proxy:               convertProxy(instance.Spec.Proxy),
		ApiUrl:              instance.Spec.APIURL,
		Namespace:           ns,
		NetworkZone:         instance.Spec.NetworkZone,
		TrustedCerts:        instance.Spec.TrustedCAs,
		SkipCertCheck:       instance.Spec.SkipCertCheck,
		DisableHostRequests: instance.FeatureDisableHostsRequests(),
	})

	if err != nil {
		message := fmt.Sprintf("Failed to create Dynatrace API Client: %s", err)

		for _, t := range tokens {
			updateCR = r.setAndLogCondition(&sts.Conditions, metav1.Condition{
				Type:    t.Type,
				Status:  metav1.ConditionFalse,
				Reason:  dynatracev1beta1.ReasonTokenError,
				Message: message,
			}, dkName) || updateCR
		}

		return nil, updateCR, err
	}

	for _, t := range tokens {
		if strings.TrimSpace(t.Value) != t.Value {
			updateCR = r.setAndLogCondition(&sts.Conditions, metav1.Condition{
				Type:    t.Type,
				Status:  metav1.ConditionFalse,
				Reason:  dynatracev1beta1.ReasonTokenUnauthorized,
				Message: fmt.Sprintf("Token on secret %s has leading and/or trailing spaces", secretKey),
			}, dkName) || updateCR
			continue
		}

		// At this point, we can query the Dynatrace API to verify whether our tokens are correct. To avoid excessive requests,
		// we wait at least 5 mins between proves.
		if *t.Timestamp != nil && r.Now.Time.Before((*t.Timestamp).Add(5*time.Minute)) {
			oldCondition := meta.FindStatusCondition(sts.Conditions, t.Type)
			if oldCondition.Reason != dynatracev1beta1.ReasonTokenReady {
				r.ValidTokens = false
			}
			continue
		}

		nowCopy := r.Now
		*t.Timestamp = &nowCopy
		updateCR = true
		ss, err := dtc.GetTokenScopes(t.Value)

		var serr dtclient.ServerError
		if ok := errors.As(err, &serr); ok && serr.Code == http.StatusUnauthorized {
			r.setAndLogCondition(&sts.Conditions, metav1.Condition{
				Type:    t.Type,
				Status:  metav1.ConditionFalse,
				Reason:  dynatracev1beta1.ReasonTokenUnauthorized,
				Message: fmt.Sprintf("Token on secret %s unauthorized", secretKey),
			}, dkName)
			continue
		}

		if err != nil {
			r.setAndLogCondition(&sts.Conditions, metav1.Condition{
				Type:    t.Type,
				Status:  metav1.ConditionFalse,
				Reason:  dynatracev1beta1.ReasonTokenError,
				Message: fmt.Sprintf("error when querying token on secret %s: %v", secretKey, err),
			}, dkName)
			continue
		}

		if !ss.Contains(t.Scope) {
			r.setAndLogCondition(&sts.Conditions, metav1.Condition{
				Type:    t.Type,
				Status:  metav1.ConditionFalse,
				Reason:  dynatracev1beta1.ReasonTokenScopeMissing,
				Message: fmt.Sprintf("Token on secret %s missing scope %s", secretKey, t.Scope),
			}, dkName)
			continue
		}

		r.setAndLogCondition(&sts.Conditions, metav1.Condition{
			Type:    t.Type,
			Status:  metav1.ConditionTrue,
			Reason:  dynatracev1beta1.ReasonTokenReady,
			Message: "Ready",
		}, dkName)
	}

	if updateSecret {
		if err := r.Client.Update(ctx, secret); err != nil {
			return dtc, updateCR, fmt.Errorf("failed to update secret: %s", err)
		}
	}

	return dtc, updateCR, nil
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
	}
}

func (r *DynatraceClientReconciler) setAndLogCondition(conditions *[]metav1.Condition, condition metav1.Condition, dkName string) bool {
	c := meta.FindStatusCondition(*conditions, condition.Type)

	if condition.Reason != dynatracev1beta1.ReasonTokenReady {
		r.ValidTokens = false
		log.Info("problem with token detected", "dynakube", dkName, "token", condition.Type,
			"msg", condition.Message)
	}

	if c != nil && c.Reason == condition.Reason && c.Message == condition.Message && c.Status == condition.Status {
		return false
	}

	meta.SetStatusCondition(conditions, condition)
	return true
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
