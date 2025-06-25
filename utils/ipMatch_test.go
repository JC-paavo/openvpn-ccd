package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNetworkConflict(t *testing.T) {
	tests := []struct {
		name     string
		ip1      string
		mask1    string
		ip2      string
		mask2    string
		expected bool
		wantErr  bool
	}{
		{
			name:     "same network",
			ip1:      "192.168.1.1",
			mask1:    "255.255.255.0",
			ip2:      "192.168.1.2",
			mask2:    "255.255.255.0",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "different networks",
			ip1:      "192.168.1.1",
			mask1:    "255.255.255.0",
			ip2:      "192.168.2.1",
			mask2:    "255.255.255.0",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "overlapping networks",
			ip1:      "192.168.1.1",
			mask1:    "255.255.0.0",
			ip2:      "192.168.2.1",
			mask2:    "255.255.255.0",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "invalid ip1",
			ip1:      "invalid",
			mask1:    "255.255.255.0",
			ip2:      "192.168.1.2",
			mask2:    "255.255.255.0",
			expected: false,
			wantErr:  true,
		},
		{
			name:     "invalid mask1",
			ip1:      "192.168.1.1",
			mask1:    "invalid",
			ip2:      "192.168.1.2",
			mask2:    "255.255.255.0",
			expected: false,
			wantErr:  true,
		},
		{
			name:     "invalid ip2",
			ip1:      "192.168.1.1",
			mask1:    "255.255.255.0",
			ip2:      "invalid",
			mask2:    "255.255.255.0",
			expected: false,
			wantErr:  true,
		},
		{
			name:     "invalid mask2",
			ip1:      "192.168.1.1",
			mask1:    "255.255.255.0",
			ip2:      "192.168.1.2",
			mask2:    "invalid",
			expected: false,
			wantErr:  true,
		},
		{
			name:     "ipv6 not supported",
			ip1:      "2001:db8::1",
			mask1:    "ffff:ffff::",
			ip2:      "2001:db8::2",
			mask2:    "ffff:ffff::",
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IsNetworkConflict(tt.ip1, tt.mask1, tt.ip2, tt.mask2)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
