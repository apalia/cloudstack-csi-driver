package cloud

import "context"

func (c *client) GetNodeInfo(ctx context.Context, vmName string) (*VM, error) {
	// First, try to read the instance ID from meta-data (cloud-init)
	if id := c.metadataInstanceID(ctx); id != "" {
		// Instance ID found using metadata

		// Use CloudStack API to get VM info
		return c.GetVMByID(ctx, id)
	}

	// VM ID was not found using metadata.
	// Use VM name instead
	return c.getVMByName(ctx, vmName)
}
