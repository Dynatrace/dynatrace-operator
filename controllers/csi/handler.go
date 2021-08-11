package dtcsi

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("csi_mapper")

func ConfigureCSIDriver(
	client client.Client, scheme *runtime.Scheme, operatorPodName, operatorNamespace string,
	rec *utils.Reconciliation, updateInterval time.Duration) error {

	if rec.Instance.Spec.CodeModules.Enabled {
		err := enableCSIDriver(client, scheme, operatorPodName, operatorNamespace, rec, updateInterval)
		if err != nil {
			return err
		}
	} else {
		err := disableCSIDriver(client, rec)
		if err != nil {
			return err
		}
	}
	return nil
}

// disableCSIDriver disables csi driver by removing its daemon set.
// ensures csi driver is disabled, when additional CodeModules are disabled.
func disableCSIDriver(clt client.Client, rec *utils.Reconciliation) error {
	csiDaemonSet, err := getCsiDaemonSet(clt, rec.Instance.Namespace)
	if k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	itemIndex := -1
	for i, ownerReference := range csiDaemonSet.OwnerReferences {
		if ownerReference.UID == rec.Instance.UID {
			itemIndex = i
			break
		}
	}

	if itemIndex == -1 {
		// Dynakube was not found in existing OwnerReferences
		return nil
	}

	if len(csiDaemonSet.OwnerReferences) > 1 {
		csiDaemonSet.OwnerReferences = append(
			csiDaemonSet.OwnerReferences[:itemIndex],
			csiDaemonSet.OwnerReferences[itemIndex+1:]...)
	} else {
		// Delete CSI Daemonset manually if no OwnerReferences are left
		err = clt.Delete(context.TODO(), csiDaemonSet)
		return err
	}

	log.Info("Removing Dynakube from CSI Daemonset")
	err = clt.Update(context.TODO(), csiDaemonSet)
	if err != nil {
		return err
	}
	return nil
}

// enableCSIDriver tries to enable csi driver, by creating its daemon set.
func enableCSIDriver(
	client client.Client, scheme *runtime.Scheme, operatorPodName string, operatorNamespace string,
	rec *utils.Reconciliation, updateInterval time.Duration) error {

	csiDaemonSet, err := getCsiDaemonSet(client, rec.Instance.Namespace)
	if k8serrors.IsNotFound(err) {
		log.Info("enabling csi driver")
		csiDaemonSetReconciler := NewReconciler(client, scheme, rec.Log, rec.Instance, operatorPodName, operatorNamespace)
		upd, err := csiDaemonSetReconciler.Reconcile()
		if err != nil {
			return err
		}
		if rec.Update(upd, updateInterval, "CSI driver reconciled") {
			return nil
		}
		return nil
	} else if err != nil {
		return err
	}

	for _, ownerReference := range csiDaemonSet.OwnerReferences {
		if ownerReference.UID == rec.Instance.UID {
			// Dynakube already defined as Owner of CSI DaemonSet
			return nil
		}
	}

	csiDaemonSet.OwnerReferences = append(csiDaemonSet.OwnerReferences, createOwnerReference(rec.Instance))
	err = client.Update(context.TODO(), csiDaemonSet)
	if err != nil {
		return err
	}
	return nil
}

func createOwnerReference(dynakube *v1alpha1.DynaKube) metav1.OwnerReference {
	trueVal := true
	return metav1.OwnerReference{
		APIVersion:         dynakube.APIVersion,
		Kind:               dynakube.Kind,
		Name:               dynakube.Name,
		UID:                dynakube.UID,
		Controller:         &trueVal,
		BlockOwnerDeletion: &trueVal,
	}
}

func getCsiDaemonSet(clt client.Client, namespace string) (*appsv1.DaemonSet, error) {
	csiDaemonSet := &appsv1.DaemonSet{}
	err := clt.Get(context.TODO(),
		client.ObjectKey{
			Name:      DaemonSetName,
			Namespace: namespace,
		}, csiDaemonSet)

	if err != nil {
		return nil, err
	}
	return csiDaemonSet, nil
}
