package csivolumes

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	PodNameContextKey = "csi.storage.k8s.io/pod.name"

	// CSIVolumeAttributeModeField used for identifying the origin of the NodePublishVolume request
	CSIVolumeAttributeModeField     = "mode"
	CSIVolumeAttributeDynakubeField = "dynakube"
	CSIOtelSpanId                   = "otelSpanId"
	CSIOtelTraceId                  = "otelTraceId"
)

// Represents the basic information about a volume
type VolumeInfo struct {
	VolumeID   string
	TargetPath string
}

// Represents the config needed to mount a volume
type VolumeConfig struct {
	OtelSpanContext *trace.SpanContext
	VolumeInfo
	PodName      string
	Mode         string
	DynakubeName string
}

// Transforms the NodePublishVolumeRequest into a VolumeConfig
func ParseNodePublishVolumeRequest(req *csi.NodePublishVolumeRequest) (*VolumeConfig, error) {
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability missing in request")
	}

	volID := req.GetVolumeId()
	if volID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	if req.GetVolumeCapability().GetBlock() != nil {
		return nil, status.Error(codes.InvalidArgument, "cannot have block access type")
	}

	if req.GetVolumeCapability().GetMount() == nil {
		return nil, status.Error(codes.InvalidArgument, "expecting to have mount access type")
	}

	volCtx := req.GetVolumeContext()
	if volCtx == nil {
		return nil, status.Error(codes.InvalidArgument, "Publish context missing in request")
	}

	podName := volCtx[PodNameContextKey]
	if podName == "" {
		return nil, status.Error(codes.InvalidArgument, "No Pod Name included with request")
	}

	mode := volCtx[CSIVolumeAttributeModeField]
	if mode == "" {
		return nil, status.Error(codes.InvalidArgument, "No mode attribute included with request")
	}

	spanContext := getSpanContext(volCtx)

	dynakubeName := volCtx[CSIVolumeAttributeDynakubeField]
	if dynakubeName == "" {
		return nil, status.Error(codes.InvalidArgument, "No dynakube attribute included with request")
	}

	volumeConfig := &VolumeConfig{
		VolumeInfo: VolumeInfo{
			VolumeID:   volID,
			TargetPath: targetPath,
		},
		PodName:         podName,
		Mode:            mode,
		DynakubeName:    dynakubeName,
		OtelSpanContext: spanContext,
	}

	return volumeConfig, nil
}

func getSpanContext(volCtx map[string]string) *trace.SpanContext {
	var spanContext *trace.SpanContext

	if spanID, ok := volCtx[CSIOtelSpanId]; ok {
		if traceID, ok := volCtx[CSIOtelTraceId]; ok {
			sid, spanErr := trace.SpanIDFromHex(spanID)
			tid, traceErr := trace.TraceIDFromHex(traceID)

			if spanErr == nil && traceErr == nil {
				spanContextConfig := trace.SpanContextConfig{
					TraceID: tid,
					SpanID:  sid,
				}
				newSpanContext := trace.NewSpanContext(spanContextConfig)
				spanContext = &newSpanContext
			}
		}
	}

	return spanContext
}

// Transforms the NodeUnpublishVolumeRequest into a VolumeInfo
func ParseNodeUnpublishVolumeRequest(req *csi.NodeUnpublishVolumeRequest) (*VolumeInfo, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	return &VolumeInfo{VolumeID: volumeID, TargetPath: targetPath}, nil
}
