package kubemon

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ServiceAccountName = "dynatrace-kubernetes-monitoring"
)

// addToDashboard makes a rest call to the dynatrace api to add the activegate instance to the dashboard
// Returns the id of the entry on success or error otherwise
func (r *Reconciler) addToDashboard() (string, error) {
	serviceAccount, err := r.findServiceAccount()
	if err != nil {
		return "", err
	}

	token, err := r.findBearerTokenSecret(serviceAccount)
	if err != nil {
		return "", err
	}

	bearerToken, hasBearerToken := token.Data["token"]
	if !hasBearerToken {
		return "", fmt.Errorf("secret has no bearer token")
	}

	return r.postToApiEndpoint(bearerToken)
}

func (r *Reconciler) handleAddToDashboardResult(id string, addToDashboardErr error, log logr.Logger) {
	if id == "" {
		id = "<unset>"
	}

	if addToDashboardErr != nil {
		r.handleAddToDashboardError(id, addToDashboardErr, log)
	} else {
		log.Info("added DynaKube to Kubernetes dashboard", "id", id)
	}
}

func (r *Reconciler) handleAddToDashboardError(id string, addToDashboardErr error, log logr.Logger) {
	if serverError, isServerError := addToDashboardErr.(dtclient.ServerError); isServerError {
		r.handleAddToDashboardServerError(id, serverError, log)
	} else {
		log.Error(addToDashboardErr, "error when adding DynaKube Kubernetes configuration")
	}
}

func (r *Reconciler) handleAddToDashboardServerError(id string, serverError dtclient.ServerError, log logr.Logger) {
	if serverError.Code == 400 {
		log.Info("error returned from Dynatrace API when adding DynaKube Kubernetes configuration, ignore if configuration already exist", "id", id, "error", serverError.Message)
	} else {
		log.Error(fmt.Errorf("error returned from Dynatrace API"), "error returned from Dynatrace API", "id", id, "error", serverError.Message)
	}
}

func (r *Reconciler) findServiceAccount() (*corev1.ServiceAccount, error) {
	serviceAccountName := r.instance.Spec.KubernetesMonitoringSpec.ServiceAccountName
	if serviceAccountName == "" {
		serviceAccountName = ServiceAccountName
	}

	serviceAccount := &corev1.ServiceAccount{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: r.instance.Namespace,
		Name:      serviceAccountName,
	}, serviceAccount)

	return serviceAccount, err
}

func (r *Reconciler) findBearerTokenSecret(serviceAccount *corev1.ServiceAccount) (*corev1.Secret, error) {
	if len(serviceAccount.Secrets) <= 0 {
		return nil, fmt.Errorf("could not find token name in service account secrets")
	}

	tokenName := serviceAccount.Secrets[0].Name
	if tokenName == "" {
		return nil, fmt.Errorf("bearer token name is empty")
	}

	return r.findSecret(tokenName)
}

func (r *Reconciler) postToApiEndpoint(bearerToken []byte) (string, error) {
	// The same endpoint can not be used multiple times, so use as semi-unique name
	// Remove protocol prefix, if any
	ip := r.buildAddToDashboardIp()
	sanitizedLabel := r.buildAddToDashboardLabel(ip)
	return r.dtc.AddToDashboard(sanitizedLabel, r.instance.Spec.KubernetesMonitoringSpec.KubernetesAPIEndpoint, string(bearerToken))
}

func (r *Reconciler) buildAddToDashboardLabel(ip string) string {
	label := fmt.Sprintf("%s-%s-%s", r.instance.Namespace, r.instance.Name, ip)

	// Take only words and numbers
	regex := regexp.MustCompile(`[a-zA-Z\d]+`)
	labelParts := regex.FindAllString(label, -1)

	// And join them with safe dashes
	sanitizedLabel := strings.Join(labelParts, "-")
	return sanitizedLabel
}

func (r *Reconciler) buildAddToDashboardIp() string {
	ip := strings.TrimPrefix(r.instance.Spec.KubernetesMonitoringSpec.KubernetesAPIEndpoint, "https://")
	ip = strings.TrimPrefix(ip, "http://")
	ip = strings.ReplaceAll(ip, ":", "_")
	return ip
}

func (r *Reconciler) findSecret(tokenName string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: r.instance.Namespace,
		Name:      tokenName,
	}, secret)
	return secret, err
}
