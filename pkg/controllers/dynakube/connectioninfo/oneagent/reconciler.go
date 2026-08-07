// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oaconnectioninfo

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client       client.Client
	timeProvider *timeprovider.Provider
	secrets      k8ssecret.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		client:       clt,
		timeProvider: timeprovider.New(),
		secrets:      k8ssecret.Query(clt, apiReader),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, oaClient oneagent.Client, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "connectioninfo")

	if !dk.OneAgent().IsAppInjectionNeeded() && !dk.OneAgent().IsDaemonsetRequired() && !dk.LogMonitoring().IsEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		err := r.secrets.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: dk.OneAgent().GetTenantSecret(), Namespace: dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up OneAgent tenant-secret")
		}

		meta.RemoveStatusCondition(dk.Conditions(), oaConnectionInfoConditionType)
		dk.Status.OneAgent.ConnectionInfo = communication.ConnectionInfo{}

		return nil // clean-up shouldn't cause a failure
	}

	oldStatus := dk.Status.DeepCopy()

	err := r.reconcileConnectionInfo(ctx, oaClient, dk)
	if err != nil {
		return err
	}

	needStatusUpdate, err := hasher.IsDifferent(oldStatus, dk.Status)
	if err != nil {
		return errors.WithMessage(err, "failed to compare connection info status hashes")
	} else if needStatusUpdate {
		err = dk.UpdateStatus(ctx, r.client)
	}

	return err
}

func (r *Reconciler) reconcileConnectionInfo(ctx context.Context, oaClient oneagent.Client, dk *dynakube.DynaKube) error {
	log := logd.FromContext(ctx)

	connectionInfo, err := oaClient.GetConnectionInfo(ctx, getRequiredActiveGateServiceIPs(dk))
	if err != nil && !IsPostponedError(err) {
		k8sconditions.SetDynatraceAPIError(dk.Conditions(), oaConnectionInfoConditionType, err)

		return errors.WithMessage(err, "failed to get OneAgent connection info")
	}

	r.setDynakubeStatus(dk, connectionInfo)
	log.Info("OneAgent connection info updated")

	if errors.Is(err, oneagent.NoCommunicationEndpointsError) {
		log.Info("no received OneAgent connection info, tenant API requests not yet throttled", "tenant", connectionInfo.TenantUUID)
		setEmptyCommunicationHostsCondition(dk.Conditions())

		return oneagent.NoCommunicationEndpointsError
	}

	if errors.Is(err, oneagent.StaleNetworkZoneEndpointsError) {
		log.Info("OneAgent endpoints do not contain the local ActiveGate Service IP yet, postponing OneAgent deployment",
			"tenant", connectionInfo.TenantUUID,
			"endpoints", connectionInfo.Endpoints,
			"serviceIPs", dk.Status.ActiveGate.ServiceIPs)
		setStaleNetworkZoneEndpointsCondition(dk.Conditions())

		return oneagent.StaleNetworkZoneEndpointsError
	}

	err = r.createTenantTokenSecret(ctx, dk, dk.OneAgent().GetTenantSecret(), connectionInfo)
	if err != nil {
		return err
	}

	dk.Status.OneAgent.ConnectionInfo.TenantTokenHash, err = hasher.GenerateHash(connectionInfo.TenantToken)
	if err != nil {
		return errors.Wrap(err, "failed to generate TenantTokenHash")
	}

	log.Info("received OneAgent connection info", "communication endpoints", connectionInfo.Endpoints, "tenant", connectionInfo.TenantUUID)

	return nil
}

// IsPostponedError reports whether the error indicates a transient
// OneAgent connection-info state that resolves itself once the local ActiveGate is
// ready or has re-registered. Callers treat these as "not yet ready" and trigger a
// fast requeue instead of surfacing them as reconcile failures.
func IsPostponedError(err error) bool {
	return errors.Is(err, oneagent.NoCommunicationEndpointsError) || errors.Is(err, oneagent.StaleNetworkZoneEndpointsError)
}

func (r *Reconciler) createTenantTokenSecret(ctx context.Context, dk *dynakube.DynaKube, secretName string, connectionInfo oneagent.ConnectionInfo) error {
	log := logd.FromContext(ctx)

	secret, err := connectioninfo.BuildTenantSecret(dk, k8slabel.OneAgentComponentLabel, secretName, connectionInfo.TenantToken)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = r.secrets.CreateOrUpdate(ctx, secret)
	if err != nil {
		log.Info("could not create or update secret for connection info", "secretName", secret.Name)
		k8sconditions.SetKubeAPIError(dk.Conditions(), oaConnectionInfoConditionType, err)

		return err
	}

	k8sconditions.SetSecretCreated(dk.Conditions(), oaConnectionInfoConditionType, secret.Name)

	return nil
}

func (r *Reconciler) setDynakubeStatus(dk *dynakube.DynaKube, connectionInfo oneagent.ConnectionInfo) {
	dk.Status.OneAgent.ConnectionInfo.TenantUUID = connectionInfo.TenantUUID
	dk.Status.OneAgent.ConnectionInfo.Endpoints = connectionInfo.Endpoints
}

func getRequiredActiveGateServiceIPs(dk *dynakube.DynaKube) []string {
	if dk == nil || dk.Spec.NetworkZone == "" || !dk.ActiveGate().IsRoutingEnabled() || len(dk.Status.ActiveGate.ServiceIPs) == 0 {
		return nil
	}

	return dk.Status.ActiveGate.ServiceIPs
}
