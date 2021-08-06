package mount

import (
	"context"
	"os"

	"k8s.io/mount-utils"
	utilsexec "k8s.io/utils/exec"
	exec "k8s.io/utils/exec/testing"
)

type fakeMounter struct {
	mount.SafeFormatAndMount
	utilsexec.Interface
}

func (m *fakeMounter) GetStatistics(volumePath string) (volumeStatistics, error) {
	return volumeStatistics{}, nil
}

func (m *fakeMounter) IsBlockDevice(devicePath string) (bool, error) {
	return true, nil
}

// NewFake creates an fake implementation of the
// mount.Interface, to be used in tests.
func NewFake() Interface {
	return &fakeMounter{
		mount.SafeFormatAndMount{
			Interface: mount.NewFakeMounter([]mount.MountPoint{}),
			Exec:      &exec.FakeExec{DisableScripts: true},
		},
		utilsexec.New(),
	}
}

func (m *fakeMounter) GetDevicePath(ctx context.Context, volumeID string, hypervisor string) (string, error) {
	return "/dev/sdb", nil
}

func (m *fakeMounter) GetDeviceName(mountPath string) (string, int, error) {
	return mount.GetDeviceNameFromMount(m, mountPath)
}

func (*fakeMounter) ExistsPath(filename string) (bool, error) {
	return true, nil
}

func (*fakeMounter) MakeDir(pathname string) error {
	err := os.MkdirAll(pathname, os.FileMode(0755))
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func (*fakeMounter) MakeFile(pathname string) error {
	return nil
}

func (m *fakeMounter) CleanScsi(ctx context.Context, deviceID, hypervisor string) {
	//Do nothing
}
