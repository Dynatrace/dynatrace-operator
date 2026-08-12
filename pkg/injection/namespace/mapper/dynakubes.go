// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package mapper

import (
	"cmp"
	"context"
	goerrors "errors"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
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

	matchedNamespaces map[string]namespaceMatch
}

type namespaceMatch struct {
	namespace *corev1.Namespace
	flags     matchFlags
}

type matchFlags int

const (
	flagOneAgent matchFlags = 1 << iota
	flagMetadata
	flagOTLP
)

func (f matchFlags) isAny() bool       { return f > 0 }
func (f matchFlags) isOneAgent() bool  { return f&flagOneAgent != 0 }
func (f matchFlags) isMetadata() bool  { return f&flagMetadata != 0 }
func (f matchFlags) isBootstrap() bool { return f.isOneAgent() || f.isMetadata() }
func (f matchFlags) isOTLP() bool      { return f&flagOTLP != 0 }

func NewDynakubeMapper(ctx context.Context, clt client.Client, apiReader client.Reader, operatorNs string, dk *dynakube.DynaKube) DynakubeMapper {
	ctx, _ = logd.NewFromContext(ctx, "namespace-mapper")

	return DynakubeMapper{
		ctx:               ctx,
		client:            clt,
		apiReader:         apiReader,
		operatorNs:        operatorNs,
		dk:                dk,
		secrets:           k8ssecret.Query(clt, apiReader),
		matchedNamespaces: make(map[string]namespaceMatch),
	}
}

// MapFromDynakube checks all the namespaces to all the dynakubes
// updates the labels on the namespaces if necessary,
// finds conflicting dynakubes (2 dynakube with codeModules on the same namespace)
func (dm *DynakubeMapper) MapFromDynakube() error {
	modifiedNs, err := dm.MatchingNamespaces()
	if err != nil {
		return err
	}

	for _, ns := range modifiedNs {
		if err := dm.client.Update(dm.ctx, ns); err != nil {
			return err
		}
	}

	if err := dm.deleteUnmatchedReplicatedSecrets(); err != nil {
		return errors.WithMessage(err, "delete replicated secrets")
	}

	oaActive := dm.dk.OneAgent().IsAppInjectionNeeded()
	meActive := dm.dk.MetadataEnrichment().IsEnabled()
	otlpActive := dm.dk.OTLPExporterConfiguration().IsEnabled()
	setNamespacesMonitoredSelectorCondition(dm.ctx, dm.dk.Conditions(), oneAgentNamespacesMonitoredConditionType, oaActive, dm.namespaceNamesFor(flagOneAgent))
	setNamespacesMonitoredSelectorCondition(dm.ctx, dm.dk.Conditions(), metadataEnrichmentNamespacesMonitoredConditionType, meActive, dm.namespaceNamesFor(flagMetadata))
	setNamespacesMonitoredSelectorCondition(dm.ctx, dm.dk.Conditions(), otlpExporterNamespacesMonitoredConditionType, otlpActive, dm.namespaceNamesFor(flagOTLP))

	return nil
}

// BootstrapNamespaces returns the namespaces the bootstrapper init secret should be replicated to.
func (dm *DynakubeMapper) BootstrapNamespaces() []corev1.Namespace {
	return dm.namespacesFor(flagOneAgent | flagMetadata)
}

// OTLPNamespaces returns the namespaces the OTLP exporter secret should be replicated to.
func (dm *DynakubeMapper) OTLPNamespaces() []corev1.Namespace {
	return dm.namespacesFor(flagOTLP)
}

func (dm *DynakubeMapper) namespacesFor(query matchFlags) []corev1.Namespace {
	var namespaces []corev1.Namespace

	for _, entry := range dm.matchedNamespaces {
		if entry.flags&query != 0 {
			namespaces = append(namespaces, *entry.namespace)
		}
	}

	slices.SortFunc(namespaces, func(a, b corev1.Namespace) int { return cmp.Compare(a.Name, b.Name) })

	return namespaces
}

