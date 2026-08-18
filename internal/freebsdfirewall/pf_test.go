package freebsdfirewall

import "testing"

func TestPFEnabled_requires_enabled_runtime_status(t *testing.T) {
	tests := []struct {
		name string
		info string
		want bool
	}{
		{
			name: "enabled",
			info: "Status: Enabled for 0 days 00:01:00\nDebug: Urgent\n",
			want: true,
		},
		{
			name: "disabled",
			info: "Status: Disabled\nDebug: Urgent\n",
			want: false,
		},
		{
			name: "enabled word outside status",
			info: "Status: Disabled\nNote: Enabled in rc.conf\n",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := PFEnabled(tt.info)

			// Then
			if got != tt.want {
				t.Fatalf("PFEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
