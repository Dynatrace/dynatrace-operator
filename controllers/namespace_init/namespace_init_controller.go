package namespace_init

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"text/template"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	mapper "github.com/Dynatrace/dynatrace-operator/namespacemapper"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	//go:embed init.sh.tmpl
	scriptContent string
	scriptTmpl    = template.Must(template.New("initScript").Parse(scriptContent))
)

type ReconcileNamespaceInit struct {
	client    client.Client
	apiReader client.Reader
	logger    logr.Logger
	namespace string
}

type namespaceMapping struct {
	namespace string
	dynakube  string
}

type script struct {
	ApiUrl        string
	SkipCertCheck bool
	PaaSToken     string
	Proxy         string
	TrustedCAs    []byte
	ClusterID     string
	TenantUUID    string
	IMNodes       map[string]string
}

func applyForConfigMapName(ns string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return (event.Object.GetName() == mapper.CodeModulesMapName || event.Object.GetName() == mapper.DataIngestMapName) &&
				event.Object.GetNamespace() == ns
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			return (event.ObjectNew.GetName() == mapper.CodeModulesMapName || event.ObjectNew.GetName() == mapper.DataIngestMapName) &&
				event.ObjectNew.GetNamespace() == ns
		},
		DeleteFunc: func(event event.DeleteEvent) bool {
			return (event.Object.GetName() == mapper.CodeModulesMapName || event.Object.GetName() == mapper.DataIngestMapName) &&
				event.Object.GetNamespace() == ns
		},
	}
}

func Add(mgr manager.Manager, ns string) error {
	logger := log.Log.WithName("namespaces.controller")
	apmExists, err := kubeobjects.CheckIfOneAgentAPMExists(mgr.GetConfig())
	if err != nil {
		return err
	}
	if apmExists {
		logger.Info("OneAgentAPM object detected - Namespace reconciler disabled until the OneAgent Operator has been uninstalled")
		return nil
	}
	return NewReconciler(mgr, ns, logger).SetupWithManager(mgr, ns)
}

// NewReconciler returns a new ReconcileNamespaceInit
func NewReconciler(mgr manager.Manager, ns string, logger logr.Logger) *ReconcileNamespaceInit {
	return &ReconcileNamespaceInit{
		client:    mgr.GetClient(),
		apiReader: mgr.GetAPIReader(),
		namespace: ns,
		logger:    logger,
	}
}

func (r *ReconcileNamespaceInit) SetupWithManager(mgr ctrl.Manager, ns string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		WithEventFilter(applyForConfigMapName(ns)).
		Complete(r)
}

func (r *ReconcileNamespaceInit) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("Reconciling namespace map")

	mappingConfigMap := &corev1.ConfigMap{}
	err := r.client.Get(ctx, client.ObjectKey{Name: request.Name, Namespace: request.Namespace}, mappingConfigMap)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	namespaceMap := getNamespaceMapping(mappingConfigMap.Data)

	kubeSystemUID, err := kubesystem.GetUID(r.apiReader)
	if err != nil {
		return reconcile.Result{}, err
	}

	infraMonitoringNodes, err := r.getInfraMonitoringNodes()
	if err != nil {
		return reconcile.Result{}, err
	}

	if err = r.replicateInitScriptAsSecret(namespaceMap, kubeSystemUID, infraMonitoringNodes); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileNamespaceInit) replicateInitScriptAsSecret(namespaceMap []namespaceMapping, kubeSystemUID types.UID, infraMonitoringNodes map[string]string) error {
	for _, mapping := range namespaceMap {
		scriptData, err := r.prepareScriptForDynaKube(mapping.dynakube, kubeSystemUID, infraMonitoringNodes)
		if err != nil {
			return err
		}

		data, err := scriptData.generate()
		if err != nil {
			return err
		}

		if err = kubeobjects.CreateOrUpdateSecretIfNotExists(r.client, r.apiReader, webhook.SecretConfigName, mapping.namespace, data, corev1.SecretTypeOpaque, r.logger); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileNamespaceInit) prepareScriptForDynaKube(dk string, kubeSystemUID types.UID, infraMonitoringNodes map[string]string) (*script, error) {
	var dynaKube dynatracev1alpha1.DynaKube
	if err := r.client.Get(context.TODO(), client.ObjectKey{Name: dk, Namespace: r.namespace}, &dynaKube); err != nil {
		return nil, err
	}

	var tokens corev1.Secret
	if err := r.client.Get(context.TODO(), client.ObjectKey{Name: dynaKube.Tokens(), Namespace: r.namespace}, &tokens); err != nil {
		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	var proxy string
	if dynaKube.Spec.Proxy != nil {
		if dynaKube.Spec.Proxy.ValueFrom != "" {
			var ps corev1.Secret
			if err := r.client.Get(context.TODO(), client.ObjectKey{Name: dynaKube.Spec.Proxy.ValueFrom, Namespace: r.namespace}, &ps); err != nil {
				return nil, fmt.Errorf("failed to query proxy: %w", err)
			}
			proxy = string(ps.Data["proxy"])
		} else if dynaKube.Spec.Proxy.Value != "" {
			proxy = dynaKube.Spec.Proxy.Value
		}
	}

	var trustedCAs []byte
	if dynaKube.Spec.TrustedCAs != "" {
		var cam corev1.ConfigMap
		if err := r.client.Get(context.TODO(), client.ObjectKey{Name: dynaKube.Spec.TrustedCAs, Namespace: r.namespace}, &cam); err != nil {
			return nil, fmt.Errorf("failed to query ca: %w", err)
		}
		trustedCAs = []byte(cam.Data["certs"])
	}

	return &script{
		ApiUrl:        dynaKube.Spec.APIURL,
		SkipCertCheck: dynaKube.Spec.SkipCertCheck,
		PaaSToken:     string(tokens.Data[dtclient.DynatracePaasToken]),
		Proxy:         proxy,
		TrustedCAs:    trustedCAs,
		ClusterID:     string(kubeSystemUID),
		TenantUUID:    dynaKube.Status.ConnectionInfo.TenantUUID,
		IMNodes:       infraMonitoringNodes,
	}, nil
}

func (r *ReconcileNamespaceInit) getInfraMonitoringNodes() (map[string]string, error) {
	var ims dynatracev1alpha1.DynaKubeList
	if err := r.client.List(context.TODO(), &ims, client.InNamespace(r.namespace)); err != nil {
		return nil, errors.WithMessage(err, "failed to query DynaKubeList")
	}

	imNodes := map[string]string{}
	for i := range ims.Items {
		if s := &ims.Items[i].Status; s.ConnectionInfo.TenantUUID != "" && ims.Items[i].Spec.InfraMonitoring.Enabled {
			for key := range s.OneAgent.Instances {
				if key != "" {
					imNodes[key] = s.ConnectionInfo.TenantUUID
				}
			}
		}
	}

	return imNodes, nil
}

func getNamespaceMapping(configMapData map[string]string) []namespaceMapping {
	var mapping []namespaceMapping
	for ns, dk := range configMapData {
		mapping = append(mapping, namespaceMapping{
			namespace: ns,
			dynakube:  dk,
		})
	}

	return mapping
}

func (s *script) generate() (map[string][]byte, error) {
	var buf bytes.Buffer

	if err := scriptTmpl.Execute(&buf, s); err != nil {
		return nil, err
	}

	data := map[string][]byte{
		"init.sh": buf.Bytes(),
	}

	if s.TrustedCAs != nil {
		data["ca.pem"] = s.TrustedCAs
	}

	if s.Proxy != "" {
		data["proxy"] = []byte(s.Proxy)
	}

	return data, nil
}
