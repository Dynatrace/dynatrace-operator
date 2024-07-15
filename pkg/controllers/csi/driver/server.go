/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package csidriver

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes/app"
	hostvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/driver/volumes/host"
	csiotel "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/internal/otel"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtotel"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Server struct {
	csi.UnimplementedIdentityServer
	csi.UnimplementedNodeServer

	fs      afero.Afero
	mounter mount.Interface
	db      metadata.Access

	publishers map[string]csivolumes.Publisher
	opts       dtcsi.CSIOptions
	path       metadata.PathResolver
}

var _ csi.IdentityServer = &Server{}
var _ csi.NodeServer = &Server{}

func NewServer(opts dtcsi.CSIOptions, db metadata.Access) *Server {
	return &Server{
		opts:    opts,
		fs:      afero.Afero{Fs: afero.NewOsFs()},
		mounter: mount.New(""),
		db:      db,
		path:    metadata.PathResolver{RootDir: opts.RootDir},
	}
}

func (svr *Server) SetupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(svr)
}

func (svr *Server) Start(ctx context.Context) error {
	defer metadata.LogAccessOverview(svr.db)

	proto, addr, err := parseEndpoint(svr.opts.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse endpoint '%s': %w", svr.opts.Endpoint, err)
	}

	if proto == "unix" {
		if err := svr.fs.Remove(addr); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove old endpoint on '%s': %w", addr, err)
		}
	}

	svr.publishers = map[string]csivolumes.Publisher{
		appvolumes.Mode:  appvolumes.NewAppVolumePublisher(svr.fs, svr.mounter, svr.db, svr.path),
		hostvolumes.Mode: hostvolumes.NewHostVolumePublisher(svr.fs, svr.mounter, svr.db, svr.path),
	}

	log.Info("starting listener", "protocol", proto, "address", addr)

	listener, err := net.Listen(proto, addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(logGRPC()))

	go func() {
		ticker := time.NewTicker(memoryMetricTick)

		done := false
		for !done {
			select {
			case <-ctx.Done():
				log.Info("stopping server")
				server.GracefulStop()
				log.Info("stopped server")

				done = true
			case <-ticker.C:
				var m runtime.MemStats

				runtime.ReadMemStats(&m)
				memoryUsageMetric.Set(float64(m.Alloc))
			}
		}
	}()

	csi.RegisterIdentityServer(server, svr)
	csi.RegisterNodeServer(server, svr)

	log.Info("listening for connections on address", "address", listener.Addr())

	err = server.Serve(listener)
	server.GracefulStop()

	return err
}

func (svr *Server) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{Name: dtcsi.DriverName, VendorVersion: version.Version}, nil
}

func (svr *Server) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{}, nil
}

func (svr *Server) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{}, nil
}

func (svr *Server) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer, csiotel.SpanOptions()...)
	defer span.End()

	volumeCfg, err := csivolumes.ParseNodePublishVolumeRequest(req)
	if err != nil {
		return nil, dtotel.RecordError(span, err)
	}

	if volumeCfg.OtelSpanContext != nil {
		span.AddLink(trace.Link{SpanContext: *volumeCfg.OtelSpanContext})
	}

	if isMounted, err := isMounted(svr.mounter, volumeCfg.TargetPath); err != nil {
		return nil, dtotel.RecordError(span, err)
	} else if isMounted {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	publisher, ok := svr.publishers[volumeCfg.Mode]
	if !ok {
		return nil, status.Error(codes.Internal, "unknown csi mode provided, mode="+volumeCfg.Mode)
	}

	logRequest(volumeCfg, req, span)

	return publisher.PublishVolume(ctx, *volumeCfg)
}

