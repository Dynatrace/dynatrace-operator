package utils

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DynatracePaasToken = "paasToken"
	DynatraceApiToken  = "apiToken"
)

// GetDeployment returns the Deployment object who is the owner of this pod.
func GetDeployment(c client.Client, ns string) (*appsv1.Deployment, error) {
	var pod corev1.Pod
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		return nil, errors.New("POD_NAME environment variable does not exist")
	}

	err := c.Get(context.TODO(), client.ObjectKey{Name: podName, Namespace: ns}, &pod)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rsOwner := metav1.GetControllerOf(&pod)
	if rsOwner == nil {
		return nil, errors.Errorf("no controller found for Pod: %s", pod.Name)
	} else if rsOwner.Kind != "ReplicaSet" {
		return nil, errors.Errorf("unexpected controller found for Pod: %s, kind: %s", pod.Name, rsOwner.Kind)
	}

	var rs appsv1.ReplicaSet
	if err := c.Get(context.TODO(), client.ObjectKey{Name: rsOwner.Name, Namespace: ns}, &rs); err != nil {
		return nil, errors.WithStack(err)
	}

	dOwner := metav1.GetControllerOf(&rs)
	if dOwner == nil {
		return nil, errors.Errorf("no controller found for ReplicaSet: %s", pod.Name)
	} else if dOwner.Kind != "Deployment" {
		return nil, errors.Errorf("unexpected controller found for ReplicaSet: %s, kind: %s", pod.Name, dOwner.Kind)
	}

	var d appsv1.Deployment
	if err := c.Get(context.TODO(), client.ObjectKey{Name: dOwner.Name, Namespace: ns}, &d); err != nil {
		return nil, errors.WithStack(err)
	}
	return &d, nil
}

// CreateOrUpdateSecretIfNotExists creates a secret in case it does not exist or updates it if there are changes
func CreateOrUpdateSecretIfNotExists(c client.Client, r client.Reader, secretName string, targetNS string, data map[string][]byte, secretType corev1.SecretType, log logr.Logger) error {
	var cfg corev1.Secret
	err := r.Get(context.TODO(), client.ObjectKey{Name: secretName, Namespace: targetNS}, &cfg)
	if k8serrors.IsNotFound(err) {
		log.Info("Creating OneAgent config secret")
		if err := c.Create(context.TODO(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: targetNS,
			},
			Type: secretType,
			Data: data,
		}); err != nil {
			return errors.Wrapf(err, "failed to create secret %s", secretName)
		}
		return nil
	}

	if err != nil {
		return errors.Wrapf(err, "failed to query for secret %s", secretName)
	}

	if !reflect.DeepEqual(data, cfg.Data) {
		log.Info(fmt.Sprintf("Updating secret %s", secretName))
		cfg.Data = data
		if err := c.Update(context.TODO(), &cfg); err != nil {
			return errors.Wrapf(err, "failed to update secret %s", secretName)
		}
	}

	return nil
}

func GetField(values map[string]string, key, defaultValue string) string {
	if values == nil {
		return defaultValue
	}
	if x := values[key]; x != "" {
		return x
	}
	return defaultValue
}

// CheckIfOneAgentAPMExists checks if a OneAgentAPM object exists
func CheckIfOneAgentAPMExists(cfg *rest.Config) (bool, error) {
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}
	_, resourceList, err := client.ServerGroupsAndResources()
	if err != nil {
		return false, err
	}

	for _, resource := range resourceList {
		for _, apiResource := range resource.APIResources {
			if apiResource.Kind == "OneAgentAPM" {
				return true, nil
			}
		}
	}
	return false, nil
}
