package namespace

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func Add(mgr manager.Manager, ns string) error {
	return add(mgr, &ReconcileNamespaces{
		client:                  mgr.GetClient(),
		apiReader:               mgr.GetAPIReader(),
		namespace:               ns,
		logger:                  log.Log.WithName("namespaces.controller"),
		pullSecretGeneratorFunc: utils.GeneratePullSecretData,
		addNodeProps:            os.Getenv("ONEAGENT_OPERATOR_DEBUG_NODE_PROPERTIES") == "true",
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
	client                  client.Client
	apiReader               client.Reader
	logger                  logr.Logger
	namespace               string
	pullSecretGeneratorFunc func(c client.Client, oa dynatracev1alpha1.DynaKube, tkns *corev1.Secret) (map[string][]byte, error)
	addNodeProps            bool
}

func (r *ReconcileNamespaces) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	targetNS := request.Name

	ctx := context.TODO()

	log := r.logger.WithValues("name", targetNS)
	log.Info("reconciling Namespace")

	var ns corev1.Namespace
	if err := r.client.Get(ctx, client.ObjectKey{Name: targetNS}, &ns); errors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to query Namespace: %w", err)
	}

	if ns.Labels == nil {
		return reconcile.Result{}, nil
	}

	oaName := ns.Labels[webhook.LabelInstance]
	if oaName == "" {
		return reconcile.Result{}, nil
	}

	// TODO(lrgar): to replace with list of OneAgentIM objects.
	var ims dynatracev1alpha1.DynaKubeList
	if err := r.client.List(ctx, &ims, client.InNamespace(r.namespace)); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to query OneAgentIMs: %w", err)
	}

	var apm dynatracev1alpha1.DynaKube
	if err := r.client.Get(ctx, client.ObjectKey{Name: oaName, Namespace: r.namespace}, &apm); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to query OneAgentAPM: %w", err)
	}

	imNodes := map[string]string{}
	for i := range ims.Items {
		if s := &ims.Items[i].Status; s.BaseOneAgentStatus.EnvironmentID != "" {
			for key := range s.OneAgentStatus.Instances {
				imNodes[key] = s.BaseOneAgentStatus.EnvironmentID
			}
		}
	}

	var tkns corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{Name: utils.GetTokensName(apm), Namespace: r.namespace}, &tkns); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to query tokens: %w", err)
	}

	script, err := newScript(ctx, r.client, apm, tkns, imNodes, r.namespace)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to generate init script: %w", err)
	}
	script.AddNodeProps = r.addNodeProps

	data, err := script.generate()
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to generate script: %w", err)
	}

	// The default cache-based Client doesn't support cross-namespace queries, unless configured to do so in Manager
	// Options. However, this is our only use-case for it, so using the non-cached Client instead.
	err = utils.CreateOrUpdateSecretIfNotExists(r.client, r.apiReader, webhook.SecretConfigName, targetNS, data, corev1.SecretTypeOpaque, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	if apm.Spec.OneAgentCodeModule.Image == "" {
		pullSecretData, err := r.pullSecretGeneratorFunc(r.client, apm, &tkns)
		if err != nil {
			return reconcile.Result{}, err
		}
		err = utils.CreateOrUpdateSecretIfNotExists(r.client, r.apiReader, webhook.PullSecretName, targetNS, pullSecretData, corev1.SecretTypeDockerConfigJson, log)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}

type script struct {
	DynaKube     *dynatracev1alpha1.DynaKube
	PaaSToken    string
	Proxy        string
	TrustedCAs   []byte
	ClusterID    string
	AddNodeProps bool
	IMNodes      map[string]string
}

func newScript(ctx context.Context, c client.Client, dynaKube dynatracev1alpha1.DynaKube, tkns corev1.Secret, imNodes map[string]string, ns string) (*script, error) {
	var kubeSystemNS corev1.Namespace
	if err := c.Get(ctx, client.ObjectKey{Name: "kube-system"}, &kubeSystemNS); err != nil {
		return nil, fmt.Errorf("failed to query for cluster ID: %w", err)
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

var scriptTmpl = template.Must(template.New("initScript").Parse(`#!/usr/bin/env bash

set -eu

if [[ -f "/var/lib/dynatrace/oneagent/agent/config/ruxithost.id" ]]; then
	echo "WARNING: full-stack OneAgent has been injected to this container. App-only and full-stack injection can conflict with each other."
fi

api_url="{{.OneAgent.Spec.APIURL}}"
config_dir="/mnt/config"
target_dir="/mnt/oneagent"
paas_token="{{.PaaSToken}}"
proxy="{{.Proxy}}"
skip_cert_checks="{{if .OneAgent.Spec.SkipCertCheck}}true{{else}}false{{end}}"
custom_ca="{{if .TrustedCAs}}true{{else}}false{{end}}"
fail_code=0
cluster_id="{{.ClusterID}}"

{{- if .AddNodeProps}}
declare -A im_nodes
im_nodes=(
	{{- range $node, $tenant := .IMNodes}}
	["{{$node}}"]="{{$tenant}}"
	{{- end}}
)

set +u
host_tenant="${im_nodes[${K8S_NODE_NAME}]}"
set -u
{{- end}}

archive=$(mktemp)

if [[ "${FAILURE_POLICY}" == "fail" ]]; then
	fail_code=1
fi

if [[ "${INSTALLER_URL}" != "" ]] || [[ "${USE_IMMUTABLE_IMAGE}" != "true" ]]; then
	curl_params=(
		"--silent"
		"--output" "${archive}"
	)

	if [[ "${INSTALLER_URL}" != "" ]]; then
		curl_params+=("${INSTALLER_URL}")
	else
		curl_params+=(
			"${api_url}/v1/deployment/installer/agent/unix/paas/latest?flavor=${FLAVOR}&include=${TECHNOLOGIES}&bitness=64"
			"--header" "Authorization: Api-Token ${paas_token}"
		)
	fi

	if [[ "${skip_cert_checks}" == "true" ]]; then
		curl_params+=("--insecure")
	fi

	if [[ "${custom_ca}" == "true" ]]; then
		curl_params+=("--cacert" "${config_dir}/ca.pem")
	fi

	if [[ "${proxy}" != "" ]]; then
		curl_params+=("--proxy" "${proxy}")
	fi

	echo "Downloading OneAgent package..."
	if ! curl "${curl_params[@]}"; then
		echo "Failed to download the OneAgent package."
		exit "${fail_code}"
	fi

	echo "Unpacking OneAgent package..."
	if ! unzip -o -d "${target_dir}" "${archive}"; then
		echo "Failed to unpack the OneAgent package."
		mv "${archive}" "${target_dir}/package.zip"
		exit "${fail_code}"
	fi
else
	echo "Copy OneAgent package..."
	if ! cp -r "/opt/dynatrace/oneagent/." "${target_dir}"; then
		echo "Failed to copy the OneAgent package."
		exit "${fail_code}"
	fi
fi

echo "Configuring OneAgent..."
echo -n "${INSTALLPATH}/agent/lib64/liboneagentproc.so" >> "${target_dir}/ld.so.preload"

for i in $(seq 1 $CONTAINERS_COUNT)
do
	container_name_var="CONTAINER_${i}_NAME"
	container_image_var="CONTAINER_${i}_IMAGE"

	container_name="${!container_name_var}"
	container_image="${!container_image_var}"

	container_conf_file="${target_dir}/container_${container_name}.conf"

	echo "Writing ${container_conf_file} file..."
	cat <<EOF >${container_conf_file}
[container]
containerName ${container_name}
imageName ${container_image}
k8s_fullpodname ${K8S_PODNAME}
k8s_poduid ${K8S_PODUID}
k8s_containername ${container_name}
k8s_basepodname ${K8S_BASEPODNAME}
k8s_namespace ${K8S_NAMESPACE}
EOF

{{- if .AddNodeProps}}
	if [[ ! -z "${host_tenant}" ]]; then		
		if [[ "{{.OneAgent.Status.EnvironmentID}}" == "${host_tenant}" ]]; then
			cat <<EOF >>${container_conf_file}
k8s_node_name ${K8S_NODE_NAME}
k8s_cluster_id ${cluster_id}
EOF
		fi

		cat <<EOF >>${container_conf_file}

[host]
tenant ${host_tenant}
EOF
	fi
{{- end}}
done
`))

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
