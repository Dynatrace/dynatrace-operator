package dtcsi

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("csi_mapper")

const (
	CsiMapperConfigMapName = "dynatrace-csi-mapper"
)

type csiMapper struct {
	client    client.Client
	configMap *corev1.ConfigMap
}

func ConfigureCSIDriver(
	client client.Client, scheme *runtime.Scheme, operatorPodName, operatorNamespace string,
	rec *utils.Reconciliation, updateInterval time.Duration) error {
	configMap, err := loadOrCreateConfigMap(client, rec.Instance.Namespace)
	if err != nil {
		return err
	}

	csiMapper := &csiMapper{
		client:    client,
		configMap: configMap,
	}
	if rec.Instance.Spec.CodeModules.Enabled {
		if !csiMapper.any() {
			// enable csi driver, if first Dynakube with CodeModules enabled
			log.Info("enabling csi driver")
			upd, err := NewReconciler(client, scheme, rec.Log, rec.Instance, operatorPodName, operatorNamespace).Reconcile()
			if err != nil {
				return err
			}
			if err = csiMapper.add(rec.Instance.Name); err != nil {
				return err
			}
			if rec.Update(upd, updateInterval, "CSI driver reconciled") {
				return nil
			}
		}
		if err := csiMapper.add(rec.Instance.Name); err != nil {
			return err
		}
	} else {
		if err := csiMapper.remove(rec.Instance.Name); err != nil {
			return err
		}
		if !csiMapper.any() {
			log.Info("ensuring csi driver is disabled")
			// disable csi driver, no Dynakubes with CodeModules enabled exist any more
			// ensures csi driver is disabled, when additional CodeModules are disabled
			ds := appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DaemonSetName,
					Namespace: rec.Instance.Namespace,
				},
			}
			if err := csiMapper.ensureDeleted(&ds); rec.Error(err) {
				return err
			}
		}
	}
	return nil
}

// add name of Dynakube with CodeModules enabled to ConfigMap.
// Should happen when Dynakube was created or setting was enabled.
func (c *csiMapper) add(dynakube string) error {
	if c.configMap.Data == nil {
		c.configMap.Data = map[string]string{}
	}

	if _, contains := c.configMap.Data[dynakube]; !contains {
		c.configMap.Data[dynakube] = ""
		return c.client.Update(context.TODO(), c.configMap)
	}
	return nil
}

// remove name of Dynakube from ConfigMap.
// Should happen when Dynakube was deleted or setting was disabled.
func (c *csiMapper) remove(dynakube string) error {
	if c.configMap.Data == nil {
		return nil
	}

	if _, contains := c.configMap.Data[dynakube]; contains {
		delete(c.configMap.Data, dynakube)
		return c.client.Update(context.TODO(), c.configMap)
	}
	return nil
}

// any checks if ConfigMap contains entries.
// If entries exist, there are Dynakubes with CodeModules enabled.
func (c *csiMapper) any() bool {
	if c.configMap.Data == nil {
		return false
	}

	return len(c.configMap.Data) > 0
}

func loadOrCreateConfigMap(kubernetesClient client.Client, namespace string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	// check for existing config map
	err := kubernetesClient.Get(
		context.TODO(),
		client.ObjectKey{Name: CsiMapperConfigMapName, Namespace: namespace},
		configMap)

	if k8serrors.IsNotFound(err) {
		// create config map
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CsiMapperConfigMapName,
				Namespace: namespace,
			},
		}
		log.Info("creating ConfigMap")
		err = kubernetesClient.Create(context.TODO(), configMap)
	}
	return configMap, err
}

func (c *csiMapper) ensureDeleted(obj client.Object) error {
	if err := c.client.Delete(context.TODO(), obj); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}
