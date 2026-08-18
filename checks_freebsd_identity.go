package main

import "github.com/elysium-suite/aeacus/internal/accountdb"

// PasswordChanged passes if the user's exact password field differs from the
// known value in the scoring configuration.
func (c cond) PasswordChanged() (bool, error) {
	c.requireArgs("User", "Value")
	contents, err := readFile("/etc/master.passwd")
	if err != nil {
		return false, err
	}

	hash, err := accountdb.PasswordHash(contents, c.User)
	if err != nil {
		return false, err
	}

	return hash != c.Value, nil
}

// UserInGroup passes if the user has the group's GID as its primary GID or
// appears as an exact supplementary member of the exact group record.
func (c cond) UserInGroup() (bool, error) {
	c.requireArgs("User", "Group")
	contents, err := readFile("/etc/group")
	if err != nil {
		return false, err
	}

	group, found, err := accountdb.FindGroup(contents, c.Group)
	if err != nil || !found {
		return false, err
	}
	if group.HasMember(c.User) {
		return true, nil
	}

	contents, err = readFile("/etc/master.passwd")
	if err != nil {
		return false, err
	}
	primaryGID, err := accountdb.PrimaryGID(contents, c.User)
	if err != nil {
		return false, err
	}
	return primaryGID == group.GID, nil
}
