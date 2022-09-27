package nodes

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	logStoreName = "log-store"
	namespace    = "dynatrace"

	logKey = "log"
)

type logStore struct {
	apiReader client.Reader
	clt       client.Client
	disabled  bool
}

func newLogStore(apiReader client.Reader, clt client.Client) *logStore {
	return &logStore{
		apiReader: apiReader,
		clt:       clt,
	}
}

func (store *logStore) createIfNotExist(ctx context.Context) error {
	log.WithName("logStore").Info("create if not exist")

	var storeMap v1.ConfigMap
	err := store.apiReader.Get(ctx, client.ObjectKey{Name: logStoreName, Namespace: namespace}, &storeMap)

	if err != nil && k8serrors.IsNotFound(err) {
		return store.create(ctx)
	} else if err != nil {
		log.WithName("logStore").Info("disabled due to error", "error", err.Error())
		store.disabled = true
	}

	return errors.WithStack(err)
}

func (store *logStore) create(ctx context.Context) error {
	log.WithName("logStore").Info("create")
	if store.disabled {
		log.WithName("logStore").Info("store is disabled")
		return nil
	}

	err := store.clt.Create(ctx, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      logStoreName,
			Namespace: namespace,
		},
		Data: map[string]string{
			time.Now().Format(getTimeStampAsKey()): "creating log",
		},
	})

	if err != nil {
		store.disabled = true
		log.WithName("logStore").Info("store is disabled due to error")
	}

	return errors.WithStack(err)
}

func (store *logStore) info(ctx context.Context, message string, values ...interface{}) error {
	log.WithName("logStore").Info("info")
	if store.disabled {
		log.WithName("logStore").Info("store is disabled")
		return nil
	}

	storeMap, err := store.get(ctx)

	if err != nil {
		return err
	}

	storeMap.Data[getTimeStampAsKey()] = fmt.Sprintf(message, values)

	err = store.save(ctx, storeMap)
	return err
}

func (store *logStore) get(ctx context.Context) (v1.ConfigMap, error) {
	log.WithName("logStore").Info("get")
	if store.disabled {
		log.WithName("logStore").Info("store is disabled")
		return v1.ConfigMap{}, nil
	}

	var storeMap v1.ConfigMap
	err := store.apiReader.Get(ctx, client.ObjectKey{Name: logStoreName, Namespace: namespace}, &storeMap)

	return storeMap, errors.WithStack(err)
}

func (store *logStore) save(ctx context.Context, storeMap v1.ConfigMap) error {
	log.WithName("logStore").Info("save")
	if store.disabled {
		log.WithName("logStore").Info("store is disabled")
		return nil
	}

	err := store.clt.Update(ctx, &storeMap)

	return errors.WithStack(err)
}

func getTimeStampAsKey() string {
	return time.Now().Format("2006-01-02_15.04.05.000")
}
