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

func TestIsLoopbackIP(t *testing.T) {
	Convey("TestIsLoopbackIP", t, func() {
		Convey("127.0.0.1 is loopback", func() {
			So(IsLoopbackIP("127.0.0.1"), ShouldBeTrue)
		})
		Convey("::1 is loopback", func() {
			So(IsLoopbackIP("::1"), ShouldBeTrue)
		})
		Convey("127.255.255.255 is loopback (entire 127.0.0.0/8)", func() {
			So(IsLoopbackIP("127.255.255.255"), ShouldBeTrue)
		})
		Convey("external IPv4 is not loopback", func() {
			So(IsLoopbackIP("1.2.3.4"), ShouldBeFalse)
		})
		Convey("private IPv4 is not loopback", func() {
			So(IsLoopbackIP("10.0.0.1"), ShouldBeFalse)
		})
		Convey("whitespace is trimmed", func() {
			So(IsLoopbackIP("  127.0.0.1  "), ShouldBeTrue)
		})
		Convey("empty string is not loopback", func() {
			So(IsLoopbackIP(""), ShouldBeFalse)
		})
		Convey("invalid string is not loopback", func() {
			So(IsLoopbackIP("not-an-ip"), ShouldBeFalse)
		})
	})
}
