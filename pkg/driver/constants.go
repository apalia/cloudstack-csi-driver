package driver

// DriverName is the name of the CSI plugin
const DriverName = "csi.cloudstack.apache.org"

// Topology keys
const (
	ZoneKey = "topology." + DriverName + "/zone"
	HostKey = "topology." + DriverName + "/host"
)

// Volume parameters keys
const (
	DiskOfferingKey = DriverName + "/disk-offering-id"
)

const deviceIDContextKey = "deviceID"
