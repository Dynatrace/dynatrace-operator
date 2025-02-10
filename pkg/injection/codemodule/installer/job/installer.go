package job

import (
	"context"
	"os"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/common"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/codemodule/installer/symlink"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	jobutil "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/job"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Properties struct {
	Owner        client.Object
	ApiReader    client.Reader
	Client       client.Client
	ImageUri     string
	PathResolver metadata.PathResolver
	PullSecrets  []string
}

func NewInstaller(ctx context.Context, fs afero.Fs, props *Properties) installer.Installer {
	return &Installer{
		fs:       fs,
		props:    props,
		nodeName: env.GetNodeName(),
	}
}

type Installer struct {
	fs       afero.Fs
	props    *Properties
	nodeName string
}

func (inst *Installer) InstallAgent(ctx context.Context, targetDir string) (bool, error) {
	log.Info("installing agent via Job", "image", inst.props.ImageUri, "target dir", targetDir)

	err := inst.fs.MkdirAll(inst.props.PathResolver.AgentSharedBinaryDirBase(), common.MkDirFileMode)
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

	if err := symlink.CreateForCurrentVersionIfNotExists(inst.fs, targetDir); err != nil {
		_ = inst.fs.RemoveAll(targetDir)

		log.Info("failed to create symlink for agent installation", "err", err)

		return false, errors.WithStack(err)
	}

	return true, nil
}

func (inst *Installer) isReady(ctx context.Context, targetDir, jobName string) (bool, error) {
	if inst.isAlreadyPresent(targetDir) {
		log.Info("agent already installed", "image", inst.props.ImageUri, "target dir", targetDir)

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

func (installer *Installer) isAlreadyPresent(targetDir string) bool {
	_, err := installer.fs.Stat(targetDir)

	return !os.IsNotExist(err)
}

func (inst *Installer) query() jobutil.QueryObject {
	return jobutil.Query(inst.props.Client, inst.props.ApiReader, log)
}
