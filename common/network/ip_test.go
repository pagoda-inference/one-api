package network

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestIsIpInSubnet(t *testing.T) {
	ctx := context.Background()
	ip1 := "192.168.0.5"
	ip2 := "125.216.250.89"
	subnet := "192.168.0.0/24"
	Convey("TestIsIpInSubnet", t, func() {
		So(isIpInSubnet(ctx, ip1, subnet), ShouldBeTrue)
		So(isIpInSubnet(ctx, ip2, subnet), ShouldBeFalse)
	})
}

func TestGetClientIPFromXFF(t *testing.T) {
	Convey("TestGetClientIPFromXFF", t, func() {
		Convey("single IPv4", func() {
			So(GetClientIPFromXFF("1.2.3.4"), ShouldEqual, "1.2.3.4")
		})
		Convey("comma-separated list takes the first", func() {
			So(GetClientIPFromXFF("1.2.3.4, 5.6.7.8"), ShouldEqual, "1.2.3.4")
		})
		Convey("leading/trailing spaces are trimmed", func() {
			So(GetClientIPFromXFF("  1.2.3.4  , 5.6.7.8"), ShouldEqual, "1.2.3.4")
		})
		Convey("IPv6 is supported", func() {
			So(GetClientIPFromXFF("::1"), ShouldEqual, "::1")
		})
		Convey("empty header returns empty string", func() {
			So(GetClientIPFromXFF(""), ShouldEqual, "")
		})
		Convey("invalid IP returns empty string", func() {
			So(GetClientIPFromXFF("not-an-ip"), ShouldEqual, "")
		})
		Convey("malformed first entry returns empty string", func() {
			So(GetClientIPFromXFF("not-an-ip, 1.2.3.4"), ShouldEqual, "")
		})
	})
}
