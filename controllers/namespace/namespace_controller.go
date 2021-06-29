package namespace

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"text/template"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

//go:embed init.sh.tmpl
var scriptContent string

var scriptTmpl = template.Must(template.New("initScript").Parse(scriptContent))

func Add(mgr manager.Manager, ns string) error {
	logger := log.Log.WithName("namespaces.controller")
	apmExists, err := utils.CheckIfOneAgentAPMExists(mgr.GetConfig())
	if err != nil {
		return err
	}
	if apmExists {
		logger.Info("OneAgentAPM object detected - Namespace reconciler disabled until the OneAgent Operator has been uninstalled")
		return nil
	}

	return add(mgr, &ReconcileNamespaces{
		client:    mgr.GetClient(),
		apiReader: mgr.GetAPIReader(),
		namespace: ns,
		logger:    logger,
	})
}

func add(mgr manager.Manager, r *ReconcileNamespaces) error {
	// Create a new controller
	c, err := controller.New("namespace-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Namespaces
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

type ReconcileNamespaces struct {
	client    client.Client
	apiReader client.Reader
	logger    logr.Logger
	namespace string
}

func (r *ReconcileNamespaces) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	targetNS := request.Name
	log := r.logger.WithValues("name", targetNS)
	log.Info("reconciling Namespace")

	var ns corev1.Namespace
	if err := r.client.Get(ctx, client.ObjectKey{Name: targetNS}, &ns); k8serrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to query Namespace")
	}

	if ns.Labels == nil {
		return reconcile.Result{}, nil
	}

	oaName := ns.Labels[webhook.LabelInstance]
	if oaName == "" {
		return reconcile.Result{}, nil
	}

	var dk dynatracev1alpha1.DynaKube
	if err := r.client.Get(ctx, client.ObjectKey{Name: oaName, Namespace: r.namespace}, &dk); err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to query DynaKubes")
	}

	tokenName := dk.Tokens()
	if !dk.Spec.CodeModules.Enabled {
		_ = r.ensureSecretDeleted(tokenName, targetNS)
		return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	var ims dynatracev1alpha1.DynaKubeList
	if err := r.client.List(ctx, &ims, client.InNamespace(r.namespace)); err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to query DynaKubeList")
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

	var tkns corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{Name: tokenName, Namespace: r.namespace}, &tkns); err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to query tokens")
	}

	script, err := newScript(ctx, r.client, dk, tkns, imNodes, r.namespace)
	if err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to generate init script")
	}

	data, err := script.generate()
	if err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to generate script")
	}

	// The default cache-based Client doesn't support cross-namespace queries, unless configured to do so in Manager
	// Options. However, this is our only use-case for it, so using the non-cached Client instead.
	err = utils.CreateOrUpdateSecretIfNotExists(r.client, r.apiReader, webhook.SecretConfigName, targetNS, data, corev1.SecretTypeOpaque, log)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}

type script struct {
	DynaKube   *dynatracev1alpha1.DynaKube
	PaaSToken  string
	Proxy      string
	TrustedCAs []byte
	ClusterID  string
	IMNodes    map[string]string
}

func (r *ReconcileNamespaces) ensureSecretDeleted(name string, ns string) error {
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
	if err := r.client.Delete(context.TODO(), &secret); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}

func newScript(ctx context.Context, c client.Client, dynaKube dynatracev1alpha1.DynaKube, tkns corev1.Secret, imNodes map[string]string, ns string) (*script, error) {
	var kubeSystemNS corev1.Namespace
	if err := c.Get(ctx, client.ObjectKey{Name: "kube-system"}, &kubeSystemNS); err != nil {
		return nil, fmt.Errorf("failed to query for cluster UUID: %w", err)
	}

	var proxy string
	if dynaKube.Spec.Proxy != nil {
		if dynaKube.Spec.Proxy.ValueFrom != "" {
			var ps corev1.Secret
			if err := c.Get(ctx, client.ObjectKey{Name: dynaKube.Spec.Proxy.ValueFrom, Namespace: ns}, &ps); err != nil {
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
		if err := c.Get(ctx, client.ObjectKey{Name: dynaKube.Spec.TrustedCAs, Namespace: ns}, &cam); err != nil {
			return nil, fmt.Errorf("failed to query ca: %w", err)
		}
		trustedCAs = []byte(cam.Data["certs"])
	}

	return &script{
		DynaKube:   &dynaKube,
		PaaSToken:  string(tkns.Data[utils.DynatracePaasToken]),
		Proxy:      proxy,
		TrustedCAs: trustedCAs,
		ClusterID:  string(kubeSystemNS.UID),
		IMNodes:    imNodes,
	}, nil
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
