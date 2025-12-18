package livenessprobe

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/rpc"
)

var (
	log = logd.Get().WithName("csi-livenessprobe")
)

type Server struct {
	csiAddress   string
	healthPort   string
	driverName   string
	probeTimeout time.Duration
}

func NewServer(driverName string, csiAddress string, healthPort string, probeTimeout time.Duration) *Server {
	return &Server{
		csiAddress:   csiAddress,
		healthPort:   healthPort,
		driverName:   driverName,
		probeTimeout: probeTimeout,
	}
}

func (s *Server) Start(ctx context.Context) error {
	log.Info("starting livenessprobe")

	if err := s.isDriverRunning(ctx); err != nil {
		return err
	}

	http.HandleFunc(" /healthz", s.probeRequest)

	httpServer := &http.Server{
		Addr:              ":" + s.healthPort,
		ReadHeaderTimeout: 3 * time.Second,
	}

	go func() {
		<-ctx.Done()
		log.Info("stopping HTTP server")

		sctx, cancelFunc := context.WithTimeout(context.Background(), s.probeTimeout)
		defer cancelFunc()

		err := httpServer.Shutdown(sctx)
		if err != nil {
			log.Error(err, "failed to shutdown HTTP server")
		}

		log.Info("stopped HTTP server")
	}()

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) probeRequest(w http.ResponseWriter, r *http.Request) {
	log.Debug("probeRequest")

	ctx, cancelFunc := context.WithTimeout(r.Context(), s.probeTimeout)
	defer cancelFunc()

	conn, err := connection.Connect(ctx, s.csiAddress, nil, connection.WithTimeout(s.probeTimeout))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeResponse(w, http.StatusInternalServerError, err.Error())
		log.Error(err, "failed to establish connection to CSI driver")

		return
	}
	defer conn.Close()

	log.Debug("sending probe request to CSI driver")

	ready, err := rpc.Probe(ctx, conn)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeResponse(w, http.StatusInternalServerError, err.Error())
		log.Error(err, "health check failed")

		return
	}

	if !ready {
		w.WriteHeader(http.StatusInternalServerError)
		writeResponse(w, http.StatusInternalServerError, "driver is not ready")
		log.Error(nil, "driver is not ready")

		return
	}

	w.WriteHeader(http.StatusOK)

	writeResponse(w, http.StatusOK, "ok")

	log.Debug("health check succeeded")
}

func writeResponse(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)

	_, err := w.Write([]byte(message))
	if err != nil {
		log.Error(err, "failed to write response", "statusCode", statusCode, "message", message)
	}
}

func (s *Server) isDriverRunning(ctx context.Context) error {
	conn, err := connection.Connect(ctx, s.csiAddress, nil, connection.WithTimeout(0))
	if err != nil {
		log.Error(err, "failed to establish connection to CSI driver")

		return err
	}
	defer conn.Close()

	driverName, err := rpc.GetDriverName(ctx, conn)
	if err != nil {
		log.Error(err, "failed to get driver name")

		return err
	}

	log.Info("CSI driver is running", "driver name", driverName)

	return nil
}
