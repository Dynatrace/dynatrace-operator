package dtcsi

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ConfigureCSIDriver(
	client client.Client, scheme *runtime.Scheme, operatorPodName, operatorNamespace string,
	rec *utils.Reconciliation, updateInterval time.Duration) error {

	if rec.Instance.Spec.CodeModules.Enabled {
		err := addDynakube(client, scheme, operatorPodName, operatorNamespace, rec, updateInterval)
		if err != nil {
			return err
		}
	} else {
		err := removeDynakube(client, rec)
		if err != nil {
			return err
		}
	}
	return nil
}

// addDynakube enables csi driver, by creating its DaemonSet (if it does not exist yet)
// and adds the current Dynakube to the OwnerReferences of the DaemonSet
func addDynakube(
	client client.Client, scheme *runtime.Scheme, operatorPodName string, operatorNamespace string,
	rec *utils.Reconciliation, updateInterval time.Duration) error {

	csiDaemonSet, err := getCSIDaemonSet(client, rec.Instance.Namespace)
	if k8serrors.IsNotFound(err) {
		rec.Log.Info("enabling csi driver")
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

// removeDynakube removes the current Dynakube from the OwnerReferences of the DaemonSet
// and deletes the DaemonSet if no Owners are left.
func removeDynakube(clt client.Client, rec *utils.Reconciliation) error {
	csiDaemonSet, err := getCSIDaemonSet(clt, rec.Instance.Namespace)
	if k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	itemIndex, itemFound := findOwnerReferenceIndex(csiDaemonSet.OwnerReferences, rec.Instance.UID)
	if !itemFound {
		// Dynakube was not found in existing OwnerReferences
		return nil
	}

	err = updateOrDeleteDaemonSet(clt, csiDaemonSet, itemIndex, rec.Log)
	if err != nil {
		return err
	}
	return nil
}

func updateOrDeleteDaemonSet(clt client.Client, csiDaemonSet *appsv1.DaemonSet, itemIndex int, log logr.Logger) error {
	if len(csiDaemonSet.OwnerReferences) > 1 {
		csiDaemonSet.OwnerReferences = append(
			csiDaemonSet.OwnerReferences[:itemIndex],
			csiDaemonSet.OwnerReferences[itemIndex+1:]...)
	} else {
		// Delete CSI DaemonSet manually if no OwnerReferences are left
		return clt.Delete(context.TODO(), csiDaemonSet)
	}

	log.Info("Removing Dynakube from CSI DaemonSet")
	return clt.Update(context.TODO(), csiDaemonSet)
}

func findOwnerReferenceIndex(ownerReferences []metav1.OwnerReference, instanceUID types.UID) (int, bool) {
	for i, ownerReference := range ownerReferences {
		if ownerReference.UID == instanceUID {
			return i, true
		}
	}
	return 0, false
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

func getCSIDaemonSet(clt client.Client, namespace string) (*appsv1.DaemonSet, error) {
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
