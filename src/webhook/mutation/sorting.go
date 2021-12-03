package mutation

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
)

type ByEnv []corev1.EnvVar

func (e ByEnv) Len() int {
	return len(e)
}

func (e ByEnv) Less(i, j int) bool {
	if e[i].Name == e[j].Name {
		if e[i].Value == e[j].Value {
			return e[i].ValueFrom.String() < e[j].ValueFrom.String()
		}
		return e[i].Value < e[j].Value
	}
	return e[i].Name < e[j].Name
}

func (e ByEnv) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

type ByVolume []corev1.Volume

func (v ByVolume) Len() int {
	return len(v)
}

func (v ByVolume) Less(i, j int) bool {
	return v[i].Name < v[j].Name
}

func (v ByVolume) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

type ByVolumeMount []corev1.VolumeMount

func (vm ByVolumeMount) Len() int {
	return len(vm)
}

func (vm ByVolumeMount) Less(i, j int) bool {
	return vm[i].Name < vm[j].Name
}

func (vm ByVolumeMount) Swap(i, j int) {
	vm[i], vm[j] = vm[j], vm[i]
}

func sortPodInternals(pod *corev1.Pod) {
	sort.Sort(ByVolume(pod.Spec.Volumes))

	for _, container := range pod.Spec.InitContainers {
		doSort(container)
	}
	for _, container := range pod.Spec.Containers {
		doSort(container)
	}
}

func doSort(c corev1.Container) {
	sort.Sort(ByVolumeMount(c.VolumeMounts))
	sort.Sort(ByEnv(c.Env))
}
