package livenessprobe

import (
	"net/http"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/rpc"
	"golang.org/x/net/context"
)

const (
	defaultProbeTimeout = 9 * time.Second
)

var (
	log = logd.Get().WithName("csi-livenessprobe")
)

type Server struct {
	endpoint     string
	healthPort   string
	driverName   string
	probeTimeout time.Duration
}

func NewServer(driverName string, endpoint string, healthPort string, probeTimeout string) *Server {
	probeTimeoutDuration, err := time.ParseDuration(probeTimeout)
	if err != nil {
		log.Error(err, "unable to parse probe timeout duration. Value set to 9s", "probe timeout", probeTimeout)

		probeTimeoutDuration = defaultProbeTimeout
	}

	return &Server{
		endpoint:     endpoint,
		healthPort:   healthPort,
		driverName:   driverName,
		probeTimeout: probeTimeoutDuration,
	}
}

func (srv *Server) Start(ctx context.Context) error {
	log.Info("starting livenessprobe")

	conn, err := connection.Connect(ctx, srv.endpoint, nil, connection.WithTimeout(0))
	if err != nil {
		log.Error(err, "failed to establish connection to CSI driver")

		return err
	}

	driverName, err := rpc.GetDriverName(ctx, conn)
	conn.Close()

	if err != nil {
		log.Error(err, "failed to get driver name")

		return err
	}

	log.Info("starting server", "driver name", driverName)

	http.HandleFunc(" /healthz", srv.probeRequest)

	httpServer := &http.Server{
		Addr:              ":" + srv.healthPort,
		ReadHeaderTimeout: 3 * time.Second,
	}

	return httpServer.ListenAndServe()
}

func (srv *Server) probeRequest(w http.ResponseWriter, r *http.Request) {
	log.Debug("probeRequest")

	ctx, cancelFunc := context.WithTimeout(r.Context(), srv.probeTimeout)
	defer cancelFunc()

	conn, err := connection.Connect(ctx, srv.endpoint, nil, connection.WithTimeout(srv.probeTimeout))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Error(err, "failed to establish connection to CSI driver")

		return
	}
	defer conn.Close()

	log.Debug("sending probe request to CSI driver")

	ready, err := rpc.Probe(ctx, conn)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Error(err, "health check failed")

		return
	}

	if !ready {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("driver is not ready"))
		log.Error(nil, "driver is not ready")

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`ok`))
	log.Debug("health check succeeded")
}
