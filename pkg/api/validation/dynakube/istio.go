// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/utils/ptr"
)

const (
	errorNoIstioInstalled = `No resources for istio available`
)

var virtualServiceGVK = istiov1beta1.SchemeGroupVersion.WithKind("VirtualService")

func isIstioNotInstalled(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if ptr.Deref(dk.Spec.EnableIstio, false) && !k8scrd.IsInstalled(ctx, dv.apiReader, virtualServiceGVK) {
		return errorNoIstioInstalled
	}

	return ""
}
