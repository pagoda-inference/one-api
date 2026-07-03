package network

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/pagoda-inference/one-api/common/logger"
)

func splitSubnets(subnets string) []string {
	res := strings.Split(subnets, ",")
	for i := 0; i < len(res); i++ {
		res[i] = strings.TrimSpace(res[i])
	}
	return res
}

func isValidSubnet(subnet string) error {
	_, _, err := net.ParseCIDR(subnet)
	if err != nil {
		return fmt.Errorf("failed to parse subnet: %w", err)
	}
	return nil
}

func isIpInSubnet(ctx context.Context, ip string, subnet string) bool {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		logger.Errorf(ctx, "failed to parse subnet: %s", err.Error())
		return false
	}
	return ipNet.Contains(net.ParseIP(ip))
}

func IsValidSubnets(subnets string) error {
	for _, subnet := range splitSubnets(subnets) {
		if err := isValidSubnet(subnet); err != nil {
			return err
		}
	}
	return nil
}

func IsIpInSubnets(ctx context.Context, ip string, subnets string) bool {
	for _, subnet := range splitSubnets(subnets) {
		if isIpInSubnet(ctx, ip, subnet) {
			return true
		}
	}
	return false
}

// GetIPFromRemoteAddr extracts the host IP from an address in "host:port"
// format (e.g. http.Request.RemoteAddr). Returns the IP string if valid,
// or an empty string if the address cannot be parsed or is not a valid IP.
func GetIPFromRemoteAddr(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		if net.ParseIP(remoteAddr) != nil {
			return remoteAddr
		}
		return ""
	}
	if net.ParseIP(host) == nil {
		return ""
	}
	return host
}
