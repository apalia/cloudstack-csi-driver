package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

const (
	cloudInitInstanceFilePath = "/run/cloud-init/instance-data.json"
	cloudStackCloudName       = "cloudstack"
)

func (c *client) metadataInstanceID(ctx context.Context) string {
	slog := ctxzap.Extract(ctx).Sugar()

	// Try a NODE_ID environment variable
	if envNodeID := os.Getenv("NODE_ID"); envNodeID != "" {
		slog.Debugf("Found CloudStack VM ID from environment variable NODE_ID: %s", envNodeID)
		return envNodeID
	}

	// Try cloud-init
	slog.Debug("Try with cloud-init")
	if _, err := os.Stat(cloudInitInstanceFilePath); err == nil {
		slog.Debugf("File %s exists", cloudInitInstanceFilePath)
		ciData, err := c.readCloudInit(ctx, cloudInitInstanceFilePath)
		if err != nil {
			slog.Errorf("Cannot read cloud-init instance data: %v", err)
		} else {
			if ciData.V1.InstanceID != "" {
				slog.Debugf("Found CloudStack VM ID from cloud-init: %s", ciData.V1.InstanceID)
				return ciData.V1.InstanceID
			}
		}
		slog.Error("cloud-init instance ID is not provided")
	} else if os.IsNotExist(err) {
		slog.Debugf("File %s does not exist", cloudInitInstanceFilePath)
	} else {
		slog.Errorf("Cannot read %s: %v", cloudInitInstanceFilePath, err)
	}

	slog.Debug("CloudStack VM ID not found in meta-data.")
	return ""
}

func (c *client) metadataProjectID(ctx context.Context) string {
	slog := ctxzap.Extract(ctx).Sugar()

	// Try cloud-init
	slog.Debug("Try with cloud-init")
	if _, err := os.Stat(cloudInitInstanceFilePath); err == nil {
		slog.Debugf("File %s exists", cloudInitInstanceFilePath)
		ciData, err := c.readCloudInit(ctx, cloudInitInstanceFilePath)
		if err != nil {
			slog.Errorf("Cannot read cloud-init instance data: %v", err)
		} else {
			if ciData.Ds.Metadata.ProjectID != "" {
				return ciData.Ds.Metadata.ProjectID
			}
		}
		slog.Error("cloud-init project ID is not provided")
	}

	slog.Debug("CloudStack project ID not found in meta-data.")
	return ""
}

type cloudInitInstanceData struct {
	V1 cloudInitV1 `json:"v1"`
	Ds cloudInitDs `json:"ds"`
}

type cloudInitV1 struct {
	CloudName  string `json:"cloud_name"`
	InstanceID string `json:"instance_id"`
	Zone       string `json:"availability_zone"`
}

type cloudInitDs struct {
	Metadata cloudInitMetadata `json:"meta_data"`
}

type cloudInitMetadata struct {
	ProjectID string `json:"project-uuid"`
}

func (c *client) readCloudInit(ctx context.Context, instanceFilePath string) (*cloudInitInstanceData, error) {
	slog := ctxzap.Extract(ctx).Sugar()

	b, err := ioutil.ReadFile(instanceFilePath)
	if err != nil {
		slog.Errorf("Cannot read %s", instanceFilePath)
		return nil, err
	}

	var data cloudInitInstanceData
	if err := json.Unmarshal(b, &data); err != nil {
		slog.Errorf("Cannot parse JSON file %s", instanceFilePath)
		return nil, err
	}

	if strings.ToLower(data.V1.CloudName) != cloudStackCloudName {
		slog.Errorf("Cloud-Init cloud name is %s, only %s is supported", data.V1.CloudName, cloudStackCloudName)
		return nil, fmt.Errorf("Cloud-Init cloud name is %s, only %s is supported", data.V1.CloudName, cloudStackCloudName)
	}

	return &data, nil
}
