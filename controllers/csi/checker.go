package dtcsi

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	configMapName = "dynatrace-csi-checker"
)

type Checker struct {
	client            client.Client
	logger            logr.Logger
	configMap         *corev1.ConfigMap
	namespace         string
	operatorPodName   string
	operatorNamespace string
}

func NewChecker(kubernetesClient client.Client, logger logr.Logger, namespace string, operatorPodName, operatorNamespace string) (*Checker, error) {
	configMap, err := loadOrCreateConfigMap(kubernetesClient, logger, namespace)
	if err != nil {
		return nil, err
	}
	return &Checker{
		client:            kubernetesClient,
		logger:            logger,
		configMap:         configMap,
		namespace:         namespace,
		operatorPodName:   operatorPodName,
		operatorNamespace: operatorNamespace,
	}, nil
}

func (c *Checker) ConfigureCsiDriver(rec *utils.Reconciliation, scheme *runtime.Scheme, updateInterval time.Duration) error {
	if rec.Instance.Spec.CodeModules.Enabled {
		if !c.any() {
			// enable csi driver, if first Dynakube with CodeModules enabled
			upd, err := NewReconciler(c.client, scheme, c.logger, rec.Instance, c.operatorPodName, c.operatorNamespace).Reconcile()
			if err != nil {
				return err
			}
			if err = c.add(rec.Instance.Name); err != nil {
				return err
			}
			if rec.Update(upd, updateInterval, "CSI driver reconciled") {
				return nil
			}
		}
		if err := c.add(rec.Instance.Name); err != nil {
			return err
		}
	} else {
		if err := c.remove(rec.Instance.Name); err != nil {
			return err
		}
		if !c.any() {
			// disable csi driver, no Dynakubes with CodeModules enabled exist anymore
			ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: DaemonSetName, Namespace: rec.Instance.Namespace}}
			if err := c.ensureDeleted(&ds); rec.Error(err) {
				return err
			}
		}
	}
	return nil
}

// Add name of Dynakube with CodeModules enabled to ConfigMap.
// Should happen when Dynakube was created or setting was enabled.
func (c *Checker) add(dynakube string) error {
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
func (c *Checker) remove(dynakube string) error {
	if c.configMap.Data == nil {
		return nil
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
func (c *Checker) any() bool {
	if c.configMap.Data == nil {
		return false
	}

	c.logger.Info("Checking if ConfigMap has entries")
	return len(c.configMap.Data) > 0
}

// loadOrCreateConfigMap loads existing ConfigMap or creates it if it doesn't exist
func loadOrCreateConfigMap(kubernetesClient client.Client, logger logr.Logger, namespace string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	// check for existing config map
	err := kubernetesClient.Get(
		context.TODO(),
		client.ObjectKey{Name: configMapName, Namespace: namespace},
		configMap)

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
	return configMap, err
}

func (c *Checker) ensureDeleted(obj client.Object) error {
	if err := c.client.Delete(context.TODO(), obj); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}
