package dtcsi

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/controllers"
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
	dkState *controllers.DynakubeState, updateInterval time.Duration) error {
	configMap, err := loadOrCreateConfigMap(client, dkState.Instance.Namespace)
	if err != nil {
		return err
	}

	csiMapper := &csiMapper{
		client:    client,
		configMap: configMap,
	}
	if dkState.Instance.Spec.CodeModules.Enabled {
		if !csiMapper.hasActiveCSIDrivers() {
			err := enableCSIDriver(client, scheme, operatorPodName, operatorNamespace, dkState, updateInterval, csiMapper)
			if err != nil {
				return err
			}
		}
		if err := csiMapper.add(dkState.Instance.Name); err != nil {
			return err
		}
	} else {
		if err := csiMapper.remove(dkState.Instance.Name); err != nil {
			return err
		}
		if !csiMapper.hasActiveCSIDrivers() {
			err = disableCSIDriver(dkState, csiMapper)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// disableCSIDriver disables csi driver by removing its daemon set.
// ensures csi driver is disabled, when additional CodeModules are disabled.
func disableCSIDriver(dkState *controllers.DynakubeState, csiMapper *csiMapper) error {
	log.Info("ensuring csi driver is disabled")
	ds := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DaemonSetName,
			Namespace: dkState.Instance.Namespace,
		},
	}
	if err := csiMapper.ensureDeleted(&ds); dkState.Error(err) {
		return err
	}
	return nil
}

// enableCSIDriver tries to enable csi driver, by creating its daemon set.
func enableCSIDriver(
	client client.Client, scheme *runtime.Scheme, operatorPodName string, operatorNamespace string,
	dkState *controllers.DynakubeState, updateInterval time.Duration, csiMapper *csiMapper) error {

	log.Info("enabling csi driver")
	upd, err := NewReconciler(client, scheme, dkState.Log, dkState.Instance, operatorPodName, operatorNamespace).Reconcile()
	if err != nil {
		return err
	}
	if err = csiMapper.add(dkState.Instance.Name); err != nil {
		return err
	}
	if dkState.Update(upd, updateInterval, "CSI driver reconciled") {
		return nil
	}
	return nil
}

// add name of Dynakube with CodeModules enabled to ConfigMap.
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

// hasActiveCSIDrivers checks if CSI drivers are currently active,
// by checking if the ConfigMap has entries.
func (c *csiMapper) hasActiveCSIDrivers() bool {
	if c.configMap.Data == nil {
		return false
	}

	return len(c.configMap.Data) > 0
}

func loadOrCreateConfigMap(clt client.Client, namespace string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	// check for existing config map
	err := clt.Get(
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
		err = clt.Create(context.TODO(), configMap)
	}
	return configMap, err
}

func (c *csiMapper) ensureDeleted(obj client.Object) error {
	if err := c.client.Delete(context.TODO(), obj); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}