func (dm *DynakubeMapper) namespaceNamesFor(query matchFlags) []string {
	var namespaces []string

	for namespace, entry := range dm.matchedNamespaces {
		if entry.flags&query != 0 {
			namespaces = append(namespaces, namespace)
		}
	}

	slices.Sort(namespaces)

	return namespaces
}

func (dm *DynakubeMapper) MatchingNamespaces() ([]*corev1.Namespace, error) {
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

// UnmapFromDynaKube removes the injection label from all provided namespaces and deletes the secrets
func (dm *DynakubeMapper) UnmapFromDynaKube(namespaces []corev1.Namespace) error {
	for _, ns := range namespaces {
		delete(ns.Labels, dtwebhook.InjectionInstanceLabel)

		if err := dm.client.Update(dm.ctx, &ns); err != nil {
			return errors.WithMessagef(err, "failed to remove label %s from namespace %s", dtwebhook.InjectionInstanceLabel, ns.Name)
		}

		if err := dm.deleteReplicatedSecrets(ns.Name); err != nil {
			return err
		}
	}

	_ = meta.RemoveStatusCondition(dm.dk.Conditions(), oneAgentNamespacesMonitoredConditionType.String())
	_ = meta.RemoveStatusCondition(dm.dk.Conditions(), metadataEnrichmentNamespacesMonitoredConditionType.String())
	_ = meta.RemoveStatusCondition(dm.dk.Conditions(), otlpExporterNamespacesMonitoredConditionType.String())

	return nil
}

func (dm *DynakubeMapper) deleteUnmatchedReplicatedSecrets() error {
	for namespace, entry := range dm.matchedNamespaces {
		var errs []error

		if !entry.flags.isBootstrap() {
			errs = append(
				errs,
				dm.secrets.DeleteForNamespace(dm.ctx, consts.BootstrapperInitSecretName, namespace),
				dm.secrets.DeleteForNamespace(dm.ctx, consts.BootstrapperInitCertsSecretName, namespace),
			)
		}

		if !entry.flags.isOTLP() {
			errs = append(
				errs,
				dm.secrets.DeleteForNamespace(dm.ctx, consts.OTLPExporterSecretName, namespace),
				dm.secrets.DeleteForNamespace(dm.ctx, consts.OTLPExporterCertsSecretName, namespace),
			)
		}

		if err := goerrors.Join(errs...); err != nil {
			return err
		}
	}

	return nil
}

func (dm *DynakubeMapper) deleteReplicatedSecrets(namespaceName string) error {
	return goerrors.Join(
		dm.secrets.DeleteForNamespace(dm.ctx, consts.BootstrapperInitSecretName, namespaceName),
		dm.secrets.DeleteForNamespace(dm.ctx, consts.BootstrapperInitCertsSecretName, namespaceName),
		dm.secrets.DeleteForNamespace(dm.ctx, consts.OTLPExporterSecretName, namespaceName),
		dm.secrets.DeleteForNamespace(dm.ctx, consts.OTLPExporterCertsSecretName, namespaceName),
	)
}

func (dm *DynakubeMapper) mapFromDynakube(nsList *corev1.NamespaceList, dkList *dynakube.DynaKubeList) ([]*corev1.Namespace, error) {
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

	// Compile the selectors once instead of per amount of namespaces
	selectors := compileSelectors(dm.dk)

	for i := range nsList.Items {
		namespace := &nsList.Items[i]

		previouslyInjected := namespace.Labels[dtwebhook.InjectionInstanceLabel] == dm.dk.Name

		flags := match(dm.dk, namespace, selectors)

		if previouslyInjected || flags.isAny() {
			entry := dm.matchedNamespaces[namespace.Name]
			entry.namespace = namespace
			entry.flags |= flags
			dm.matchedNamespaces[namespace.Name] = entry
		}

		updated, err := updateNamespace(dm.ctx, namespace, dkList)
		if err != nil {
			return nil, err
		}

		if updated {
			modifiedNs = append(modifiedNs, namespace)
		}
	}

	return modifiedNs, nil
}
