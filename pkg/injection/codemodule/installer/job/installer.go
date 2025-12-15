package job

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	jobsettings "github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/job/settings"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sjob"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Properties struct {
	CSIJob       jobsettings.Settings
	Owner        client.Object
	APIReader    client.Reader
	Client       client.Client
	ImageURI     string
	PathResolver metadata.PathResolver
	PullSecrets  []string
}

func NewInstaller(ctx context.Context, props *Properties) installer.Installer {
	return &Installer{
		props:    props,
		nodeName: k8senv.GetNodeName(),
	}
}

type Installer struct {
	props    *Properties
	nodeName string
}

func (inst *Installer) InstallAgent(ctx context.Context, targetDir string) (bool, error) {
	log.Info("installing agent via Job", "image", inst.props.ImageURI, "target dir", targetDir)

	err := os.MkdirAll(inst.props.PathResolver.AgentSharedBinaryDirBase(), common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create the base shared agent directory", "err", err)

		return false, errors.WithStack(err)
	}

	jobName := inst.buildJobName()

	ready, err := inst.isReady(ctx, targetDir, jobName)
	if err != nil {
		return false, err
	}

	if !ready {
		return false, nil
	}

	if err := symlink.CreateForCurrentVersionIfNotExists(targetDir); err != nil {
		_ = os.RemoveAll(targetDir)

		log.Info("failed to create symlink for agent installation", "err", err)

		return false, errors.WithStack(err)
	}

	return true, nil
}

func (inst *Installer) isReady(ctx context.Context, targetDir, jobName string) (bool, error) {
	if inst.isAlreadyPresent(targetDir) {
		log.Info("agent already installed", "image", inst.props.ImageURI, "target dir", targetDir)

		_ = os.RemoveAll(inst.props.PathResolver.AgentJobWorkDirForJob(jobName))

		return true, inst.query().DeleteForNamespace(ctx, jobName, inst.props.Owner.GetNamespace(), &client.DeleteOptions{PropagationPolicy: ptr.To(metav1.DeletePropagationBackground)})
	}

	job, err := inst.query().Get(ctx, types.NamespacedName{Name: jobName, Namespace: inst.props.Owner.GetNamespace()})
	if err != nil && !k8serrors.IsNotFound(err) {
		log.Info("failed to determine the status of the download job", "err", err)

		return false, err
	} else if err == nil {
		log.Info("job is not finished", "job", jobName)

		if job.Status.Failed > 0 {
			return false, errors.Errorf("the job is failing; job: %s", jobName)
		}

		return false, nil
	}

	log.Info("creating new download job", "job", jobName)

	job, err = inst.buildJob(jobName, targetDir)
	if err != nil {
		return false, err
	}

	return false, inst.query().WithOwner(inst.props.Owner).Create(ctx, job)
}

func (inst *Installer) isAlreadyPresent(targetDir string) bool {
	_, err := os.Stat(targetDir)

	return !os.IsNotExist(err)
}

func (inst *Installer) query() k8sjob.QueryObject {
	return k8sjob.Query(inst.props.Client, inst.props.APIReader, log)
}
