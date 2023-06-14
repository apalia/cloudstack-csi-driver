package driver

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/util"
)

// onlyVolumeCapAccessMode is the only volume capability access
// mode possible for CloudStack: SINGLE_NODE_WRITER, since a
// CloudStack volume can only be attached to a single node at
// any given time.
var onlyVolumeCapAccessMode = csi.VolumeCapability_AccessMode{
	Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
}

type controllerServer struct {
	csi.UnimplementedControllerServer
	connector cloud.Interface
	locks     map[string]*sync.Mutex
}

// NewControllerServer creates a new Controller gRPC server.
func NewControllerServer(connector cloud.Interface) csi.ControllerServer {
	return &controllerServer{
		connector: connector,
		locks:     make(map[string]*sync.Mutex),
	}
}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {

	// Check arguments

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume name missing in request")
	}
	name := req.GetName()

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities missing in request")
	}
	if !isValidVolumeCapabilities(volCaps) {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not supported. Only SINGLE_NODE_WRITER supported.")
	}

	if req.GetParameters() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume parameters missing in request")
	}
	diskOfferingID := req.GetParameters()[DiskOfferingKey]
	if diskOfferingID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Missing parameter %v", DiskOfferingKey)
	}

	// Check if a volume with that name already exists
	if vol, err := cs.connector.GetVolumeByName(ctx, name); err == cloud.ErrNotFound {
		// The volume does not exist
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "CloudStack error: %v", err)
	} else {
		// The volume exists. Check if it suits the request.
		if ok, message := checkVolumeSuitable(vol, diskOfferingID, req.GetCapacityRange(), req.GetAccessibilityRequirements()); !ok {
			return nil, status.Errorf(codes.AlreadyExists, "Volume %v already exists but does not satisfy request: %s", name, message)
		}
		// Existing volume is ok
		return &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				VolumeId:      vol.ID,
				CapacityBytes: vol.Size,
				AccessibleTopology: []*csi.Topology{
					Topology{ZoneID: vol.ZoneID}.ToCSI(),
				},
			},
		}, nil
	}

	// We have to create the volume

	// Determine volume size using requested capacity range
	sizeInGB, err := determineSize(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Determine zone using topology constraints
	var zoneID string
	topologyRequirement := req.GetAccessibilityRequirements()
	if topologyRequirement == nil || topologyRequirement.GetRequisite() == nil {
		// No topology requirement. Use random zone
		zones, err := cs.connector.ListZonesID(ctx)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		n := len(zones)
		if n == 0 {
			return nil, status.Error(codes.Internal, "No zone available")
		}
		zoneID = zones[rand.Intn(n)]
	} else {
		reqTopology := topologyRequirement.GetRequisite()
		if len(reqTopology) > 1 {
			return nil, status.Error(codes.InvalidArgument, "Too many topology requirements")
		}
		t, err := NewTopology(reqTopology[0])
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "Cannot parse topology requirements")
		}
		zoneID = t.ZoneID
	}

	volID, err := cs.connector.CreateVolume(ctx, diskOfferingID, zoneID, name, sizeInGB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Cannot create volume %s: %v", name, err.Error())
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volID,
			CapacityBytes: util.GigaBytesToBytes(sizeInGB),
			AccessibleTopology: []*csi.Topology{
				Topology{ZoneID: zoneID}.ToCSI(),
			},
		},
	}, nil
}

func checkVolumeSuitable(vol *cloud.Volume,
	diskOfferingID string, capRange *csi.CapacityRange, topologyRequirement *csi.TopologyRequirement) (bool, string) {

	if vol.DiskOfferingID != diskOfferingID {
		return false, fmt.Sprintf("Disk offering %s; requested disk offering %s", vol.DiskOfferingID, diskOfferingID)
	}

	if capRange != nil {
		if capRange.GetLimitBytes() > 0 && vol.Size > capRange.GetLimitBytes() {
			return false, fmt.Sprintf("Disk size %v bytes > requested limit size %v bytes", vol.Size, capRange.GetLimitBytes())
		}
		if capRange.GetRequiredBytes() > 0 && vol.Size < capRange.GetRequiredBytes() {
			return false, fmt.Sprintf("Disk size %v bytes < requested required size %v bytes", vol.Size, capRange.GetRequiredBytes())
		}
	}

	if topologyRequirement != nil && topologyRequirement.GetRequisite() != nil {
		reqTopology := topologyRequirement.GetRequisite()
		if len(reqTopology) > 1 {
			return false, "Too many topology requirements"
		}
		t, err := NewTopology(reqTopology[0])
		if err != nil {
			return false, "Cannot parse topology requirements"
		}
		if t.ZoneID != vol.ZoneID {
			return false, fmt.Sprintf("Volume in zone %s, requested zone is %s", vol.ZoneID, t.ZoneID)
		}
	}

	return true, ""
}

