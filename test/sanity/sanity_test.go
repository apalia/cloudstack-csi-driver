// +build sanity

package sanity

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubernetes-csi/csi-test/v4/pkg/sanity"
	"go.uber.org/zap"

	"github.com/apalia/cloudstack-csi-driver/pkg/cloud/fake"
	"github.com/apalia/cloudstack-csi-driver/pkg/driver"
	"github.com/apalia/cloudstack-csi-driver/pkg/mount"
)

func TestSanity(t *testing.T) {
	// Setup driver
	dir, err := ioutil.TempDir("", "sanity-cloudstack-csi")
	if err != nil {
		t.Fatalf("error creating directory: %v", err)
	}
	defer os.RemoveAll(dir)

	targetPath := filepath.Join(dir, "target")
	stagingPath := filepath.Join(dir, "staging")
	endpoint := "unix://" + filepath.Join(dir, "csi.sock")

	config := sanity.NewTestConfig()
	config.TargetPath = targetPath
	config.StagingPath = stagingPath
	config.Address = endpoint
	config.TestVolumeParameters = map[string]string{
		driver.DiskOfferingKey: "9743fd77-0f5d-4ef9-b2f8-f194235c769c",
	}

	csiDriver, err := driver.New(endpoint, fake.New(), mount.NewFake(), "node", "v0", zap.NewNop())
	if err != nil {
		t.Fatalf("error creating driver: %v", err)
	}
	go func() {
		csiDriver.Run()
	}()

	sanity.Test(t, config)
}
