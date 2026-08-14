package main

import (
	"os/exec"

	"github.com/elysium-suite/aeacus/internal/freebsdfirewall"
	"golang.org/x/sys/unix"
)

// FirewallUp passes when PF or IPFW is enforcing policy in the running kernel.
func (c cond) FirewallUp() (bool, error) {
	pfInfo, err := exec.Command("/sbin/pfctl", "-s", "info").Output()
	if err == nil && freebsdfirewall.PFEnabled(string(pfInfo)) {
		return true, nil
	}

	ipfwEnabled, err := unix.SysctlUint32("net.inet.ip.fw.enable")
	if err == nil {
		return ipfwEnabled != 0, nil
	}

	return false, nil
}
