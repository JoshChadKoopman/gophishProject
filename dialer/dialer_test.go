package dialer

import (
	"fmt"
	"net"
	"strings"
	"syscall"
	"testing"
)

// Test-scoped constants to satisfy lint rules about repeated literals.
const (
	denyErrSubstring  = "upstream connection denied"
	allowedErrFmt     = "error dialing allowed host. got %v"
	unexpectedHostFmt = "unexpected allowed hosts: %v"
	loopbackIPv4      = "127.0.0.1"
	publicIPv4        = "1.1.1.1"
	internalAddr      = "192.168.1.2:80"
	testCIDR24        = "10.0.0.0/24"
)

// ---------- restrictedControl tests ----------

func TestDefaultDeny(t *testing.T) {
	control := restrictedControl([]*net.IPNet{})
	host := "169.254.169.254"
	conn := new(syscall.RawConn)
	got := control("tcp4", fmt.Sprintf("%s:80", host), *conn)
	if got == nil || !strings.Contains(got.Error(), denyErrSubstring) {
		t.Fatalf("expected deny error for link-local host, got %v", got)
	}
}

func TestDefaultAllow(t *testing.T) {
	control := restrictedControl([]*net.IPNet{})
	conn := new(syscall.RawConn)
	got := control("tcp4", fmt.Sprintf("%s:80", publicIPv4), *conn)
	if got != nil {
		t.Fatalf(allowedErrFmt, got)
	}
}

func TestCustomAllow(t *testing.T) {
	_, ipRange, _ := net.ParseCIDR(fmt.Sprintf("%s/32", loopbackIPv4))
	allowed := []*net.IPNet{ipRange}
	control := restrictedControl(allowed)
	conn := new(syscall.RawConn)
	got := control("tcp4", fmt.Sprintf("%s:80", loopbackIPv4), *conn)
	if got != nil {
		t.Fatalf(allowedErrFmt, got)
	}
}

func TestCustomDeny(t *testing.T) {
	_, ipRange, _ := net.ParseCIDR(fmt.Sprintf("%s/32", loopbackIPv4))
	allowed := []*net.IPNet{ipRange}
	control := restrictedControl(allowed)
	conn := new(syscall.RawConn)
	got := control("tcp4", internalAddr, *conn)
	if got == nil || !strings.Contains(got.Error(), denyErrSubstring) {
		t.Fatalf("expected deny error for internal host, got %v", got)
	}
}

func TestSingleIP(t *testing.T) {
	orig := DefaultDialer.AllowedHosts()
	defer func() {
		DefaultDialer.allowedHosts = nil
		DefaultDialer.SetAllowedHosts(orig)
	}()

	DefaultDialer.allowedHosts = nil
	DefaultDialer.SetAllowedHosts([]string{loopbackIPv4})
	control := DefaultDialer.Dialer().Control
	conn := new(syscall.RawConn)

	got := control("tcp4", internalAddr, *conn)
	if got == nil || !strings.Contains(got.Error(), denyErrSubstring) {
		t.Fatalf("expected deny error, got %v", got)
	}

	DefaultDialer.allowedHosts = nil
	host := "::1"
	DefaultDialer.SetAllowedHosts([]string{host})
	control = DefaultDialer.Dialer().Control
	conn = new(syscall.RawConn)

	got = control("tcp4", internalAddr, *conn)
	if got == nil || !strings.Contains(got.Error(), denyErrSubstring) {
		t.Fatalf("expected deny error, got %v", got)
	}

	// Test an allowed connection
	got = control("tcp6", fmt.Sprintf("[%s]:80", host), *conn)
	if got != nil {
		t.Fatalf(allowedErrFmt, got)
	}
}

// ---------- Unsafe network type ----------

func TestUnsafeNetworkType(t *testing.T) {
	control := restrictedControl([]*net.IPNet{})
	conn := new(syscall.RawConn)
	got := control("udp4", fmt.Sprintf("%s:53", publicIPv4), *conn)
	if got == nil || !strings.Contains(got.Error(), "not a safe network type") {
		t.Fatalf("expected unsafe-network error, got %v", got)
	}
}

// ---------- Invalid address format ----------

func TestInvalidAddressFormat(t *testing.T) {
	control := restrictedControl([]*net.IPNet{})
	conn := new(syscall.RawConn)
	// No port in address
	got := control("tcp4", publicIPv4, *conn)
	if got == nil || !strings.Contains(got.Error(), "not a valid host/port pair") {
		t.Fatalf("expected invalid host/port error, got %v", got)
	}
}

func TestInvalidIPAddress(t *testing.T) {
	control := restrictedControl([]*net.IPNet{})
	conn := new(syscall.RawConn)
	got := control("tcp4", "not-an-ip:80", *conn)
	if got == nil || !strings.Contains(got.Error(), "not a valid IP address") {
		t.Fatalf("expected invalid IP error, got %v", got)
	}
}

// ---------- IPv6 tests ----------

