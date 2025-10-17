package mapper

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DynakubeMapper manages the mapping creation from the dynakube's side
type DynakubeMapper struct {
	ctx        context.Context
	client     client.Client
	apiReader  client.Reader
	dk         *dynakube.DynaKube
	operatorNs string
	secrets    k8ssecret.QueryObject
}

func NewDynakubeMapper(ctx context.Context, clt client.Client, apiReader client.Reader, operatorNs string, dk *dynakube.DynaKube) DynakubeMapper {
	return DynakubeMapper{
		ctx:        ctx,
		client:     clt,
		apiReader:  apiReader,
		operatorNs: operatorNs,
		dk:         dk,
		secrets:    k8ssecret.Query(clt, apiReader, log),
	}
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

		err := dm.secrets.DeleteForNamespace(dm.ctx, consts.BootstrapperInitSecretName, ns.Name)
		if err != nil {
			return err
		}

		err = dm.secrets.DeleteForNamespace(dm.ctx, consts.BootstrapperInitCertsSecretName, ns.Name)
		if err != nil {
			return err
		}

		err = dm.secrets.DeleteForNamespace(dm.ctx, consts.OTLPExporterSecretName, ns.Name)
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
	for _, ns := range modifiedNs {
		setUpdatedViaDynakubeAnnotation(ns)

		if err := dm.client.Update(dm.ctx, ns); err != nil {
			return err
		}
	}

	return nil
}
