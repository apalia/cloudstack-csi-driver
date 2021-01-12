package driver

import (
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

func TestDetermineSize(t *testing.T) {
	cases := []struct {
		name          string
		capacityRange *csi.CapacityRange
		expectedSize  int64
		expectError   bool
	}{
		{"no range", nil, 1, false},
		{"only limit", &csi.CapacityRange{LimitBytes: 100 * 1024 * 1024 * 1024}, 1, false},
		{"only limit (too small)", &csi.CapacityRange{LimitBytes: 1024 * 1024}, 0, true},
		{"only required", &csi.CapacityRange{RequiredBytes: 50 * 1024 * 1024 * 1024}, 50, false},
		{"required and limit", &csi.CapacityRange{RequiredBytes: 25 * 1024 * 1024 * 1024, LimitBytes: 100 * 1024 * 1024 * 1024}, 25, false},
		{"required = limit", &csi.CapacityRange{RequiredBytes: 30 * 1024 * 1024 * 1024, LimitBytes: 30 * 1024 * 1024 * 1024}, 30, false},
		{"required = limit (not GB int)", &csi.CapacityRange{RequiredBytes: 3_000_000_000, LimitBytes: 3_000_000_000}, 0, true},
		{"no int GB int possible", &csi.CapacityRange{RequiredBytes: 4_000_000_000, LimitBytes: 1_000_001_000}, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := &csi.CreateVolumeRequest{
				CapacityRange: c.capacityRange,
			}
			size, err := determineSize(req)
			if err != nil && !c.expectError {
				t.Errorf("Unexepcted error: %v", err.Error())
			}
			if err == nil && c.expectError {
				t.Error("Expected an error")
			}
			if size != c.expectedSize {
				t.Errorf("Expected size %v, got %v", c.expectedSize, size)
			}
		})
	}
}
