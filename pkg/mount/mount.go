// Package mount provides utilities to detect,
// format and mount storage devices.
package mount

import (
	"context"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"golang.org/x/sys/unix"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/mount-utils"
	"k8s.io/utils/exec"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	diskIDPath = "/dev/disk/by-path"
)

// Interface defines the set of methods to allow for
// mount operations on a system.
type Interface interface {
	mount.Interface
	exec.Interface

	FormatAndMount(source string, target string, fstype string, options []string) error

	CleanScsi(ctx context.Context, deviceID, hypervisor string)

	GetStatistics(volumePath string) (volumeStatistics, error)
	IsBlockDevice(devicePath string) (bool, error)

	GetDevicePath(ctx context.Context, volumeID string, hypervisor string) (string, error)
	GetDeviceName(mountPath string) (string, int, error)
	ExistsPath(filename string) (bool, error)
	MakeDir(pathname string) error
	MakeFile(pathname string) error
}

type mounter struct {
	mount.SafeFormatAndMount
	exec.Interface
}

type volumeStatistics struct {
	AvailableBytes, TotalBytes, UsedBytes    int64
	AvailableInodes, TotalInodes, UsedInodes int64
}

// New creates an implementation of the mount.Interface.
func New() Interface {
	return &mounter{
		mount.SafeFormatAndMount{
			Interface: mount.New(""),
			Exec:      exec.New(),
		},
		exec.New(),
	}
}

func (m *mounter) GetDevicePath(ctx context.Context, deviceID string, hypervisor string) (string, error) {

	deviceID = CorrectDeviceId(ctx, deviceID, hypervisor)

	deviceID = fmt.Sprintf("pci-0000:00:10.0-scsi-0:0:%s:0", deviceID)
	ctxzap.Extract(ctx).Sugar().Debugf("device path: %s/%s", diskIDPath, deviceID)

	backoff := wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   1.1,
		Steps:    15,
	}

	var devicePath string
	err := wait.ExponentialBackoffWithContext(ctx, backoff, func() (bool, error) {
		path, err := m.getDevicePathBySerialID(deviceID)
		if err != nil {
			return false, err
		}
		if path != "" {
			devicePath = path
			ctxzap.Extract(ctx).Sugar().Debugf("device path found: %s", path)
			return true, nil
		}
		m.rescanScsi(ctx)
		return false, nil
	})

	if err == wait.ErrWaitTimeout {
		return "", fmt.Errorf("Failed to find device for the deviceID: %q within the alloted time", deviceID)
	} else if devicePath == "" {
		return "", fmt.Errorf("Device path was empty for deviceID: %q", deviceID)
	}
	return devicePath, nil
}

//Corrects the device id on the node. the scsi id may not match th id which is set from te cloudstack controller
//1. ClousStack assumes that SCSI ID 3 is always the CD-ROM and is ignoring this id.
//https://github.com/apache/cloudstack/blob/98d42750cc21dfce5a8dd6d1880e09a621e0152e/server/src/main/java/com/cloud/storage/VolumeApiServiceImpl.java#L3442
//2. SCSI ID 7 is reserved for the Virtual SCSI Controller
//https://docs.vmware.com/en/VMware-vSphere/6.0/com.vmware.vsphere.hostclient.doc/GUID-5872D173-A076-42FE-8D0B-9DB0EB0E7362_copy.html
func CorrectDeviceId(ctx context.Context, deviceID, hypervisor string) string {
	ctxzap.Extract(ctx).Sugar().Debugf("device id: '%s' (Hypervisor: %s)", deviceID, hypervisor)

	if strings.ToLower(hypervisor) == "vmware" {
		ctxzap.Extract(ctx).Sugar().Warnf("volume hypervisor is VMWare, try to correct SCSI ID between ID 3-7")
		idInt, _ := strconv.Atoi(deviceID)
		if idInt > 3 && idInt <= 7 {
			idInt--
			deviceID = fmt.Sprintf("%d", idInt)
			ctxzap.Extract(ctx).Sugar().Warnf("new device id: %s", deviceID)
		}
	}

	return deviceID
}