func TestIPv6LinkLocalDeniedByDefault(t *testing.T) {
	_, ipRange, _ := net.ParseCIDR("8.8.8.8/32")
	allowed := []*net.IPNet{ipRange}
	control := restrictedControl(allowed)
	conn := new(syscall.RawConn)
	got := control("tcp6", "[fe80::1]:80", *conn)
	if got == nil || !strings.Contains(got.Error(), denyErrSubstring) {
		t.Fatalf("expected deny for IPv6 link-local when custom hosts set, got %v", got)
	}
}

func TestIPv6LoopbackDeniedWithCustomHosts(t *testing.T) {
	_, ipRange, _ := net.ParseCIDR("8.8.8.8/32")
	allowed := []*net.IPNet{ipRange}
	control := restrictedControl(allowed)
	conn := new(syscall.RawConn)
	got := control("tcp6", "[::1]:80", *conn)
	if got == nil || !strings.Contains(got.Error(), denyErrSubstring) {
		t.Fatalf("expected deny for IPv6 loopback, got %v", got)
	}
}

func TestIPv6ExternalAllowed(t *testing.T) {
	control := restrictedControl([]*net.IPNet{})
	conn := new(syscall.RawConn)
	got := control("tcp6", "[2606:4700:4700::1111]:80", *conn)
	if got != nil {
		t.Fatalf("expected external IPv6 to be allowed, got %v", got)
	}
}

// ---------- SetAllowedHosts tests ----------

func TestSetAllowedHostsInvalidCIDR(t *testing.T) {
	d := &RestrictedDialer{}
	err := d.SetAllowedHosts([]string{"not-valid-cidr"})
	if err == nil {
		t.Fatal("expected error for invalid CIDR")
	}
}

func TestSetAllowedHostsCIDRRange(t *testing.T) {
	d := &RestrictedDialer{}
	err := d.SetAllowedHosts([]string{testCIDR24})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hosts := d.AllowedHosts()
	if len(hosts) != 1 || hosts[0] != testCIDR24 {
		t.Fatalf(unexpectedHostFmt, hosts)
	}
}

func TestSetAllowedHostsIPv6Single(t *testing.T) {
	d := &RestrictedDialer{}
	err := d.SetAllowedHosts([]string{"2001:db8::1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hosts := d.AllowedHosts()
	if len(hosts) != 1 || hosts[0] != "2001:db8::1/128" {
		t.Fatalf(unexpectedHostFmt, hosts)
	}
}

// ---------- AllowedHosts on empty dialer ----------

func TestAllowedHostsEmpty(t *testing.T) {
	d := &RestrictedDialer{}
	hosts := d.AllowedHosts()
	if len(hosts) != 0 {
		t.Fatalf("expected empty allowed hosts, got %v", hosts)
	}
}

// ---------- Global SetAllowedHosts function ----------

func TestGlobalSetAllowedHosts(t *testing.T) {
	orig := DefaultDialer.AllowedHosts()
	defer func() {
		DefaultDialer.allowedHosts = nil
		DefaultDialer.SetAllowedHosts(orig)
	}()

	DefaultDialer.allowedHosts = nil
	SetAllowedHosts([]string{"10.0.0.1"})
	hosts := DefaultDialer.AllowedHosts()
	if len(hosts) != 1 || hosts[0] != "10.0.0.1/32" {
		t.Fatalf(unexpectedHostFmt, hosts)
	}
}

// ---------- Global Dialer function ----------

func TestGlobalDialerFunction(t *testing.T) {
	d := Dialer()
	if d == nil {
		t.Fatal("expected non-nil dialer")
	}
	if d.Timeout.Seconds() != 30 {
		t.Fatalf("expected 30s timeout, got %v", d.Timeout)
	}
}

// ---------- Pre-parsed CIDR lists ----------

func TestDefaultDenyListPreParsed(t *testing.T) {
	if len(defaultDeny) == 0 {
		t.Fatal("expected pre-parsed defaultDeny list to be non-empty")
	}
}

func TestAllInternalListPreParsed(t *testing.T) {
	if len(allInternal) == 0 {
		t.Fatal("expected pre-parsed allInternal list to be non-empty")
	}
	// Sanity check: allInternal should contain 169.254.0.0/16
	found := false
	for _, n := range allInternal {
		if n.String() == "169.254.0.0/16" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected allInternal to contain 169.254.0.0/16")
	}
}

// ---------- Helper function unit tests ----------

func TestIsAllowed(t *testing.T) {
	_, ipRange, _ := net.ParseCIDR(testCIDR24)
	allowed := []*net.IPNet{ipRange}

	if !isAllowed(net.ParseIP("10.0.0.5"), allowed) {
		t.Fatal("expected 10.0.0.5 to be in allowed range")
	}
	if isAllowed(net.ParseIP("10.0.1.5"), allowed) {
		t.Fatal("expected 10.0.1.5 to NOT be in allowed range")
	}
}

func TestIsDenied(t *testing.T) {
	if !isDenied(net.ParseIP("169.254.169.254"), defaultDeny) {
		t.Fatal("expected 169.254.169.254 to be denied by default deny list")
	}
	if isDenied(net.ParseIP("1.1.1.1"), defaultDeny) {
		t.Fatal("expected 1.1.1.1 to NOT be denied by default deny list")
	}
}
