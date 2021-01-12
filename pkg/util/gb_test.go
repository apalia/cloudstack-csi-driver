package util

import (
	"strconv"
	"testing"
)

func TestRoundUpBytesToGB(t *testing.T) {
	cases := []struct {
		b          int64
		expectedGb int64
	}{
		{100, 1},
		{3221225472, 3},
		{3000000000, 3},
		{50 * 1024 * 1024 * 1024, 50},
		{50*1024*1024*1024 - 1, 50},
		{50*1024*1024*1024 + 1, 51},
	}
	for _, c := range cases {
		t.Run(strconv.FormatInt(c.b, 10), func(t *testing.T) {
			gb := RoundUpBytesToGB(c.b)
			if gb != c.expectedGb {
				t.Errorf("%v bytes: expecting %v, got %v", c.b, c.expectedGb, gb)
			}
		})
	}
}

func TestGigaBytesToBytes(t *testing.T) {
	var gb int64 = 5
	b := GigaBytesToBytes(gb)
	var expectedBytes int64 = 5368709120
	if b != expectedBytes {
		t.Errorf("Expected %v, got %v", expectedBytes, b)
	}
	back := RoundUpBytesToGB(b)
	if back != gb {
		t.Errorf("Expected %v, got %v", gb, back)
	}
}
