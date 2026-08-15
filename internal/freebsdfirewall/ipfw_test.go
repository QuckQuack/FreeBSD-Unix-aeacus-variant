package freebsdfirewall

import "testing"

func TestIPFWEnabled_returns_true_when_either_address_family_is_enabled(t *testing.T) {
	tests := []struct {
		name string
		ipv4 uint32
		ipv6 uint32
		want bool
	}{
		{name: "IPv4 only", ipv4: 1, ipv6: 0, want: true},
		{name: "IPv6 only", ipv4: 0, ipv6: 1, want: true},
		{name: "both disabled", ipv4: 0, ipv6: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := IPFWEnabled(tt.ipv4, tt.ipv6)

			// Then
			if got != tt.want {
				t.Fatalf("IPFWEnabled(%d, %d) = %v, want %v", tt.ipv4, tt.ipv6, got, tt.want)
			}
		})
	}
}
