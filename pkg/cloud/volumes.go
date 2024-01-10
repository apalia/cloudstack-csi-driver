package cloud

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/apalia/cloudstack-csi-driver/pkg/util"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

func (c *client) GetVolumeByID(ctx context.Context, volumeID string) (*Volume, error) {
	p := c.Volume.NewListVolumesParams()
	p.SetId(volumeID)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "ListVolumes", "params", map[string]string{
		"id": volumeID,
	})
	l, err := c.Volume.ListVolumes(p)
	if err != nil {
		return nil, err
	}
	if l.Count == 0 {
		return nil, ErrNotFound
	}
	if l.Count > 1 {
		return nil, ErrTooManyResults
	}
	vol := l.Volumes[0]
	v := Volume{
		ID:               vol.Id,
		Name:             vol.Name,
		Size:             vol.Size,
		DiskOfferingID:   vol.Diskofferingid,
		ZoneID:           vol.Zoneid,
		VirtualMachineID: vol.Virtualmachineid,
		DeviceID:         strconv.FormatInt(vol.Deviceid, 10),
	}
	return &v, nil
}

func (c *client) GetVolumeByName(ctx context.Context, name string) (*Volume, error) {
	p := c.Volume.NewListVolumesParams()
	p.SetName(name)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "ListVolumes", "params", map[string]string{
		"name": name,
	})
	l, err := c.Volume.ListVolumes(p)
	if err != nil {
		return nil, err
	}
	if l.Count == 0 {
		return nil, ErrNotFound
	}
	if l.Count > 1 {
		return nil, ErrTooManyResults
	}
	vol := l.Volumes[0]
	v := Volume{
		ID:               vol.Id,
		Name:             vol.Name,
		Size:             vol.Size,
		DiskOfferingID:   vol.Diskofferingid,
		ZoneID:           vol.Zoneid,
		VirtualMachineID: vol.Virtualmachineid,
		DeviceID:         strconv.FormatInt(vol.Deviceid, 10),
	}
	return &v, nil
}

func (c *client) CreateVolume(ctx context.Context, diskOfferingID, zoneID, name string, sizeInGB int64) (string, error) {
	p := c.Volume.NewCreateVolumeParams()
	p.SetDiskofferingid(diskOfferingID)
	p.SetZoneid(zoneID)
	p.SetName(name)
	p.SetSize(sizeInGB)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "CreateVolume", "params", map[string]string{
		"diskofferingid": diskOfferingID,
		"zoneid":         zoneID,
		"name":           name,
		"size":           strconv.FormatInt(sizeInGB, 10),
	})
	vol, err := c.Volume.CreateVolume(p)
	if err != nil {
		return "", err
	}
	return vol.Id, nil
}

func (c *client) DeleteVolume(ctx context.Context, id string) error {
	p := c.Volume.NewDeleteVolumeParams(id)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "DeleteVolume", "params", map[string]string{
		"id": id,
	})
	_, err := c.Volume.DeleteVolume(p)
	if err != nil && strings.Contains(err.Error(), "4350") {
		// CloudStack error InvalidParameterValueException
		return ErrNotFound
	}
	return err
}

func (c *client) AttachVolume(ctx context.Context, volumeID, vmID string) (string, error) {
	p := c.Volume.NewAttachVolumeParams(volumeID, vmID)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "AttachVolume", "params", map[string]string{
		"id":               volumeID,
		"virtualmachineid": vmID,
	})
	r, err := c.Volume.AttachVolume(p)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(r.Deviceid, 10), nil
}

func (c *client) DetachVolume(ctx context.Context, volumeID string) error {
	p := c.Volume.NewDetachVolumeParams()
	p.SetId(volumeID)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "DetachVolume", "params", map[string]string{
		"id": volumeID,
	})
	_, err := c.Volume.DetachVolume(p)
	return err
}

// ExpandVolume expands the volume to new size
func (c *client) ExpandVolume(ctx context.Context, volumeID string, newSizeInGB int64) error {
	volume, _, err := c.Volume.GetVolumeByID(volumeID)
	if err != nil {
		return fmt.Errorf("failed to retrieve volume '%s': %v", volumeID, err)
	}
	if volume.State != "Allocated" && volume.State != "Ready" {
		return fmt.Errorf("volume '%s' is not in 'Allocated' or 'Ready' state to get resized", volumeID)
	}
	currentSize := volume.Size
	currentSizeInGB := util.RoundUpBytesToGB(currentSize)
	volumeName := volume.Name
	p := c.Volume.NewResizeVolumeParams(volumeID)
	p.SetId(volumeID)
	p.SetSize(newSizeInGB)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "ExpandVolume", "params", map[string]string{
		"name":           volumeName,
		"volumeid":       volumeID,
		"current_size":   strconv.FormatInt(currentSizeInGB, 10),
		"requested_size": strconv.FormatInt(newSizeInGB, 10),
	})
	// Execute the API call to resize the volume
	expandedVol, err := c.Volume.ResizeVolume(p)
	if err != nil {
		// Handle the error accordingly
		return fmt.Errorf("failed to expand volume '%s': %v", volumeID, err)
	}
	fmt.Printf("Volume %s resied to %d successfully", expandedVol.Id, expandedVol.Size)
	return nil
}