func (m *mounter) getDevicePathBySerialID(volumeID string) (string, error) {
	source := filepath.Join(diskIDPath, volumeID)
	_, err := os.Stat(source)
	if err == nil {
		return source, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	return "", nil
}

func (m *mounter) rescanScsi(ctx context.Context) {
	log := ctxzap.Extract(ctx).Sugar()
	log.Debug("Scaning SCSI host...")

	args := []string{}
	cmd := m.Exec.Command("rescan-scsi-bus.sh", args...)
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running rescan-scsi-bus.sh: %v\n", err)
	}
}

func (m *mounter) CleanScsi(ctx context.Context, deviceID, hypervisor string) {
	log := ctxzap.Extract(ctx).Sugar()

	deviceID = CorrectDeviceId(ctx, deviceID, hypervisor)

	devicePath := fmt.Sprintf("/sys/class/scsi_device/0:0:%s:0/device/delete", deviceID)
	log.Debugf("removing SCSI devices on %s", devicePath)
	args := []string{deviceID}
	cmd := m.Exec.Command("clean-scsi-bus.sh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running echo 1 > %s: %v\n", devicePath, err)
	}

	fmt.Println(string(out))
}

func (m *mounter) GetDeviceName(mountPath string) (string, int, error) {
	return mount.GetDeviceNameFromMount(m, mountPath)
}

// diskUUIDToSerial reproduces CloudStack function diskUuidToSerial
// from https://github.com/apache/cloudstack/blob/0f3f2a0937/plugins/hypervisors/kvm/src/main/java/com/cloud/hypervisor/kvm/resource/LibvirtComputingResource.java#L3000
//
// This is what CloudStack do *with KVM hypervisor* to translate
// a CloudStack volume UUID to libvirt disk serial.
func diskUUIDToSerial(uuid string) string {
	uuidWithoutHyphen := strings.ReplaceAll(uuid, "-", "")
	if len(uuidWithoutHyphen) < 20 {
		return uuidWithoutHyphen
	}
	return uuidWithoutHyphen[:20]
}

func (*mounter) ExistsPath(filename string) (bool, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (*mounter) MakeDir(pathname string) error {
	err := os.MkdirAll(pathname, os.FileMode(0755))
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func (*mounter) MakeFile(pathname string) error {
	f, err := os.OpenFile(pathname, os.O_CREATE, os.FileMode(0644))
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}

//Copy Pasta from https://github.com/digitalocean/csi-digitalocean/blob/db266f4044178a96c5aa9e2420efae8723af75f4/driver/mounter.go
func (m *mounter) GetStatistics(volumePath string) (volumeStatistics, error) {
	isBlock, err := m.IsBlockDevice(volumePath)
	if err != nil {
		return volumeStatistics{}, fmt.Errorf("failed to determine if volume %s is block device: %v", volumePath, err)
	}

	if isBlock {
		// See http://man7.org/linux/man-pages/man8/blockdev.8.html for details
		output, err := m.Exec.Command("blockdev", "getsize64", volumePath).CombinedOutput()
		if err != nil {
			return volumeStatistics{}, fmt.Errorf("error when getting size of block volume at path %s: output: %s, err: %v", volumePath, string(output), err)
		}
		strOut := strings.TrimSpace(string(output))
		gotSizeBytes, err := strconv.ParseInt(strOut, 10, 64)
		if err != nil {
			return volumeStatistics{}, fmt.Errorf("failed to parse size %s into int", strOut)
		}

		return volumeStatistics{
			TotalBytes: gotSizeBytes,
		}, nil
	}

	var statfs unix.Statfs_t
	// See http://man7.org/linux/man-pages/man2/statfs.2.html for details.
	err = unix.Statfs(volumePath, &statfs)
	if err != nil {
		return volumeStatistics{}, err
	}

	volStats := volumeStatistics{
		AvailableBytes: int64(statfs.Bavail) * int64(statfs.Bsize),
		TotalBytes:     int64(statfs.Blocks) * int64(statfs.Bsize),
		UsedBytes:      (int64(statfs.Blocks) - int64(statfs.Bfree)) * int64(statfs.Bsize),

		AvailableInodes: int64(statfs.Ffree),
		TotalInodes:     int64(statfs.Files),
		UsedInodes:      int64(statfs.Files) - int64(statfs.Ffree),
	}

	return volStats, nil
}

func (m *mounter) IsBlockDevice(devicePath string) (bool, error) {
	var stat unix.Stat_t
	err := unix.Stat(devicePath, &stat)
	if err != nil {
		return false, err
	}

	return (stat.Mode & unix.S_IFMT) == unix.S_IFBLK, nil
}
