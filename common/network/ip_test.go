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

func TestGetIPFromRemoteAddr(t *testing.T) {
	Convey("TestGetIPFromRemoteAddr", t, func() {
		Convey("IPv4 with port", func() {
			So(GetIPFromRemoteAddr("1.2.3.4:12345"), ShouldEqual, "1.2.3.4")
		})
		Convey("IPv6 with port", func() {
			So(GetIPFromRemoteAddr("[::1]:12345"), ShouldEqual, "::1")
		})
		Convey("plain IPv4 without port", func() {
			So(GetIPFromRemoteAddr("1.2.3.4"), ShouldEqual, "1.2.3.4")
		})
		Convey("empty string returns empty", func() {
			So(GetIPFromRemoteAddr(""), ShouldEqual, "")
		})
		Convey("invalid value returns empty", func() {
			So(GetIPFromRemoteAddr("invalid"), ShouldEqual, "")
		})
	})
}
