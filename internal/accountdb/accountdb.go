package accountdb

import (
	"errors"
	"fmt"
	"strings"
)

var ErrUserNotFound = errors.New("accountdb: user not found")

func PasswordHash(contents, username string) (string, error) {
	for lineNumber, line := range strings.Split(contents, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) != 10 {
			return "", fmt.Errorf("master.passwd line %d has %d fields, want 10", lineNumber+1, len(fields))
		}
		if fields[0] == username {
			return fields[1], nil
		}
	}

	return "", fmt.Errorf("%s: %w", username, ErrUserNotFound)
}

func SupplementaryMember(contents, group, username string) (bool, error) {
	for lineNumber, line := range strings.Split(contents, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) != 4 {
			return false, fmt.Errorf("group line %d has %d fields, want 4", lineNumber+1, len(fields))
		}
		if fields[0] != group {
			continue
		}

		for _, member := range strings.Split(fields[3], ",") {
			if member == username {
				return true, nil
			}
		}
		return false, nil
	}

	return false, nil
}
