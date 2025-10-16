package mapper

import (
	"context"
	"sort"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
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

	matchedOANamespaces []string
	matchedMENamespaces []string
}

type matchedNamespaces struct {
	oneAgent           []string
	metadataEnrichment []string
}

func NewDynakubeMapper(ctx context.Context, clt client.Client, apiReader client.Reader, operatorNs string, dk *dynakube.DynaKube) DynakubeMapper {
	return DynakubeMapper{
		ctx:                 ctx,
		client:              clt,
		apiReader:           apiReader,
		operatorNs:          operatorNs,
		dk:                  dk,
		secrets:             k8ssecret.Query(clt, apiReader, log),
		matchedOANamespaces: []string{},
		matchedMENamespaces: []string{},
	}
}

// MapFromDynakube checks all the namespaces to all the dynakubes
// updates the labels on the namespaces if necessary,
// finds conflicting dynakubes (2 dynakube with codeModules on the same namespace)
func (dm *DynakubeMapper) MapFromDynakube() error {
	nsList := &corev1.NamespaceList{}
	if err := dm.apiReader.List(dm.ctx, nsList); err != nil {
		return errors.Cause(err)
	}

	dkList := &dynakube.DynaKubeList{}
	if err := dm.apiReader.List(dm.ctx, dkList, &client.ListOptions{Namespace: dm.operatorNs}); err != nil {
		return errors.Cause(err)
	}

	modifiedNs, matchedNamespaces, err := dm.mapFromDynakube(nsList, dkList)
	if err != nil {
		return err
	}

	if err := dm.updateNamespaces(modifiedNs); err != nil {
		return err
	}

	if dm.dk.OneAgent().IsAppInjectionNeeded() {
		log.Info("namespaces monitored",
			"selector", "OneAgent",
			"count (at most 10 are shown)", len(matchedNamespaces.oneAgent),
			"namespaces", matchedNamespaces.oneAgent,
		)
	}

	if dm.dk.MetadataEnrichment().IsEnabled() {
		log.Info("namespaces monitored",
			"selector", "MetadataEnrichment",
			"count (at most 10 are shown)", len(matchedNamespaces.metadataEnrichment),
			"namespaces", matchedNamespaces.metadataEnrichment,
		)
	}

	oaActive := dm.dk.OneAgent().IsAppInjectionNeeded()
	meActive := dm.dk.MetadataEnrichment().IsEnabled()
	setNamespacesMonitoredSelectorCondition(dm.dk.Conditions(), "OneAgent", oaActive, matchedNamespaces.oneAgent)
	setNamespacesMonitoredSelectorCondition(dm.dk.Conditions(), "MetadataEnrichment", meActive, matchedNamespaces.metadataEnrichment)
	updateCollectedNamespacesMonitoredCondition(dm.dk.Conditions())

	return nil
}

func (dm *DynakubeMapper) MatchingNamespaces() ([]*corev1.Namespace, matchedNamespaces, error) {
	nsList := &corev1.NamespaceList{}
	if err := dm.apiReader.List(dm.ctx, nsList); err != nil {
		return nil, matchedNamespaces{}, errors.Cause(err)
	}

	dkList := &dynakube.DynaKubeList{}
	if err := dm.apiReader.List(dm.ctx, dkList, &client.ListOptions{Namespace: dm.operatorNs}); err != nil {
		return nil, matchedNamespaces{}, errors.Cause(err)
	}

	return dm.mapFromDynakube(nsList, dkList)
}

func (dm *DynakubeMapper) UnmapFromDynaKube(namespaces []corev1.Namespace) error {
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
	}

	_ = meta.RemoveStatusCondition(dm.dk.Conditions(), oneAgentNamespacesMonitoredConditionType)
	_ = meta.RemoveStatusCondition(dm.dk.Conditions(), metadataEnrichmentNamespacesMonitoredConditionType)

	return nil
}

func (dm *DynakubeMapper) mapFromDynakube(nsList *corev1.NamespaceList, dkList *dynakube.DynaKubeList) ([]*corev1.Namespace, matchedNamespaces, error) {
	var (
		modifiedNs        []*corev1.Namespace
		matchedNamespaces matchedNamespaces
	)

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

		if !isIgnoredNamespace(dm.dk, namespace.Name) {
			if ok, err := matchOneAgent(dm.dk, namespace); err != nil {
				return nil, matchedNamespaces, err
			} else if ok {
				matchedNamespaces.oneAgent = append(matchedNamespaces.oneAgent, namespace.Name)
			}

			if ok, err := matchMetadataEnrichment(dm.dk, namespace); err != nil {
				return nil, matchedNamespaces, err
			} else if ok {
				matchedNamespaces.metadataEnrichment = append(matchedNamespaces.metadataEnrichment, namespace.Name)
			}
		}

		updated, err := updateNamespace(namespace, dkList)
		if err != nil {
			return nil, matchedNamespaces, err
		}

		if updated {
			modifiedNs = append(modifiedNs, namespace)
		}
	}

	sort.Strings(matchedNamespaces.oneAgent)
	sort.Strings(matchedNamespaces.metadataEnrichment)

	return modifiedNs, matchedNamespaces, nil
}

func (dm *DynakubeMapper) updateNamespaces(modifiedNs []*corev1.Namespace) error {
	for _, ns := range modifiedNs {
		setUpdatedViaDynakubeAnnotation(ns)

		if err := dm.client.Update(dm.ctx, ns); err != nil {
			return err
		}
	}

	return nil
}
