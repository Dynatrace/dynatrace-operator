package activegate

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dao"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/parser"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/version"
	corev1 "k8s.io/api/core/v1"
)

func (r *ReconcileActiveGate) updateVersionLabel(pods []corev1.Pod) error {
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

			dockerConfig, err := parser.NewDockerConfig(imagePullSecret)
			// If an error is returned, try getting the image anyway

			labels, err2 := dao.GetImageLabels(status.Image, dockerConfig)
			if err2 != nil && err != nil {
				// If an error is returned when getting labels and an error occurred during parsing of the docker config
				// assume the error from parsing the docker config is the reason
				return err
			} else if err2 != nil {
				return err2
			}

			if _, hasImageVersionLabel := labels[version.VersionKey]; !hasImageVersionLabel {
				return fmt.Errorf("image has no version label")
			}

			pod.Labels[version.VersionKey] = labels[version.VersionKey]
		}
		err := r.client.Update(context.TODO(), pod)
		if err != nil {
			return err
		}
	}

	return nil
}
