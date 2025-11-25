package nodes

import (
	"context"
	"os"
	"slices"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/nodes/cache"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Controller struct {
	client                 client.Client
	apiReader              client.Reader
	dynatraceClientBuilder dynatraceclient.Builder
	timeProvider           *timeprovider.Provider
	podNamespace           string
	runLocal               bool
}

func Add(mgr manager.Manager, _ string) error {
	return NewController(mgr).SetupWithManager(mgr)
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Named("nodes-controller").
		Complete(controller)
}

func NewController(mgr manager.Manager) *Controller {
	return &Controller{
		client:                 mgr.GetClient(),
		apiReader:              mgr.GetAPIReader(),
		dynatraceClientBuilder: dynatraceclient.NewBuilder(mgr.GetAPIReader()),
		runLocal:               kubesystem.IsRunLocally(),
		podNamespace:           os.Getenv(k8senv.PodNamespace),
		timeProvider:           timeprovider.New(),
	}
}

func (controller *Controller) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) { //nolint: revive
	nodeName := request.Name
	dk, err := controller.determineDynakubeForNode(nodeName)

	log.Info("reconciling node name", "node", nodeName)

	if err != nil {
		return reconcile.Result{}, err
	}

	nodeCache, err := controller.getCache(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	var node corev1.Node

	err = controller.apiReader.Get(ctx, client.ObjectKey{Name: nodeName}, &node)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return reconcile.Result{}, err
		}

		err := controller.reconcileNodeDeletion(ctx, nodeCache, nodeName)
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nodeCache.Store(ctx, controller.client)
	} else if dk != nil { // Node is found in the cluster, add or update to cache
		err := controller.reconcileNodeUpdate(ctx, dk, nodeCache, &node)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// check node cache for outdated nodes and remove them, to keep cache clean
	if nodeCache.IsOutdated(controller.timeProvider.Now().UTC()) {
		if err := controller.pruneCache(ctx, nodeCache); err != nil {
			return reconcile.Result{}, err
		}

		nodeCache.UpdateTimestamp(controller.timeProvider.Now().UTC())
	}

	return reconcile.Result{}, nodeCache.Store(ctx, controller.client)
}

func (controller *Controller) reconcileNodeUpdate(ctx context.Context, dk *dynakube.DynaKube, nodeCache *cache.Cache, node *corev1.Node) error {
	nodeName := node.Name
	ipAddress := dk.Status.OneAgent.Instances[nodeName].IPAddress
	cacheEntry := cache.Entry{
		NodeName:     nodeName,
		DynaKubeName: dk.Name,
		IPAddress:    ipAddress,
		LastSeen:     controller.timeProvider.Now().UTC(),
	}

	if cached, err := nodeCache.GetEntry(nodeName); err == nil {
		cacheEntry.SetLastMarkedForTerminationTimestamp(cached.LastMarkedForTermination)
	}

	// Handle unschedulable Nodes, if they have a OneAgent instance
	if isUnschedulable(node) {
		if err := controller.markForTermination(ctx, dk, &cacheEntry); err != nil {
			return err
		}
	}

	if err := nodeCache.SetEntry(nodeName, cacheEntry); err != nil {
		return err
	}

	return nil
}

func (controller *Controller) reconcileNodeDeletion(ctx context.Context, nodeCache *cache.Cache, nodeName string) error {
	dynakube, err := controller.determineDynakubeForNode(nodeName)
	if err != nil {
		return err
	}

	cacheEntry, err := nodeCache.GetEntry(nodeName)
	if err != nil {
		if errors.Is(err, cache.ErrEntryNotFound) {
			// uncached node -> ignoring
			return nil
		}

		return err
	}

	if dynakube != nil {
		if err := controller.markForTermination(ctx, dynakube, &cacheEntry); err != nil {
			return err
		}
	}

	nodeCache.DeleteEntry(nodeName)

	return nil
}

