package accountdb

import (
	"testing"
)

func TestPasswordHash_accepts_plus_prefixed_gid(t *testing.T) {
	// Given
	contents := "alice:$6$expected:1001:+1001::0:0:Alice:/home/alice:/bin/sh\n"

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

func TestPasswordHash_skips_indented_comments_and_whitespace(t *testing.T) {
	// Given
	contents := "  \n  # managed accounts\nalice:$6$expected:1001:1001::0:0:Alice:/home/alice:/bin/sh\n"

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

func TestSupplementaryMember_skips_indented_comments_and_whitespace(t *testing.T) {
	// Given
	contents := "   \n  # managed group records\nwheel:*:0:root,alice\n"

	// When
	got, err := SupplementaryMember(contents, "wheel", "alice")

	// Then
	if err != nil {
		t.Fatalf("SupplementaryMember() error = %v", err)
	}
	if !got {
		t.Fatal("SupplementaryMember() = false, want true")
	}
}

func TestGroupRecord_accepts_plus_prefixed_gid(t *testing.T) {
	// Given
	contents := "wheel:*:+0:root,alice\n"

	// When
	group, found, err := FindGroup(contents, "wheel")

	// Then
	if err != nil {
		t.Fatalf("FindGroup() error = %v", err)
	}
	if !found {
		t.Fatal("FindGroup() found = false, want true")
	}
	if group.GID != GID(0) {
		t.Fatalf("FindGroup() GID = %d, want %d", group.GID, GID(0))
	}
	if !group.HasMember("alice") {
		t.Fatal("GroupRecord.HasMember() = false, want true")
	}
}

func TestPrimaryGID_accepts_plus_prefixed_gid(t *testing.T) {
	// Given
	contents := "alice:$6$expected:1001:+1002::0:0:Alice:/home/alice:/bin/sh\n"

	// When
	got, err := PrimaryGID(contents, "alice")

	// Then
	if err != nil {
		t.Fatalf("PrimaryGID() error = %v", err)
	}
	if got != GID(1002) {
		t.Fatalf("PrimaryGID() = %d, want %d", got, GID(1002))
	}
}

func TestPrimaryGID_skips_indented_comments_and_whitespace(t *testing.T) {
	// Given
	contents := "\t\n\t# managed accounts\nalice:$6$expected:1001:1002::0:0:Alice:/home/alice:/bin/sh\n"

	// When
	got, err := PrimaryGID(contents, "alice")

	// Then
	if err != nil {
		t.Fatalf("PrimaryGID() error = %v", err)
	}
	if got != GID(1002) {
		t.Fatalf("PrimaryGID() = %d, want %d", got, GID(1002))
	}
}

func TestFindGroup_parses_gid_as_decimal_number(t *testing.T) {
	// Given
	contents := "wheel:*:020:root\n"

	// When
	group, found, err := FindGroup(contents, "wheel")

	// Then
	if err != nil {
		t.Fatalf("FindGroup() error = %v", err)
	}
	if !found {
		t.Fatal("FindGroup() found = false, want true")
	}
	if group.GID != GID(20) {
		t.Fatalf("FindGroup() GID = %d, want %d", group.GID, GID(20))
	}
}

func TestPrimaryGID_parses_gid_as_decimal_number(t *testing.T) {
	// Given
	contents := "alice:*:1001:020::0:0:Alice:/home/alice:/bin/sh\n"

	// When
	got, err := PrimaryGID(contents, "alice")

	// Then
	if err != nil {
		t.Fatalf("PrimaryGID() error = %v", err)
	}
	if got != GID(20) {
		t.Fatalf("PrimaryGID() = %d, want %d", got, GID(20))
	}
}

func TestSupplementaryMember_rejects_invalid_group_gid(t *testing.T) {
	tests := []string{"not-a-gid", "+", "++20", "-1", "4294967296"}
	for _, gid := range tests {
		t.Run(gid, func(t *testing.T) {
			// Given
			contents := "wheel:*:" + gid + ":alice\n"

			// When
			_, err := SupplementaryMember(contents, "wheel", "alice")

			// Then
			if err == nil {
				t.Fatal("SupplementaryMember() error = nil, want invalid-GID error")
			}
		})
	}
}

func TestPrimaryGID_rejects_invalid_gid(t *testing.T) {
	tests := []string{"not-a-gid", "+", "++20", "-1", "4294967296"}
	for _, gid := range tests {
		t.Run(gid, func(t *testing.T) {
			// Given
			contents := "alice:*:1001:" + gid + "::0:0:Alice:/home/alice:/bin/sh\n"

			// When
			_, err := PrimaryGID(contents, "alice")

			// Then
			if err == nil {
				t.Fatal("PrimaryGID() error = nil, want invalid-GID error")
			}
		})
	}
}
