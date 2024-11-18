package mapper

import (
	"context"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DynakubeMapper manages the mapping creation from the dynakube's side
type DynakubeMapper struct {
	ctx        context.Context
	client     client.Client
	apiReader  client.Reader
	dk         *dynakube.DynaKube
	operatorNs string
}

func NewDynakubeMapper(ctx context.Context, clt client.Client, apiReader client.Reader, operatorNs string, dk *dynakube.DynaKube) DynakubeMapper {
	return DynakubeMapper{ctx: ctx, client: clt, apiReader: apiReader, operatorNs: operatorNs, dk: dk}
}

// MapFromDynakube checks all the namespaces to all the dynakubes
// updates the labels on the namespaces if necessary,
// finds conflicting dynakubes (2 dynakube with codeModules on the same namespace)
func (dm DynakubeMapper) MapFromDynakube() error {
	modifiedNs, err := dm.MatchingNamespaces()
	if err != nil {
		return errors.Cause(err)
	}

	return dm.updateNamespaces(modifiedNs)
}

func (dm DynakubeMapper) MatchingNamespaces() ([]*corev1.Namespace, error) {
	nsList := &corev1.NamespaceList{}
	if err := dm.apiReader.List(dm.ctx, nsList); err != nil {
		return nil, errors.Cause(err)
	}

	dkList := &dynakube.DynaKubeList{}
	if err := dm.apiReader.List(dm.ctx, dkList, &client.ListOptions{Namespace: dm.operatorNs}); err != nil {
		return nil, errors.Cause(err)
	}

	return dm.mapFromDynakube(nsList, dkList)
}

func (dm DynakubeMapper) UnmapFromDynaKube(namespaces []corev1.Namespace) error {
	for i, ns := range namespaces {
		delete(ns.Labels, dtwebhook.InjectionInstanceLabel)
		setUpdatedViaDynakubeAnnotation(&namespaces[i])

		if err := dm.client.Update(dm.ctx, &namespaces[i]); err != nil {
			return errors.WithMessagef(err, "failed to remove label %s from namespace %s", dtwebhook.InjectionInstanceLabel, ns.Name)
		}

		err := k8ssecret.Query(dm.client, dm.apiReader, log).DeleteForNamespace(dm.ctx, consts.AgentInitSecretName, ns.Name)
		if err != nil {
			return err
		}

		err = k8ssecret.Query(dm.client, dm.apiReader, log).DeleteForNamespace(dm.ctx, consts.EnrichmentEndpointSecretName, ns.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dm DynakubeMapper) mapFromDynakube(nsList *corev1.NamespaceList, dkList *dynakube.DynaKubeList) ([]*corev1.Namespace, error) {
	var updated bool

	var err error

	var modifiedNs []*corev1.Namespace

	replaced := false

	for i := range dkList.Items {
		if dkList.Items[i].Name == dm.dk.Name {
			dkList.Items[i] = *dm.dk
			replaced = true

			break
		}
	}

	if !replaced {
		dkList.Items = append(dkList.Items, *dm.dk)
	}

	for i := range nsList.Items {
		namespace := &nsList.Items[i]

		updated, err = updateNamespace(namespace, dkList)
		if err != nil {
			return nil, err
		}

		if updated {
			modifiedNs = append(modifiedNs, namespace)
		}
	}

	return modifiedNs, err
}

func (dm DynakubeMapper) updateNamespaces(modifiedNs []*corev1.Namespace) error {
	log.Info("!!! updateNamespaces")

	previousMappedNamespaces := dm.dk.Status.MappedNamespaces
	currentMappedNamespaces := make([]string, len(modifiedNs))

	for i, ns := range modifiedNs {
		log.Info("!!! adding modifiedNs", ns.Name)
		currentMappedNamespaces[i] = ns.Name
	}

	if len(previousMappedNamespaces) != 0 {
		// ns selector update cleanup
		for _, previousNs := range previousMappedNamespaces {
			if !slices.Contains(currentMappedNamespaces, previousNs) {
				log.Info("!!! cleaning", "previousNs", previousNs)
				dm.UnmapFromDynaKube([]corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: previousNs}}})
			}
		}
	}

	for _, ns := range modifiedNs {
		setUpdatedViaDynakubeAnnotation(ns)

		if err := dm.client.Update(dm.ctx, ns); err != nil {
			return err
		}
	}

	log.Info("!!! updating MappedNamespaces", "currentMappedNamespaces", currentMappedNamespaces)
	dm.dk.Status.MappedNamespaces = currentMappedNamespaces
	dm.dk.UpdateStatus(dm.ctx, dm.client)

	return nil
}
