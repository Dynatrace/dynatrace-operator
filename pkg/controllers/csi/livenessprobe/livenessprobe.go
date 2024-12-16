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
	defaultProbeTimeout = 4 * time.Second
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

func NewServer(endpoint string, healthPort string, probeTimeout string) *Server {
	probeTimeoutDuration, err := time.ParseDuration(probeTimeout)
	if err != nil {
		log.Error(err, "unable to parse probe timeout duration. Value set to 4s", "probe timeout", probeTimeout)

		probeTimeoutDuration = defaultProbeTimeout
	}

	return &Server{
		endpoint:     endpoint,
		healthPort:   healthPort,
		probeTimeout: probeTimeoutDuration,
	}
}

func (svr *Server) Start(ctx context.Context) error {
	csiConn, err := connection.Connect(ctx, svr.endpoint, nil, connection.WithTimeout(0))
	if err != nil {
		// connlib should retry forever so a returned error should mean
		// the grpc client is misconfigured rather than an error on the network or CSI driver.
		log.Error(err, "Failed to establish connection to CSI driver")

		return err
	}

	log.Info("Calling CSI driver to discover driver name")

	csiDriverName, err := rpc.GetDriverName(ctx, csiConn)
	csiConn.Close()

	if err != nil {
		// The CSI driver does not support GetDriverName, which is serious enough to crash the probe.
		log.Error(err, "Failed to get CSI driver name")

		return err
	}

	log.Info("CSI driver name", "driver", csiDriverName)

	svr.driverName = csiDriverName

	http.HandleFunc(" /healthz", svr.probeRequest)

	return http.ListenAndServe("0.0.0.0:"+svr.healthPort, nil)
}

func (svr *Server) probeRequest(w http.ResponseWriter, r *http.Request) {
	log.Info("getProbe /healtz")

	ctx, cancelFunc := context.WithTimeout(r.Context(), svr.probeTimeout)
	defer cancelFunc()

	conn, err := connection.Connect(ctx, svr.endpoint, nil, connection.WithTimeout(svr.probeTimeout))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Error(err, "Failed to establish connection to CSI driver")

		return
	}
	defer conn.Close()

	log.Info("Sending probe request to CSI driver", "driver", svr.driverName)

	ready, err := rpc.Probe(ctx, conn)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Error(err, "Health check failed")

		return
	}

	if !ready {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("driver responded but is not ready"))
		log.Error(nil, "Driver responded but is not ready")

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`ok`))
	log.Info("Health check succeeded")
}
