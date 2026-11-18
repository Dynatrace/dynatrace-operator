// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"context"
	"errors"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ConditionType is the DynaKube condition that reflects the state of the operator root CA.
	ConditionType = "TLSCA"

	ReasonCAReady    = "CAReady"
	ReasonCAInvalid  = "CAInvalid"
	ReasonCANotFound = "CANotFound"
)

// VerifyCA checks the state of the operator root CA and reflects it in the TLSCA condition of the DynaKube.
// It never creates or modifies the CA, that is the job of the tls Controller, and it never reads the DynaKube spec,
// the CA exists in every TLS mode. Only unexpected errors, like Kubernetes API errors, are returned.
func VerifyCA(ctx context.Context, clt client.Client, dk *dynakube.DynaKube) error {
	log := logd.FromContext(ctx)

	// TODO: Add check for spec.tls

	_, err := loadCA(ctx, clt, dk.Namespace, time.Now())
	secretRef := dk.Namespace + "/" + CASecretName

	switch {
	case err == nil:
		_ = meta.SetStatusCondition(dk.Conditions(), metav1.Condition{
			Type:    ConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  ReasonCAReady,
			Message: "Operator root CA " + secretRef + " is valid",
		})

		return nil
	case errors.Is(err, ErrCANotFound):
		log.Info("operator root CA not found, waiting for it to be created", "secret", secretRef)

		_ = meta.SetStatusCondition(dk.Conditions(), metav1.Condition{
			Type:    ConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonCANotFound,
			Message: "Operator root CA " + secretRef + " not found",
		})

		return nil
	case errors.Is(err, certificates.ErrInvalidCA):
		log.Info("operator root CA is invalid, waiting for it to be regenerated", "secret", secretRef, "reason", err.Error())

		_ = meta.SetStatusCondition(dk.Conditions(), metav1.Condition{
			Type:    ConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonCAInvalid,
			Message: err.Error(),
		})

		return nil
	default:
		k8sconditions.SetKubeAPIError(dk.Conditions(), ConditionType, err)

		return err
	}
}
