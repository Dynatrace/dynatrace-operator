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
	namespace string
}

func NewChecker(kubernetesClient client.Client, logger logr.Logger, namespace string) (*Checker, error) {
	return &Checker{
		client:    kubernetesClient,
		logger:    logger,
		namespace: namespace,
	}, nil
}

// Add name of Dynakube with CodeModules enabled to ConfigMap.
// Should happen when Dynakube was created or setting was enabled.
func (c *Checker) Add(dynakube string) error {
	configMap, err := c.loadConfigMap()
	if err != nil || configMap.Data == nil {
		return ErrMissing
	}

	c.logger.Info("Adding Dynakube with CodeModules enabled",
		"dynakube", dynakube)
	configMap.Data[dynakube] = ""
	return c.client.Update(context.TODO(), configMap)
}

// Remove name of Dynakube from ConfigMap.
// Should happen when Dynakube was deleted or setting was disabled.
func (c *Checker) Remove(dynakube string) error {
	configMap, err := c.loadConfigMap()
	if err != nil || configMap.Data == nil {
		return ErrMissing
	}

	c.logger.Info("Removing Dynakube with CodeModules disabled",
		"dynakube", dynakube)
	delete(configMap.Data, dynakube)
	return c.client.Update(context.TODO(), configMap)
}

// Any checks if ConfigMap contains entries.
// If entries exist, there are Dynakubes with CodeModules enabled.
func (c *Checker) Any() (bool, error) {
	configMap, err := c.loadConfigMap()
	if err != nil || configMap.Data == nil {
		return false, ErrMissing
	}

	c.logger.Info("Checking if ConfigMap has entries")
	return len(configMap.Data) > 0, nil
}

func (c *Checker) loadConfigMap() (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	// check for existing config map
	err := c.client.Get(
		context.TODO(),
		client.ObjectKey{Name: configMapName, Namespace: c.namespace},
		configMap)
	if err != nil {
		c.logger.Error(err, "error getting config map from client")
	}

	if k8serrors.IsNotFound(err) {
		// create config map
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: c.namespace,
			},
			Data: map[string]string{},
		}
		c.logger.Info("creating ConfigMap")
		err = c.client.Create(context.TODO(), configMap)
	}
	if err != nil {
		c.logger.Error(err, "error loading config map")
	}
	return configMap, err
}
