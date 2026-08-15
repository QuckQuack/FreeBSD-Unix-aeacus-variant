package accountdb

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrUserNotFound = errors.New("accountdb: user not found")

type GID uint32

type passwdRecord struct {
	password   string
	primaryGID GID
}

func PasswordHash(contents, username string) (string, error) {
	record, err := findPasswdRecord(contents, username)
	return record.password, err
}

func PrimaryGID(contents, username string) (GID, error) {
	record, err := findPasswdRecord(contents, username)
	return record.primaryGID, err
}

func findPasswdRecord(contents, username string) (passwdRecord, error) {
	for lineNumber, line := range strings.Split(contents, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) != 10 {
			return passwdRecord{}, fmt.Errorf("master.passwd line %d has %d fields, want 10", lineNumber+1, len(fields))
		}
		if fields[0] == username {
			gid, err := parseGID(fields[3], "master.passwd", lineNumber+1)
			if err != nil {
				return passwdRecord{}, err
			}
			return passwdRecord{password: fields[1], primaryGID: gid}, nil
		}
	}

	return passwdRecord{}, fmt.Errorf("%s: %w", username, ErrUserNotFound)
}

type GroupRecord struct {
	GID     GID
	members []string
}

func (g GroupRecord) HasMember(username string) bool {
	for _, member := range g.members {
		if member == username {
			return true
		}
	}
	return false
}

func FindGroup(contents, group string) (GroupRecord, bool, error) {
	for lineNumber, line := range strings.Split(contents, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) != 4 {
			return GroupRecord{}, false, fmt.Errorf("group line %d has %d fields, want 4", lineNumber+1, len(fields))
		}
		if fields[0] == group {
			gid, err := parseGID(fields[2], "group", lineNumber+1)
			if err != nil {
				return GroupRecord{}, false, err
			}
			return GroupRecord{GID: gid, members: strings.Split(fields[3], ",")}, true, nil
		}
	}

	return GroupRecord{}, false, nil
}

func SupplementaryMember(contents, group, username string) (bool, error) {
	record, found, err := FindGroup(contents, group)
	if err != nil || !found {
		return false, err
	}
	return record.HasMember(username), nil
}

func parseGID(value, database string, lineNumber int) (GID, error) {
	unsigned := strings.TrimPrefix(value, "+")
	gid, err := strconv.ParseUint(unsigned, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s line %d has invalid GID %q: %w", database, lineNumber, value, err)
	}
	return GID(gid), nil
}
