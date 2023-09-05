package cloud

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

func (c *client) GetVMByID(ctx context.Context, vmID string) (*VM, error) {
	p := c.VirtualMachine.NewListVirtualMachinesParams()
	p.SetId(vmID)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "ListVirtualMachines", "params", map[string]string{
		"id": vmID,
	})
	
	//get projectid from metadata
	projectID := c.metadataProjectID(ctx)

	if projectID != "" {
		p.SetProjectid(projectID)
	}
	
	l, err := c.VirtualMachine.ListVirtualMachines(p)
	if err != nil {
		return nil, err
	}
	if l.Count == 0 {
		return nil, ErrNotFound
	}
	if l.Count > 1 {
		return nil, ErrTooManyResults
	}
	vm := l.VirtualMachines[0]
	return &VM{
		ID:     vm.Id,
		ZoneID: vm.Zoneid,
	}, nil
}

func (c *client) getVMByName(ctx context.Context, name string) (*VM, error) {
	p := c.VirtualMachine.NewListVirtualMachinesParams()
	p.SetName(name)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "ListVirtualMachines", "params", map[string]string{
		"name": name,
	})
	
	//get projectid from metadata
	projectID := c.metadataProjectID(ctx)

	if projectID != "" {
		p.SetProjectid(projectID)
	}
	
	l, err := c.VirtualMachine.ListVirtualMachines(p)
	if err != nil {
		return nil, err
	}
	if l.Count == 0 {
		return nil, ErrNotFound
	}
	if l.Count > 1 {
		return nil, ErrTooManyResults
	}
	vm := l.VirtualMachines[0]
	return &VM{
		ID:     vm.Id,
		ZoneID: vm.Zoneid,
	}, nil
}
