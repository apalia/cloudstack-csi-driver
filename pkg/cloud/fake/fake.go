// Package fake provides a fake implementation of the cloud
// connector interface, to be used in tests.
package fake

import (
	"context"

	"github.com/hashicorp/go-uuid"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/util"
)

const zoneID = "a1887604-237c-4212-a9cd-94620b7880fa"

type fakeConnector struct {
	node          *cloud.VM
	volumesByID   map[string]cloud.Volume
	volumesByName map[string]cloud.Volume
}

// New returns a new fake implementation of the
// CloudStack connector.
func New() cloud.Interface {
	volume := cloud.Volume{
		ID:               "ace9f28b-3081-40c1-8353-4cc3e3014072",
		Name:             "vol-1",
		Size:             10,
		DiskOfferingID:   "9743fd77-0f5d-4ef9-b2f8-f194235c769c",
		ZoneID:           zoneID,
		VirtualMachineID: "",
		DeviceID:         "",
	}
	node := &cloud.VM{
		ID:     "0d7107a3-94d2-44e7-89b8-8930881309a5",
		ZoneID: zoneID,
	}
	return &fakeConnector{
		node:          node,
		volumesByID:   map[string]cloud.Volume{volume.ID: volume},
		volumesByName: map[string]cloud.Volume{volume.Name: volume},
	}
}

func (f *fakeConnector) GetVMByID(ctx context.Context, vmID string) (*cloud.VM, error) {
	if vmID == f.node.ID {
		return f.node, nil
	}
	return nil, cloud.ErrNotFound
}

func (f *fakeConnector) GetNodeInfo(ctx context.Context, vmName string) (*cloud.VM, error) {
	return f.node, nil
}

func (f *fakeConnector) ListZonesID(ctx context.Context) ([]string, error) {
	return []string{zoneID}, nil
}

func (f *fakeConnector) GetVolumeByID(ctx context.Context, volumeID string) (*cloud.Volume, error) {
	vol, ok := f.volumesByID[volumeID]
	if ok {
		return &vol, nil
	}
	return nil, cloud.ErrNotFound
}

func (f *fakeConnector) GetVolumeByName(ctx context.Context, name string) (*cloud.Volume, error) {
	vol, ok := f.volumesByName[name]
	if ok {
		return &vol, nil
	}
	return nil, cloud.ErrNotFound
}

func (f *fakeConnector) CreateVolume(ctx context.Context, diskOfferingID, zoneID, name string, sizeInGB int64) (string, error) {
	id, _ := uuid.GenerateUUID()
	vol := cloud.Volume{
		ID:             id,
		Name:           name,
		Size:           util.GigaBytesToBytes(sizeInGB),
		DiskOfferingID: diskOfferingID,
		ZoneID:         zoneID,
	}
	f.volumesByID[vol.ID] = vol
	f.volumesByName[vol.Name] = vol
	return vol.ID, nil
}

func (f *fakeConnector) DeleteVolume(ctx context.Context, id string) error {
	if vol, ok := f.volumesByID[id]; ok {
		name := vol.Name
		delete(f.volumesByName, name)
	}
	delete(f.volumesByID, id)
	return nil
}

func (f *fakeConnector) AttachVolume(ctx context.Context, volumeID, vmID string) (string, error) {
	return "1", nil
}

func (f *fakeConnector) DetachVolume(ctx context.Context, volumeID string) error {
	return nil
}
