package activegate

import (
	"fmt"
	"regexp"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/dao"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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
	// Remove protocol prefix, if any
	ip := strings.TrimPrefix(instance.Spec.KubernetesAPIEndpoint, "https://")
	ip = strings.TrimPrefix(ip, "http://")
	ip = strings.ReplaceAll(ip, ":", "_")
	label := fmt.Sprintf("%s-%s-%s", instance.Namespace, instance.Name, ip)

	// Take only words and numbers
	regex := regexp.MustCompile(`[a-zA-Z\d]+`)
	labelParts := regex.FindAllString(label, -1)

	// And join them with safe dashes
	sanitizedLabel := strings.Join(labelParts, "-")

	return dtc.AddToDashboard(sanitizedLabel, instance.Spec.KubernetesAPIEndpoint, string(bearerToken))
}

func (r *ReconcileActiveGate) handleAddToDashboardResult(id string, addToDashboardErr error, log logr.Logger) {
	if id == "" {
		id = "<unset>"
	}

	if addToDashboardErr != nil {
		if serverError, isServerError := addToDashboardErr.(dtclient.ServerError); isServerError {
			if serverError.Code == 400 {
				log.Info("error returned from Dynatrace API when adding ActiveGate Kubernetes configuration, ignore if configuration already exist", "id", id, "error", serverError.Message)
			} else {
				log.Error(fmt.Errorf("error returned from Dynatrace API"), "error returned from Dynatrace API", "id", id, "error", serverError.Message)
			}
		} else {
			log.Error(addToDashboardErr, "error when adding ActiveGate Kubernetes configuration")
		}
	} else {
		log.Info("added ActiveGate to Kubernetes dashboard", "id", id)
	}
}
