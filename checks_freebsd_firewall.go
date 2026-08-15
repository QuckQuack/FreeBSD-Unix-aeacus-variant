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

	var ipv4Enabled uint32
	if value, err := unix.SysctlUint32("net.inet.ip.fw.enable"); err == nil {
		ipv4Enabled = value
	}
	var ipv6Enabled uint32
	if value, err := unix.SysctlUint32("net.inet6.ip6.fw.enable"); err == nil {
		ipv6Enabled = value
	}
	return freebsdfirewall.IPFWEnabled(ipv4Enabled, ipv6Enabled), nil
}
