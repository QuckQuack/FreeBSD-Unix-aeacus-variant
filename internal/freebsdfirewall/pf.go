package freebsdfirewall

import "strings"

func PFEnabled(info string) bool {
	for _, line := range strings.Split(info, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "Status:" {
			return fields[1] == "Enabled"
		}
	}

	return false
}
