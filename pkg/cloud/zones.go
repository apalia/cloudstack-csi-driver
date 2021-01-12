package cloud

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

func (c *client) ListZonesID(ctx context.Context) ([]string, error) {
	result := []string{}
	p := c.Zone.NewListZonesParams()
	p.SetAvailable(true)
	ctxzap.Extract(ctx).Sugar().Infow("CloudStack API call", "command", "ListZones", "params", map[string]string{
		"available": "true",
	})
	r, err := c.Zone.ListZones(p)
	if err != nil {
		return result, err
	}
	for _, zone := range r.Zones {
		result = append(result, zone.Id)
	}
	return result, nil
}
