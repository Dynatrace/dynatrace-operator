package activegate

import (
	"fmt"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/dao"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

// addToDashboard makes a rest call to the dynatrace api to add the activegate instance to the dashboard
// Returns the id of the entry on success or error otherwise
func (r *ReconcileActiveGate) addToDashboard(apiTokenSecret *corev1.Secret, instance *dynatracev1alpha1.ActiveGate) (string, error) {
	serviceAccount, err := dao.FindServiceAccount(r.client)
	if err != nil {
		return "", err
	}
	if serviceAccount == nil {
		return "", fmt.Errorf("could not find activegate service account")
	}
	if len(serviceAccount.Secrets) <= 0 {
		return "", fmt.Errorf("could not find token name in service account secrets")
	}

	tokenName := serviceAccount.Secrets[0].Name
	if tokenName == "" {
		return "", fmt.Errorf("bearer token name is empty")
	}

	tokenSecret, err := dao.FindBearerTokenSecret(r.client, tokenName)
	if err != nil {
		return "", err
	}
	if tokenSecret == nil {
		return "", fmt.Errorf("could not find bearer token secret")
	}

	dtc, err := r.dtcBuildFunc(r.client, instance, apiTokenSecret)
	if err != nil {
		return "", err
	}

	bearerToken, hasBearerToken := tokenSecret.Data["token"]
	if !hasBearerToken {
		return "", fmt.Errorf("secret has no bearer token")
	}

	// The same endpoint can not be used multiple times, so use as semi-unique name
	ip := strings.TrimPrefix(instance.Spec.KubernetesAPIEndpoint, "https://")
	ip = strings.TrimPrefix(ip, "http://")
	label := fmt.Sprintf("%s-%s-%s", instance.Namespace, instance.Name, ip)

	id, err := dtc.AddToDashboard(label, instance.Spec.KubernetesAPIEndpoint, string(bearerToken))
	if err != nil {
		return "", err
	}
	return id, nil
}
