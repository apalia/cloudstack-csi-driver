package cloud

import (
	"context"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

func (c *client) GetDomainID(ctx context.Context) (string, error) {
	p, _, err := c.Project.GetProjectByID(c.ProjectID)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "GetProjectByID", "params", map[string]string{
		"projectID": c.ProjectID,
	})
	return p.Domainid, err
}

func (c *client) GetProjectID() string {
	return c.ProjectID
}