func determineSize(req *csi.CreateVolumeRequest) (int64, error) {
	var sizeInGB int64

	if req.GetCapacityRange() != nil {
		capRange := req.GetCapacityRange()

		required := capRange.GetRequiredBytes()
		sizeInGB = util.RoundUpBytesToGB(required)
		if sizeInGB == 0 {
			sizeInGB = 1
		}

		if limit := capRange.GetLimitBytes(); limit > 0 {
			if util.GigaBytesToBytes(sizeInGB) > limit {
				return 0, fmt.Errorf("after round-up, volume size %v GB exceeds the limit specified of %v bytes", sizeInGB, limit)
			}
		}
	}

	if sizeInGB == 0 {
		sizeInGB = 1
	}
	return sizeInGB, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	volumeID := req.GetVolumeId()
	err := cs.connector.DeleteVolume(ctx, volumeID)
	if err != nil && err != cloud.ErrNotFound {
		return nil, status.Errorf(codes.Internal, "Cannot delete volume %s: %s", volumeID, err.Error())
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	// Check arguments

	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	volumeID := req.GetVolumeId()

	if req.GetNodeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Node ID missing in request")
	}
	nodeID := req.GetNodeId()

	//Ensure only one node is processing at same time
	lock, ok := cs.locks[nodeID]
	if !ok {
		lock = &sync.Mutex{}
		cs.locks[nodeID] = lock
	}
	lock.Lock()
	defer lock.Unlock()

	if req.GetReadonly() {
		return nil, status.Error(codes.InvalidArgument, "Readonly not possible")
	}

	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability missing in request")
	}
	if req.GetVolumeCapability().AccessMode.Mode != onlyVolumeCapAccessMode.GetMode() {
		return nil, status.Error(codes.InvalidArgument, "Access mode not accepted")
	}

	// Check volume
	vol, err := cs.connector.GetVolumeByID(ctx, volumeID)
	if err == cloud.ErrNotFound {
		return nil, status.Errorf(codes.NotFound, "Volume %v not found", volumeID)
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "Error %v", err)
	}

	if vol.VirtualMachineID != "" && vol.VirtualMachineID != nodeID {
		return nil, status.Error(codes.AlreadyExists, "Volume already assigned to another node")
	}

	if _, err := cs.connector.GetVMByID(ctx, nodeID); err == cloud.ErrNotFound {
		return nil, status.Errorf(codes.NotFound, "VM %v not found", nodeID)
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "Error %v", err)
	}

	if vol.VirtualMachineID == nodeID {
		// volume already attached

		publishContext := map[string]string{
			deviceIDContextKey: vol.DeviceID,
		}
		return &csi.ControllerPublishVolumeResponse{PublishContext: publishContext}, nil
	}

	deviceID, err := cs.connector.AttachVolume(ctx, volumeID, nodeID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Cannot attach volume %s: %s", volumeID, err.Error())
	}

	publishContext := map[string]string{
		deviceIDContextKey: deviceID,
	}
	return &csi.ControllerPublishVolumeResponse{PublishContext: publishContext}, nil
}

func (cs *controllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	// Check arguments

	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	volumeID := req.GetVolumeId()
	nodeID := req.GetNodeId()

	// Check volume
	if vol, err := cs.connector.GetVolumeByID(ctx, volumeID); err == cloud.ErrNotFound {
		// Volume does not exist in CloudStack. We can safely assume this volume is no longer attached
		// The spec requires us to return OK here
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "Error %v", err)
	} else if nodeID != "" && vol.VirtualMachineID != nodeID {
		// Volume is present but not attached to this particular nodeID
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	}

	// Check VM existence
	if _, err := cs.connector.GetVMByID(ctx, nodeID); err == cloud.ErrNotFound {
		return nil, status.Errorf(codes.NotFound, "VM %v not found", nodeID)
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "Error %v", err)
	}

	err := cs.connector.DetachVolume(ctx, volumeID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Cannot detach volume %s: %s", volumeID, err.Error())
	}

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	if _, err := cs.connector.GetVolumeByID(ctx, volumeID); err == cloud.ErrNotFound {
		return nil, status.Errorf(codes.NotFound, "Volume %v not found", volumeID)
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "Error %v", err)
	}

	var confirmed *csi.ValidateVolumeCapabilitiesResponse_Confirmed
	if isValidVolumeCapabilities(volCaps) {
		confirmed = &csi.ValidateVolumeCapabilitiesResponse_Confirmed{VolumeCapabilities: volCaps}
	}
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: confirmed,
	}, nil
}

func isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) bool {
	for _, c := range volCaps {
		if c.GetAccessMode() != nil && c.GetAccessMode().GetMode() != onlyVolumeCapAccessMode.GetMode() {
			return false
		}
	}
	return true
}

func (cs *controllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
					},
				},
			},
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
					},
				},
			},
		},
	}, nil
}
