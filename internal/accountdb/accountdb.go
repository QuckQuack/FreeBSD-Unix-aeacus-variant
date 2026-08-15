package accountdb

import (
	"errors"
	"fmt"
	"strings"
)

var ErrUserNotFound = errors.New("accountdb: user not found")

func PasswordHash(contents, username string) (string, error) {
	return passwdField(contents, username, 1)
}

func PrimaryGID(contents, username string) (string, error) {
	return passwdField(contents, username, 3)
}

func passwdField(contents, username string, field int) (string, error) {
	for lineNumber, line := range strings.Split(contents, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) != 10 {
			return "", fmt.Errorf("master.passwd line %d has %d fields, want 10", lineNumber+1, len(fields))
		}
		if fields[0] == username {
			return fields[field], nil
		}
	}

	return "", fmt.Errorf("%s: %w", username, ErrUserNotFound)
}

type GroupRecord struct {
	GID     string
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
			return GroupRecord{GID: fields[2], members: strings.Split(fields[3], ",")}, true, nil
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
