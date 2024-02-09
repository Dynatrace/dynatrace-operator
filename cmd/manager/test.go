package manager

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type TestManager struct {
	manager.Manager
}

func (mgr *TestManager) GetClient() client.Client {
	return struct{ client.Client }{}
}

func (mgr *TestManager) GetAPIReader() client.Reader {
	return nil
}

func (mgr *TestManager) GetControllerOptions() config.Controller {
	return config.Controller{}
}

func (mgr *TestManager) GetScheme() *runtime.Scheme {
	return scheme.Scheme
}

func (mgr *TestManager) GetCache() cache.Cache {
	return nil
}

func (mgr *TestManager) GetLogger() logr.Logger {
	return logger.Get().WithName("test-manager").Logger
}

func (mgr *TestManager) GetRESTMapper() meta.RESTMapper {
	return nil
}

func (mgr *TestManager) Add(manager.Runnable) error {
	return nil
}

func (mgr *TestManager) Start(_ context.Context) error {
	return nil
}
