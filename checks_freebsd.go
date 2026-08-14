package main

import (
	"encoding/binary"
	"errors"
	"os"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// AutoCheckUpdatesEnabled passes if the base system is configured to
// automatically check for (and/or apply) security updates. This looks for a
// live (uncommented) "freebsd-update cron" entry in /etc/crontab -- the
// mechanism recommended by the FreeBSD Handbook -- and falls back to
// checking whether the periodic(8) daily security freebsd-update job is
// enabled in /etc/periodic.conf.
func (c cond) AutoCheckUpdatesEnabled() (bool, error) {
	cronResult, cronErr := cond{
		Path:  "/etc/crontab",
		Value: `(?m)^[^#\r\n]*freebsd-update\s+cron`,
		regex: true,
	}.FileContains()
	if cronErr == nil && cronResult {
		return true, nil
	}

	periodicResult, periodicErr := cond{
		Path:  "/etc/periodic.conf",
		Value: `(?im)^\s*daily_status_security_freebsdupdate_enable\s*=\s*"?(yes|true)"?`,
		regex: true,
	}.FileContains()
	if periodicErr != nil {
		if cronErr != nil {
			return false, cronErr
		}
		return false, periodicErr
	}
	return periodicResult, nil
}

// Command checks if a given shell command ran successfully (that is, did not
// return or raise any errors).
func (c cond) Command() (bool, error) {
	c.requireArgs("Cmd")
	if c.Cmd == "" {
		fail("Missing command for", c.Type)
	}
	err := shellCommand(c.Cmd)
	if err != nil {
		// This check does not return errors, since it is based on successful
		// execution. If any errors occurred, it means that the check failed,
		// not errored out.
		return false, nil
	}
	return true, nil
}

// GuestDisabledLDM passes if guest login is disabled for LightDM. On
// FreeBSD, LightDM (like all packages) is installed under /usr/local.
func (c cond) GuestDisabledLDM() (bool, error) {
	guestStr := `\s*allow-guest\s*=\s*false`
	result, err := cond{
		Path:  "/usr/local/etc/lightdm/lightdm.conf.d/",
		Value: guestStr,
		regex: true,
	}.DirContains()
	if !result {
		return cond{
			Path:  "/usr/local/etc/lightdm/",
			Value: guestStr,
			regex: true,
		}.DirContains()
	}
	return result, err
}

// KernelVersion passes if the running kernel release matches the value
// given. Go's syscall package does not expose a Linux-style uname(2) call on
// FreeBSD, so the kern.osrelease sysctl node is used instead -- it holds the
// same information (e.g. "14.1-RELEASE-p3").
func (c cond) KernelVersion() (bool, error) {
	c.requireArgs("Value")
	release, err := syscall.Sysctl("kern.osrelease")
	if err != nil {
		return false, err
	}
	debug("System kern.osrelease value is", release, "and our value is", c.Value)
	return release == c.Value, nil
}

// FileOwner passes if the given path is owned by the given user.
func (c cond) FileOwner() (bool, error) {
	c.requireArgs("Path", "Name")
	u, err := user.Lookup(c.Name)
	if err != nil {
		return false, err
	}

	f, err := os.Stat(c.Path)
	if err != nil {
		return false, err
	}

	uid := f.Sys().(*syscall.Stat_t).Uid
	o, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return false, err
	}
	debug("File owner for", c.Path, "uid is", strconv.FormatUint(uint64(uid), 10))
	return uint32(o) == uid, nil
}

// PermissionIs passes if a given path's permission string matches (with
// optional wildcards) the value provided.
func (c cond) PermissionIs() (bool, error) {
	c.requireArgs("Path", "Value")
	f, err := os.Stat(c.Path)
	if err != nil {
		return false, err
	}

	fileMode := f.Mode()
	modeBytes := []byte(fileMode.String())
	if len(modeBytes) != 10 {
		fail("System permission string is wrong length:", string(modeBytes))
		return false, errors.New("Invalid system permission string")
	}

	// Permission string includes suid/sgid as the special bit (MSB), while
	// GNU coreutils (and FreeBSD's own ls) replace the executable bit, which
	// we need to emulate.
	if fileMode&os.ModeSetuid != 0 {
		modeBytes[0] = '-'
		modeBytes[3] = 's'
	}
	if fileMode&os.ModeSetgid != 0 {
		modeBytes[0] = '-'
		modeBytes[6] = 's'
	}

	c.Value = strings.TrimSpace(c.Value)

	if len(c.Value) == 9 {
		// If we're provided a mode string of only 9 characters, we'll assume
		// that the 0th bit is irrelevant and should be a wildcard
		c.Value = "?" + c.Value
	} else if len(c.Value) != 10 {
		fail("Your permission string is the wrong length (should be 9 or 10 characters):", c.Value)
		return false, errors.New("Invalid user permission string")
	}

	for i := 0; i < len(c.Value); i++ {
		if c.Value[i] == '?' {
			continue
		}
		if c.Value[i] != modeBytes[i] {
			return false, nil
		}
	}
	return true, nil
}

// ProgramInstalled passes if the given package is installed via pkg(8).
func (c cond) ProgramInstalled() (bool, error) {
	c.requireArgs("Name")
	return cond{
		Cmd: "pkg info -e " + c.Name,
	}.Command()
}

