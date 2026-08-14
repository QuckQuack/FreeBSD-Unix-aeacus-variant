package accountdb

import (
	"testing"
)

func TestPasswordHash_returns_exact_password_field(t *testing.T) {
	// Given
	contents := "alice:$6$expected:1001:1001::0:0:Alice:/home/alice:/bin/sh\n"

	// When
	got, err := PasswordHash(contents, "alice")

	// Then
	if err != nil {
		t.Fatalf("PasswordHash() error = %v", err)
	}
	if got != "$6$expected" {
		t.Fatalf("PasswordHash() = %q, want %q", got, "$6$expected")
	}
}

func TestPasswordHash_rejects_record_without_exactly_ten_fields(t *testing.T) {
	tests := []struct {
		name     string
		contents string
	}{
		{
			name:     "nine fields",
			contents: "alice:$6$expected:1001:1001::0:0:/home/alice:/bin/sh\n",
		},
		{
			name:     "eleven fields",
			contents: "alice:$6$expected:1001:1001::0:0:Alice:/home/alice:/bin/sh:extra\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := PasswordHash(tt.contents, "alice")

			// Then
			if err == nil {
				t.Fatal("PasswordHash() error = nil, want malformed-record error")
			}
		})
	}
}

func TestSupplementaryMember_matches_exact_group_and_user(t *testing.T) {
	tests := []struct {
		name  string
		group string
		user  string
		want  bool
	}{
		{name: "exact member", group: "wheel", user: "alice", want: true},
		{name: "username substring", group: "wheel", user: "ann", want: false},
		{name: "group substring", group: "wheel", user: "bob", want: false},
	}
	contents := "wheel:*:0:root,alice,joann\nwheel-admin:*:1002:bob\n"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := SupplementaryMember(contents, tt.group, tt.user)

			// Then
			if err != nil {
				t.Fatalf("SupplementaryMember() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("SupplementaryMember() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSupplementaryMember_rejects_malformed_group_record(t *testing.T) {
	// Given
	contents := "wheel:*:root,alice\n"

	// When
	_, err := SupplementaryMember(contents, "wheel", "alice")

	// Then
	if err == nil {
		t.Fatal("SupplementaryMember() error = nil, want malformed-record error")
	}
}
