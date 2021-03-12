// +build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TokenSecretName = "test-token-secret"
	keyAPIToken     = "DYNATRACE_API_TOKEN"
	keyPAASToken    = "DYNATRACE_PAAS_TOKEN"
	deleteDelay     = 10 * time.Second
)

var log = logger.NewDTLogger()

func PrepareEnvironment(client client.Client, namespace string) error {
	err := deleteNamespace(client, namespace)
	if err != nil && !k8serrors.IsNotFound(errors.Cause(err)) {
		return errors.WithStack(err)
	}

	err = createNamespace(client, namespace)
	if err != nil {
		return errors.WithStack(err)
	}

	err = createTokenSecret(client, namespace)
	if err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(deployKustomize())
}

func deployKustomize() error {
	log.Info("deploying to Kubernetes")
	return errors.WithStack(deployKustomizeKubernetes())
}

func deployKustomizeKubernetes() error {
	pathToKustomize, err := getPathToKustomize()
	if err != nil {
		return errors.WithStack(err)
	}

	err = executeApply(pathToKustomize)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func executeApply(pathToKustomize string) error {
	cmd := exec.Command("kubectl", "apply", "-k", pathToKustomize)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		outputData, err := cmd.Output()
		log.Error(err, "deployment failed", "output", string(outputData))
	}
	return errors.WithStack(err)
}

func getPathToKustomize() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", errors.WithStack(err)
	}

	workingDir = workingDir[:(strings.LastIndex(workingDir, "dynatrace-operator") + len("dynatrace-operator"))]
	pathToKustomize := fmt.Sprintf("%s/config/kubernetes/", workingDir)
	log.Info(fmt.Sprintf("assuming 'kustomization.yaml' to be in '%s'", pathToKustomize))
	if _, err := os.Stat(fmt.Sprintf("%skustomization.yaml", pathToKustomize)); err != nil {
		log.Error(err, "'kustomization.yaml' not found in path", "path", pathToKustomize)
		return "", errors.WithStack(err)
	}

	return pathToKustomize, nil
}

func deleteNamespace(clt client.Client, namespace string) error {
	namespaceToDelete := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		}}
	err := clt.Delete(context.TODO(), &namespaceToDelete)
	if err != nil {
		return errors.WithStack(err)
	}
	return waitForNamespaceDeletion(clt, namespace)
}

func waitForNamespaceDeletion(clt client.Client, namespace string) error {
	namespaceToDelete := v1.Namespace{}
	err := clt.Get(context.TODO(), client.ObjectKey{Name: namespace}, &namespaceToDelete)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return errors.WithStack(err)
	}
	time.Sleep(deleteDelay)
	return waitForNamespaceDeletion(clt, namespace)
}

func createNamespace(client client.Client, namespace string) error {
	namespaceToCreate := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		}}
	err := client.Create(context.TODO(), &namespaceToCreate)
	if err != nil {
		return errors.WithStack(err)
	}
	return err
}
