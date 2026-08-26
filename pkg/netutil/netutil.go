// Package netutil provides network utility functions.
package netutil

import (
	"net"
	"time"
)

// GetLocalIP returns the first non-loopback local IP address.
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

// IsPortAvailable checks if a port is available for binding.
func IsPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", itoa(port)))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// itoa converts an integer to its string representation (simple version).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [10]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ConnectionCheck checks if a TCP connection can be established.
func ConnectionCheck(host string, port int, timeout time.Duration) bool {
	addr := net.JoinHostPort(host, itoa(port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
