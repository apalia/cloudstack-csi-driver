package driver

import (
	"errors"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

// Topology represents CloudStack storage topology.
type Topology struct {
	ZoneID string
	HostID string
}

// NewTopology converts a *csi.Topology to Topology.
func NewTopology(t *csi.Topology) (Topology, error) {
	segments := t.GetSegments()
	if segments == nil {
		return Topology{}, errors.New("Nil segment in topology")
	}

	zoneID, ok := segments[ZoneKey]
	if !ok {
		return Topology{}, errors.New("No zone in topology")
	}
	hostID := segments[HostKey]
	return Topology{zoneID, hostID}, nil
}

// ToCSI converts a Topology to a *csi.Topology.
func (t Topology) ToCSI() *csi.Topology {
	segments := make(map[string]string)
	segments[ZoneKey] = t.ZoneID
	if t.HostID != "" {
		segments[ZoneKey] = t.ZoneID
	}
	return &csi.Topology{
		Segments: segments,
	}
}
