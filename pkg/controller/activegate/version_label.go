package activegate

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dao"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dtversion"
	corev1 "k8s.io/api/core/v1"
)

func (r *ReconcileActiveGate) setVersionLabel(pods []corev1.Pod) error {
	for i := range pods {
		pod := &pods[i]
		for _, status := range pod.Status.ContainerStatuses {
			if status.Image == "" {
				// If Image is not present, skip
				continue
			}

			imagePullSecret, err := dao.GetImagePullSecret(r.client, pod)
			if err != nil {
				// Something wrong with pull secret, exit function entirely
				return err
			}

			dockerConfig, err := dtversion.NewDockerConfig(imagePullSecret)
			// If an error is returned, try getting the image anyway

			versionLabel, err2 := dtversion.GetVersionLabel(status.Image, dockerConfig)
			if err2 != nil && err != nil {
				// If an error is returned when getting labels and an error occurred during parsing of the docker config
				// assume the error from parsing the docker config is the reason
				return err
			} else if err2 != nil {
				return err2
			}

			pod.Labels[dtversion.VersionKey] = versionLabel
		}
		err := r.client.Update(context.TODO(), pod)
		if err != nil {
			return err
		}
	}

	return nil
}
