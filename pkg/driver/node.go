package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apalia/cloudstack-csi-driver/pkg/cloud"
	"github.com/apalia/cloudstack-csi-driver/pkg/mount"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// default file system type to be used when it is not provided
	defaultFsType                 = "ext4"
	maxAllowedBlockVolumesPerNode = 10
)

type nodeServer struct {
	csi.UnimplementedNodeServer
	connector cloud.Interface
	mounter   mount.Interface
	nodeName  string
}

// NewNodeServer creates a new Node gRPC server.
func NewNodeServer(connector cloud.Interface, mounter mount.Interface, nodeName string) csi.NodeServer {
	if mounter == nil {
		mounter = mount.New()
	}
	return &nodeServer{
		connector: connector,
		mounter:   mounter,
		nodeName:  nodeName,
	}
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {

	// Check parameters

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	target := req.GetStagingTargetPath()
	if target == "" {
		return nil, status.Error(codes.InvalidArgument, "Staging target not provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}
	if !isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	ctxzap.Extract(ctx).Sugar().Infow("#### Mount 1  target: " + target)
	// Now, find the device path

	v, _ := ns.connector.GetVolumeByID(ctx, volumeID)

	deviceID := req.PublishContext[deviceIDContextKey]

	devicePath, err := ns.mounter.GetDevicePath(ctx, v.DeviceID, v.Hypervisor)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Cannot find device path for volume %s: %s", volumeID, err.Error())
	}

	ctxzap.Extract(ctx).Sugar().Infow("Device found",
		"devicePath", devicePath,
		"deviceID", deviceID,
	)

	// If the access type is block, do nothing for stage
	switch volCap.GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		return &csi.NodeStageVolumeResponse{}, nil
	}

	// The access type should now be "Mount".
	// We have to format the partition.

	mnt := volCap.GetMount()
	if mnt == nil {
		return nil, status.Error(codes.InvalidArgument, "Neither block nor mount volume capability")
	}

	// Verify whether mounted
	notMnt, err := ns.mounter.IsLikelyNotMountPoint(target)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	fsType := mnt.GetFsType()
	if fsType == "" {
		fsType = defaultFsType
	}

	var mountOptions []string
	for _, f := range mnt.GetMountFlags() {
		if !hasMountOption(mountOptions, f) {
			mountOptions = append(mountOptions, f)
		}
	}

	// Volume Mount
	if notMnt {
		err = ns.mounter.FormatAndMount(devicePath, target, fsType, mountOptions)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	ctxzap.Extract(ctx).Sugar().Debugf("Staged volume device %s on %s on target %s successfully", volumeID, devicePath, target)
	return &csi.NodeStageVolumeResponse{}, nil
}

// hasMountOption returns a boolean indicating whether the given
// slice already contains a mount option. This is used to prevent
// passing duplicate option to the mount command.
func hasMountOption(options []string, opt string) bool {
	for _, o := range options {
		if o == opt {
			return true
		}
	}
	return false
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	// Check parameters

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	target := req.GetStagingTargetPath()
	if target == "" {
		return nil, status.Error(codes.InvalidArgument, "Staging target not provided")
	}

	// Check if target directory is a mount point. GetDeviceNameFromMount
	// given a mnt point, finds the device from /proc/mounts
	// returns the device name, reference count, and error code
	dev, refCount, err := ns.mounter.GetDeviceName(target)
	if err != nil {
		msg := fmt.Sprintf("failed to check if volume is mounted: %v", err)
		return nil, status.Error(codes.Internal, msg)
	}

	// From the spec: If the volume corresponding to the volume_id
	// is not staged to the staging_target_path, the Plugin MUST
	// reply 0 OK.
	if refCount == 0 {
		return &csi.NodeUnstageVolumeResponse{}, nil
	}

	if refCount > 1 {
		ctxzap.Extract(ctx).Sugar().Warnf("NodeUnstageVolume: found %d references to device %s mounted at target path %s", refCount, dev, target)
	}

	err = ns.mounter.Unmount(target)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmount target %q: %v", target, err)
	}

	ctxzap.Extract(ctx).Sugar().Debugf("NodeUnstageVolume: unmounted %d on target %d", dev, target)

	ns.mounter.CleanScsi(ctx)

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	// Check arguments
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability missing in request")
	}
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	volumeID := req.GetVolumeId()

	if req.GetTargetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}
	targetPath := req.GetTargetPath()

	if req.GetVolumeCapability().GetBlock() != nil &&
		req.GetVolumeCapability().GetMount() != nil {
		return nil, status.Error(codes.InvalidArgument, "Cannot have both block and mount access type")
	}
	if req.GetStagingTargetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "Staging target path missing in request")
	}
	v, err := ns.connector.GetVolumeByID(ctx, volumeID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "No volume found")
	}

	readOnly := req.GetReadonly()
	options := []string{"bind"}
	if readOnly {
		options = append(options, "ro")
	}

	deviceID := ""
	if req.GetPublishContext() != nil {
		deviceID = req.GetPublishContext()[deviceIDContextKey]
	}

	if req.GetVolumeCapability().GetMount() != nil {
		source := req.GetStagingTargetPath()

		notMnt, err := ns.mounter.IsLikelyNotMountPoint(targetPath)
		if err != nil {
			if os.IsNotExist(err) {
				if err := ns.mounter.MakeDir(targetPath); err != nil {
					return nil, status.Errorf(codes.Internal, "Could not create dir %q: %v", targetPath, err)
				}
			} else {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}
		if !notMnt {
			return &csi.NodePublishVolumeResponse{}, nil
		}

		fsType := req.GetVolumeCapability().GetMount().GetFsType()

		mountFlags := req.GetVolumeCapability().GetMount().GetMountFlags()

		ctxzap.Extract(ctx).Sugar().Infow("Mounting device",
			"targetPath", targetPath,
			"fsType", fsType,
			"deviceID", deviceID,
			"readOnly", readOnly,
			"volumeID", volumeID,
			"mountFlags", mountFlags,
		)

		if err := ns.mounter.Mount(source, targetPath, fsType, options); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to mount %s at %s: %s", source, targetPath, err.Error())
		}
		ctxzap.Extract(ctx).Sugar().Debugf("mount volume %s from source %s on target %s ", volumeID, source, targetPath)
	}

	if req.GetVolumeCapability().GetBlock() != nil {
		volumeID := req.GetVolumeId()

		devicePath, err := ns.mounter.GetDevicePath(ctx, v.DeviceID, v.Hypervisor)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Cannot find device path for volume %s: %s", volumeID, err.Error())
		}

		globalMountPath := filepath.Dir(targetPath)
		exists, err := ns.mounter.ExistsPath(globalMountPath)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not check if path exists %q: %v", globalMountPath, err)
		}
		if !exists {
			if err = ns.mounter.MakeDir(globalMountPath); err != nil {
				return nil, status.Errorf(codes.Internal, "Could not create dir %q: %v", globalMountPath, err)
			}
		}

		err = ns.mounter.MakeFile(targetPath)
		if err != nil {
			if removeErr := os.Remove(targetPath); removeErr != nil {
				return nil, status.Errorf(codes.Internal, "Could not remove mount target %q: %v", targetPath, removeErr)
			}
			return nil, status.Errorf(codes.Internal, "Could not create file %q: %v", targetPath, err)
		}

		if err := ns.mounter.Mount(devicePath, targetPath, "", options); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to mount %s at %s: %s", devicePath, targetPath, err.Error())
		}
		ctxzap.Extract(ctx).Sugar().Infow("### mount volume on devicePath: " + devicePath + " and targetPath: " + targetPath)
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {

	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if req.GetTargetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}
	targetPath := req.GetTargetPath()

	volumeID := req.GetVolumeId()
	if _, err := ns.connector.GetVolumeByID(ctx, volumeID); err == cloud.ErrNotFound {
		return nil, status.Errorf(codes.NotFound, "Volume %v not found", volumeID)
	} else if err != nil {
		// Error with CloudStack
		return nil, status.Errorf(codes.Internal, "Error %v", err)
	}

	ctxzap.Extract(ctx).Sugar().Debugw("node unpublish (call unmount) volume", "id", volumeID, "targetPath", targetPath)
	err := ns.mounter.Unmount(targetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unmount of targetpath %s failed with error %v", targetPath, err)
	}
	err = os.Remove(targetPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, status.Errorf(codes.Internal, "Deleting %s failed with error %v", targetPath, err)
	}
	ctxzap.Extract(ctx).Sugar().Debugf("NodeUnpublishVolume: successfully unpublish volume %s on node %s", volumeID, targetPath)

	ns.mounter.CleanScsi(ctx)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	if ns.nodeName == "" {
		return nil, status.Error(codes.Internal, "Missing node name")
	}

	vm, err := ns.connector.GetNodeInfo(ctx, ns.nodeName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if vm.ID == "" {
		return nil, status.Error(codes.Internal, "Node with no ID")
	}
	if vm.ZoneID == "" {
		return nil, status.Error(codes.Internal, "Node zone ID not found")
	}

	topology := Topology{ZoneID: vm.ZoneID}
	return &csi.NodeGetInfoResponse{
		NodeId:             vm.ID,
		AccessibleTopology: topology.ToCSI(),
		MaxVolumesPerNode:  maxAllowedBlockVolumesPerNode,
	}, nil
}

func (ns *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
					},
				},
			},
		},
	}, nil
}
