package dtcsi

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrMissing is returned when there's a problem with the ConfigMap
var ErrMissing = errors.New("config map is missing or invalid")

const configMapName = "csi-code-modules-checker-map"

type Checker struct {
	client    client.Client
	logger    logr.Logger
	configMap *corev1.ConfigMap
	namespace string
}

func NewChecker(kubernetesClient client.Client, logger logr.Logger, namespace string) (*Checker, error) {
	configMap, err := loadConfigMap(kubernetesClient, logger, namespace)
	if err != nil {
		return nil, err
	}
	return &Checker{
		client:    kubernetesClient,
		logger:    logger,
		configMap: configMap,
		namespace: namespace,
	}, nil
}

// Add name of Dynakube with CodeModules enabled to ConfigMap.
// Should happen when Dynakube was created or setting was enabled.
func (c *Checker) Add(dynakube string) error {
	if c.configMap.Data == nil {
		c.configMap.Data = map[string]string{}
	}

	if _, contains := c.configMap.Data[dynakube]; !contains {
		c.logger.Info("Adding Dynakube with CodeModules enabled",
			"dynakube", dynakube)
		c.configMap.Data[dynakube] = ""
		return c.client.Update(context.TODO(), c.configMap)
	}
	return nil
}

// Remove name of Dynakube from ConfigMap.
// Should happen when Dynakube was deleted or setting was disabled.
func (c *Checker) Remove(dynakube string) error {
	if c.configMap.Data == nil {
		return ErrMissing
	}

	if _, contains := c.configMap.Data[dynakube]; contains {
		c.logger.Info("Removing Dynakube with CodeModules disabled",
			"dynakube", dynakube)
		delete(c.configMap.Data, dynakube)
		return c.client.Update(context.TODO(), c.configMap)
	}
	return nil
}

// Any checks if ConfigMap contains entries.
// If entries exist, there are Dynakubes with CodeModules enabled.
func (c *Checker) Any() bool {
	if c.configMap.Data == nil {
		return false
	}

	c.logger.Info("Checking if ConfigMap has entries")
	return len(c.configMap.Data) > 0
}

func loadConfigMap(kubernetesClient client.Client, logger logr.Logger, namespace string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	// check for existing config map
	err := kubernetesClient.Get(
		context.TODO(),
		client.ObjectKey{Name: configMapName, Namespace: namespace},
		configMap)
	if err != nil {
		logger.Error(err, "error getting config map from client")
	}

	if k8serrors.IsNotFound(err) {
		// create config map
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
		}
		logger.Info("creating ConfigMap")
		err = kubernetesClient.Create(context.TODO(), configMap)
	}
	if err != nil {
		logger.Error(err, "error loading config map")
	}
	return configMap, err
}
