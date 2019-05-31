package utils

import "strings"

// RemovePOrt removes the port in IP address
func RemovePort(ip string) string {
	if ip == "" {
		return ""
	}
	splitIp := strings.Split(ip, ":")
	return splitIp[0]
}