// +build !windows

package dotnetdiag

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
)

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

func DefaultServerAddress(pid int) string {
	paths, err := filepath.Glob(fmt.Sprintf("%s/dotnet-diagnostic-%d-*-socket", os.TempDir(), pid))
	if err != nil || len(paths) == 0 {
		return ""
	}
	sort.Slice(paths, func(i, j int) bool { return paths[i] > paths[j] })
	return paths[0]
}
