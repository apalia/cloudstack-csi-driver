// Package driver provides the implementation of the CSI plugin.
//
// It contains the gRPC server implementation of CSI specification.
package driver

import (
	"go.uber.org/zap"

	"github.com/leaseweb/cloudstack-csi-driver/pkg/cloud"
	"github.com/leaseweb/cloudstack-csi-driver/pkg/mount"
)

// Interface is the CloudStack CSI driver interface.
type Interface interface {
	// Run the CSI driver gRPC server
	Run() error
}

type cloudstackDriver struct {
	endpoint string
	nodeName string
	version  string

	connector cloud.Interface
	mounter   mount.Interface
	logger    *zap.Logger
}

// New instantiates a new CloudStack CSI driver
func New(endpoint string, csConnector cloud.Interface, mounter mount.Interface, nodeName string, version string, logger *zap.Logger) (Interface, error) {
	return &cloudstackDriver{
		endpoint:  endpoint,
		nodeName:  nodeName,
		version:   version,
		connector: csConnector,
		mounter:   mounter,
		logger:    logger,
	}, nil
}

func (cs *cloudstackDriver) Run() error {
	ids := NewIdentityServer(cs.version)
	ctrls := NewControllerServer(cs.connector)
	ns := NewNodeServer(cs.connector, cs.mounter, cs.nodeName)

	return cs.serve(ids, ctrls, ns)
}
