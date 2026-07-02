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

// GetClientIPFromXFF extracts the first valid IP address from an
// X-Forwarded-For header value. The header may contain a comma-separated
// list of IPs ordered "client, proxy1, proxy2, ..."; the leftmost entry is
// the original client IP (cip). Returns an empty string when the header is
// empty or its first entry is not a valid IP, so callers can fall back to
// other means (e.g. gin.Context.ClientIP).
func GetClientIPFromXFF(xff string) string {
	if xff == "" {
		return ""
	}
	parts := strings.SplitN(xff, ",", 2)
	ip := strings.TrimSpace(parts[0])
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}