func (svr *Server) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (response *csi.NodeUnpublishVolumeResponse, err error) {
	ctx, span := dtotel.StartSpan(ctx, csiotel.Tracer, csiotel.SpanOptions()...)
	defer span.End()

	volumeInfo, err := csivolumes.ParseNodeUnpublishVolumeRequest(req)
	if err != nil {
		return nil, dtotel.RecordError(span, err)
	}

	for _, publisher := range svr.publishers {
		canUnpublish, err := publisher.CanUnpublishVolume(ctx, *volumeInfo)
		if err != nil {
			log.Error(dtotel.RecordError(span, err), "couldn't determine if volume can be unpublished", "publisher", publisher)
		}

		if canUnpublish {
			response, err := publisher.UnpublishVolume(ctx, *volumeInfo)
			if err != nil {
				log.Error(dtotel.RecordError(span, err), "couldn't unpublish volume properly", "publisher", publisher)
			}

			return response, nil
		}
	}

	svr.unmountUnknownVolume(*volumeInfo)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (svr *Server) unmountUnknownVolume(volumeInfo csivolumes.VolumeInfo) {
	log.Info("unmounting unknown volume", "volumeID", volumeInfo.VolumeID, "targetPath", volumeInfo.TargetPath)

	if err := svr.mounter.Unmount(volumeInfo.TargetPath); err != nil {
		log.Error(err, "failed to unmount unknown volume", "volumeID", volumeInfo.VolumeID)
	}
}

func (svr *Server) NodeStageVolume(context.Context, *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (svr *Server) NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (svr *Server) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{NodeId: svr.opts.NodeId}, nil
}

func (svr *Server) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{Capabilities: []*csi.NodeServiceCapability{}}, nil
}

func (svr *Server) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (svr *Server) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func isMounted(mounter mount.Interface, targetPath string) (bool, error) {
	isNotMounted, err := mount.IsNotMountPoint(mounter, targetPath)
	if os.IsNotExist(err) {
		isNotMounted = true
	} else if err != nil {
		return false, status.Error(codes.Internal, err.Error())
	}

	return !isNotMounted, nil
}

func logRequest(volumeCfg *csivolumes.VolumeConfig, req *csi.NodePublishVolumeRequest, span trace.Span) {
	log.Info("publishing volume",
		"csiMode", volumeCfg.Mode,
		"target", volumeCfg.TargetPath,
		"fstype", req.GetVolumeCapability().GetMount().GetFsType(),
		"readonly", req.GetReadonly(),
		"volumeID", volumeCfg.VolumeID,
		"attributes", req.GetVolumeContext(),
		"mountflags", req.GetVolumeCapability().GetMount().GetMountFlags(),
	)

	span.SetAttributes(
		attribute.KeyValue{
			Key:   "csi.mode",
			Value: attribute.StringValue(volumeCfg.Mode),
		},
		attribute.KeyValue{
			Key:   "csi.target",
			Value: attribute.StringValue(volumeCfg.TargetPath),
		},
		attribute.KeyValue{
			Key:   "csi.fstype",
			Value: attribute.StringValue(req.GetVolumeCapability().GetMount().GetFsType()),
		},
		attribute.KeyValue{
			Key:   "csi.readonly",
			Value: attribute.BoolValue(req.GetReadonly()),
		},
		attribute.KeyValue{
			Key:   "csi.volumeID",
			Value: attribute.StringValue(volumeCfg.VolumeID),
		},
		attribute.KeyValue{
			Key:   "csi.mountflags",
			Value: attribute.StringSliceValue(req.GetVolumeCapability().GetMount().GetMountFlags()),
		},
	)
}

func logGRPC() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod == "/csi.v1.Identity/Probe" || info.FullMethod == "/csi.v1.Node/NodeGetCapabilities" {
			return handler(ctx, req)
		}

		methodName := ""

		if info.FullMethod == "/csi.v1.Node/NodePublishVolume" {
			req := req.(*csi.NodePublishVolumeRequest)
			methodName = "NodePublishVolume"
			log.Info("GRPC call", "method", methodName, "volume-id", req.GetVolumeId())
		} else if info.FullMethod == "/csi.v1.Node/NodeUnpublishVolume" {
			req := req.(*csi.NodeUnpublishVolumeRequest)
			methodName = "NodeUnpublishVolume"
			log.Info("GRPC call", "method", methodName, "volume-id", req.GetVolumeId())
		}

		resp, err := handler(ctx, req)
		if err != nil {
			log.Error(err, "GRPC call failed", "method", methodName)
		}

		return resp, err
	}
}

func parseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}

	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}
