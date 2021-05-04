// +build !windows

package dotnetdiag

import "net"

func dial(addr string) (net.Conn, error) {
	ua := &net.UnixAddr{
		Name: addr,
		Net:  "unix",
	}
	conn, err := net.DialUnix("unix", nil, ua)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