// ProgramVersion passes if the given package's installed version matches the
// value provided.
func (c cond) ProgramVersion() (bool, error) {
	c.requireArgs("Name", "Value")
	return cond{
		Cmd:   `pkg query %v ` + c.Name,
		Value: c.Value,
	}.CommandOutput()
}

// ServiceUp passes if the given rc.d service is currently running.
func (c cond) ServiceUp() (bool, error) {
	c.requireArgs("Name")
	return cond{
		Cmd: "service " + c.Name + " onestatus",
	}.Command()
}

// UserExists passes if the given user exists in /etc/passwd.
func (c cond) UserExists() (bool, error) {
	c.requireArgs("User")
	return cond{
		Path:  "/etc/passwd",
		Value: "^" + c.User + ":",
		regex: true,
	}.FileContains()
}

// -----------------------------------------------------------------------
// FreeBSD-only bonus checks
//
// These have no direct Linux equivalent in aeacus, and take advantage of
// mechanisms that are unique (or much more idiomatic) to FreeBSD.
// -----------------------------------------------------------------------

// SysctlValue passes if the given sysctl MIB's value matches the value
// provided. Set "key" to the MIB name (e.g. "security.bsd.see_other_uids")
// and "value" to the expected string value (e.g. "0").
//
// Most of the MIBs this check is actually written against
// (security.bsd.*, kern.*, and similar hardening toggles) are integers at
// the kernel level, not strings. The standard library's syscall.Sysctl
// only knows how to decode string-typed sysctls -- for an integer MIB it
// reads back the raw binary bytes of the integer and reinterprets them as
// if they were text. For a value of 0, that's four NUL bytes, not the
// character "0", so the comparison against a config value like "0" would
// essentially never match. That made this check structurally unpassable
// for the exact kind of sysctl it exists to verify, regardless of the
// box's real configuration.
//
// This reads the raw bytes directly via the sysctl(2) syscall and decodes
// them by length: 4 bytes as a 32-bit integer, 8 bytes as a 64-bit
// integer (the two sizes FreeBSD's int/uint and long/quad sysctls
// actually use), formatted as a plain decimal string for comparison.
// Anything else falls back to the original NUL-trimmed string behavior,
// so genuinely string-typed sysctls still work as before.
func (c cond) SysctlValue() (bool, error) {
	c.requireArgs("Key", "Value")
	raw, err := unix.SysctlRaw(c.Key)
	if err != nil {
		return false, err
	}

	var result string
	switch len(raw) {
	case 4:
		result = strconv.FormatInt(int64(int32(binary.LittleEndian.Uint32(raw))), 10)
	case 8:
		result = strconv.FormatInt(int64(binary.LittleEndian.Uint64(raw)), 10)
	default:
		result = strings.TrimRight(string(raw), "\x00")
	}

	debug("Sysctl", c.Key, "is", result, "and our value is", c.Value)
	return result == c.Value, nil
}

// PkgAuditClean passes if `pkg audit` reports no known vulnerabilities in
// currently installed packages.
//
// `pkg audit` needs a local copy of the vulnerability database (normally
// fetched via `pkg audit -F`), and nothing else in this codebase ever
// fetches one. Without it, `pkg audit -q` returns a nonzero exit code
// unconditionally -- meaning this check was previously unpassable
// regardless of the actual state of installed packages, on every image.
// Fetching it here (-F) as part of the check fixes that on any image with
// working network access, without changing the fail-closed behavior on one
// that doesn't: a fetch failure still exits nonzero and this still counts
// as a failed check, same as every other check in this codebase when its
// underlying command errors. That's intentional -- treating a missing or
// unfetchable database as an automatic pass would let an image dodge this
// check entirely by simply denying it network access or deleting the
// database, which is a worse hole than the one being fixed here.
func (c cond) PkgAuditClean() (bool, error) {
	out, err := shellCommandOutput("pkg audit -F -q")
	if err != nil {
		// pkg audit returns a nonzero exit code when vulnerabilities are
		// found, when the database fetch above failed, or when the
		// database is otherwise unusable. All three are treated as a
		// failed check rather than an unscoreable error, consistent with
		// every other check's error handling in this codebase.
		return false, nil
	}
	return strings.TrimSpace(out) == "", nil
}

// RcVarEnabled passes if the given rc.conf(5) variable is set to YES/true,
// checking both /etc/rc.conf and any /etc/rc.conf.d/<name> overrides.
func (c cond) RcVarEnabled() (bool, error) {
	c.requireArgs("Name")
	pattern := `(?im)^\s*` + regexp.QuoteMeta(c.Name) + `\s*=\s*"?(yes|true)"?`

	result, err := cond{
		Path:  "/etc/rc.conf",
		Value: pattern,
		regex: true,
	}.FileContains()
	if err == nil && result {
		return true, nil
	}

	return cond{
		Path:  "/etc/rc.conf.d/",
		Value: pattern,
		regex: true,
	}.DirContains()
}

// JailRunning passes if a jail with the given name is currently running.
func (c cond) JailRunning() (bool, error) {
	c.requireArgs("Name")
	return cond{
		Cmd:   "jls -j " + c.Name + " name",
		Value: c.Name,
	}.CommandContains()
}
