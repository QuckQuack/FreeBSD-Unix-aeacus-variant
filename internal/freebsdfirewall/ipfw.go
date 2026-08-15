package freebsdfirewall

func IPFWEnabled(ipv4, ipv6 uint32) bool {
	return ipv4 != 0 || ipv6 != 0
}
