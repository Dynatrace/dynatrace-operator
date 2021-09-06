package mapper

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceMapper struct {
	ctx        context.Context
	client     client.Client
	apiReader  client.Reader
	operatorNs string
	targetNs   *corev1.Namespace
	logger     logr.Logger
}

func NewNamespaceMapper(ctx context.Context, clt client.Client, apiReader client.Reader, operatorNs string, targetNs *corev1.Namespace, logger logr.Logger) NamespaceMapper {
	return NamespaceMapper{ctx, clt, apiReader, operatorNs, targetNs, logger}
}

func (nm NamespaceMapper) MapFromNamespace() error {
	err := nm.mapFromNamespaceDataIngest()
	if err != nil {
		return err
	}
	return nm.mapFromNamespaceCodeModules()
}

func (nm NamespaceMapper) UnmapFromNamespace() error {
	err := nm.unmapFromNamespace(DataIngestMapName)
	if err != nil {
		return err
	}
	return nm.unmapFromNamespace(CodeModulesMapName)
}

func (nm NamespaceMapper) mapFromNamespaceDataIngest() error {
	return nm.mapFromNamespace(DataIngestMapName,
		func(dk dynatracev1alpha1.DynaKube) bool {
			return dk.Spec.DataIngestSpec.Enabled
		},
		func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.DataIngestSpec.Selector
		}, false)
}

func (nm NamespaceMapper) mapFromNamespaceCodeModules() error {
	return nm.mapFromNamespace(CodeModulesMapName,
		func(dk dynatracev1alpha1.DynaKube) bool {
			return dk.Spec.CodeModules.Enabled
		},
		func(dk dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
			return dk.Spec.CodeModules.NamespaceSelector
		}, true)
}

func (nm NamespaceMapper) findDynaKubes(dynakubeFilter dynaKubeFilterFunc) ([]dynatracev1alpha1.DynaKube, error) {
	dynaKubeList := &dynatracev1alpha1.DynaKubeList{}
	err := nm.client.List(nm.ctx, dynaKubeList)
	if err != nil {
		return nil, errors.Cause(err)
	}

	var dynakubes []dynatracev1alpha1.DynaKube
	for _, dynakube := range dynaKubeList.Items {
		if dynakubeFilter(dynakube) {
			dynakubes = append(dynakubes, dynakube)
		}
	}
	nm.logger.Info("Potential dynakubes to match to", "len", len(dynakubes))
	return dynakubes, nil
}

func (nm NamespaceMapper) matchForNamespace(dynakubes []dynatracev1alpha1.DynaKube, nsSelector namespaceSelectorFunc) (*dynatracev1alpha1.DynaKube, error) {
	var matchingDynakubes []dynatracev1alpha1.DynaKube

	for _, dk := range dynakubes {
		selector, err := metav1.LabelSelectorAsSelector(nsSelector(dk))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if selector.Matches(labels.Set(nm.targetNs.Labels)) {
			matchingDynakubes = append(matchingDynakubes, dk)
		}
	}

	if len(matchingDynakubes) > 1 {
		return nil, errors.New("namespace matches two or more DynaKubes which is unsupported. " +
			"refine the labels on your namespace metadata or DynaKube/CodeModules specification")
	}
	if len(matchingDynakubes) == 0 {
		nm.logger.Info("No matching dk found")
		return nil, nil
	}
	nm.logger.Info("Matching dk found", "dkName", matchingDynakubes[0].Name)
	return &matchingDynakubes[0], nil
}

func (nm NamespaceMapper) findForNamespace(dkFilter dynaKubeFilterFunc, nsSelector namespaceSelectorFunc) (*dynatracev1alpha1.DynaKube, error) {
	dynakubes, err := nm.findDynaKubes(dkFilter)

	if err != nil {
		return nil, err
	}

	return nm.matchForNamespace(dynakubes, nsSelector)
}

func (nm NamespaceMapper) mapFromNamespace(cfgMapName string, dkFilter dynaKubeFilterFunc, nsSelector namespaceSelectorFunc, modNsLabel bool) error {
	dynakube, err := nm.findForNamespace(dkFilter, nsSelector)

	if err != nil {
		return err
	}

	cfgMap, err := getOrCreateMap(nm.ctx, nm.client, nm.apiReader, nm.operatorNs, cfgMapName)
	if err != nil {
		return err
	}

	if dynakube == nil {
		if modNsLabel {
			removeNamespaceLabel(nm.ctx, nm.client, nm.targetNs)
		}
		return nm.removeFromMap(cfgMapName, cfgMap)
	}
	if modNsLabel {
		updateNamespaceLabel(nm.ctx, nm.client, nm.operatorNs, nm.targetNs, dynakube)
	}
	return nm.updateMap(cfgMapName, cfgMap, dynakube.Name)
}

func (nm NamespaceMapper) unmapFromNamespace(cfgMapName string) error {
	cfgMap, err := getOrCreateMap(nm.ctx, nm.client, nm.apiReader, nm.operatorNs, cfgMapName)
	if err != nil {
		return err
	}

	return nm.removeFromMap(cfgMapName, cfgMap)
}

// updateMap sets `dkName` value to `namespace` key in `cfgMap`
func (nm NamespaceMapper) updateMap(cfgMapName string, cfgMap *corev1.ConfigMap, dkName string) error {
	if cfgMap.Data == nil {
		cfgMap.Data = make(map[string]string)
	}
	if dk, ok := cfgMap.Data[nm.targetNs.Name]; !ok || dk != dkName {
		cfgMap.Data[nm.targetNs.Name] = dkName

		if err := nm.client.Update(nm.ctx, cfgMap); err != nil {
			return errors.WithMessagef(err, "failed to update %s", cfgMapName)
		}
	}
	return nil
}

func (nm NamespaceMapper) removeFromMap(cfgMapName string, cfgMap *corev1.ConfigMap) error {
	if cfgMap.Data == nil {
		return nil
	}
	if _, ok := cfgMap.Data[nm.targetNs.Name]; ok {
		delete(cfgMap.Data, nm.targetNs.Name)

		if err := nm.client.Update(nm.ctx, cfgMap); err != nil {
			return errors.WithMessagef(err, "failed to remove namespace %s from %s", nm.targetNs.Name, cfgMapName)
		}
	}
	return nil
}
