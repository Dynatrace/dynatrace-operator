package oneagent

import corev1 "k8s.io/api/core/v1"

func HasPodUserSet(ctx *corev1.PodSecurityContext) bool {
	return ctx != nil && ctx.RunAsUser != nil
}

func HasPodGroupSet(ctx *corev1.PodSecurityContext) bool {
	return ctx != nil && ctx.RunAsGroup != nil
}

func IsNonRoot(ctx *corev1.SecurityContext) bool {
	return ctx != nil &&
		(ctx.RunAsUser != nil && *ctx.RunAsUser != RootUserGroup) &&
		(ctx.RunAsGroup != nil && *ctx.RunAsGroup != RootUserGroup)
}