func (controller *Controller) sendMarkedForTermination(ctx context.Context, dk *dynakube.DynaKube, cachedNode *cache.Entry) error {
	tokenReader := token.NewReader(controller.apiReader, dk)

	tokens, err := tokenReader.ReadTokens(ctx)
	if err != nil {
		return err
	}

	dynatraceClient, err := controller.dynatraceClientBuilder.
		SetDynakube(*dk).
		SetTokens(tokens).
		Build(ctx)
	if err != nil {
		return err
	}

	entityID, err := dynatraceClient.GetHostEntityIDForIP(ctx, cachedNode.IPAddress)
	if err != nil {
		if errors.As(err, &dtclient.HostEntityNotFoundErr{}) {
			log.Info("skipping to send mark for termination event", "dynakube", dk.Name, "nodeIP", cachedNode.IPAddress, "reason", err.Error())

			return nil
		}

		if errors.As(err, &dtclient.V1HostEntityAPINotAvailableErr{}) {
			log.Info("skipping to send mark for termination event", "dynakube", dk.Name, "nodeIP", cachedNode.IPAddress, "reason", err.Error())

			return nil
		}

		log.Info("failed to send mark for termination event",
			"reason", "failed to determine entity id", "dynakube", dk.Name, "nodeIP", cachedNode.IPAddress, "cause", err)

		return err
	}

	ts := uint64(cachedNode.LastSeen.Add(-10*time.Minute).UnixNano()) / uint64(time.Millisecond) //nolint:gosec

	err = dynatraceClient.SendEvent(ctx, &dtclient.EventData{
		EventType:     dtclient.MarkedForTerminationEvent,
		Source:        "Dynatrace Operator",
		Description:   "Kubernetes node cordoned. Node might be drained or terminated.",
		StartInMillis: ts,
		EndInMillis:   ts,
		AttachRules: dtclient.EventDataAttachRules{
			EntityIDs: []string{entityID},
		},
	})
	if errors.As(err, &dtclient.V1EventsAPINotAvailableErr{}) {
		log.Info("skipping to send mark for termination event", "dynakube", dk.Name, "nodeIP", cachedNode.IPAddress, "reason", err.Error())

		return nil
	}

	return err
}

func (controller *Controller) markForTermination(ctx context.Context, dk *dynakube.DynaKube, cacheEntry *cache.Entry) error {
	if !cacheEntry.IsMarkableForTermination(controller.timeProvider.Now().UTC()) {
		return nil
	}

	cacheEntry.SetLastMarkedForTerminationTimestamp(controller.timeProvider.Now().UTC())

	log.Info("sending mark for termination event to dynatrace server", "dk", dk.Name, "ip", cacheEntry.IPAddress,
		"node", cacheEntry.NodeName)

	return controller.sendMarkedForTermination(ctx, dk, cacheEntry)
}

func isUnschedulable(node *corev1.Node) bool {
	return node.Spec.Unschedulable || hasUnschedulableTaint(node)
}

func hasUnschedulableTaint(node *corev1.Node) bool {
	for _, taint := range node.Spec.Taints {
		if slices.Contains(unschedulableTaints, taint.Key) {
			return true
		}
	}

	return false
}

func (controller *Controller) getCache(ctx context.Context) (*cache.Cache, error) {
	var owner client.Object

	if !controller.runLocal {
		deploy, err := k8sdeployment.GetDeployment(controller.client, os.Getenv(k8senv.PodName), controller.podNamespace)
		if err != nil {
			return nil, err
		}

		owner = deploy
	}

	return cache.New(ctx, controller.apiReader, controller.podNamespace, owner)
}

func (controller *Controller) pruneCache(ctx context.Context, nodeCache *cache.Cache) error {
	missingCachedNodes, err := nodeCache.Prune(ctx, controller.client, controller.timeProvider.Now().UTC())
	if err != nil {
		return err
	}

	for _, nodeName := range missingCachedNodes {
		log.Info("Removing missing cached node from cluster", "node", nodeName)

		err := controller.reconcileNodeDeletion(ctx, nodeCache, nodeName)
		if err != nil {
			return err
		}
	}

	return nil
}
