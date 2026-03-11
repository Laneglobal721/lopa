package source

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const procNetDev = "/proc/net/dev"

// ProcNetDevSource reads interface statistics from /proc/net/dev (Linux).
type ProcNetDevSource struct{}

// InterfaceStats returns current counters for the named interface.
func (ProcNetDevSource) InterfaceStats(iface string) (*InterfaceSnapshot, error) {
	f, err := os.Open(procNetDev)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", procNetDev, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Skip header lines (2 lines)
	for i := 0; i < 2 && scanner.Scan(); i++ {
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", procNetDev, err)
	}

	target := strings.TrimSpace(iface)
	for scanner.Scan() {
		line := scanner.Text()
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:idx])
		if name != target {
			continue
		}
		fields := strings.Fields(line[idx+1:])
		// Receive: bytes(0), packets(1), errs(2), drop(3), fifo(4), frame(5), compressed(6), multicast(7)
		// Transmit: bytes(8), packets(9), errs(10), drop(11), ...
		if len(fields) < 12 {
			return nil, fmt.Errorf("interface %s: not enough columns in %s", iface, procNetDev)
		}
		s := &InterfaceSnapshot{}
		s.BytesIn, _ = strconv.ParseUint(fields[0], 10, 64)
		s.PacketsIn, _ = strconv.ParseUint(fields[1], 10, 64)
		s.ErrorsIn, _ = strconv.ParseUint(fields[2], 10, 64)
		s.DropsIn, _ = strconv.ParseUint(fields[3], 10, 64)
		s.BytesOut, _ = strconv.ParseUint(fields[8], 10, 64)
		s.PacketsOut, _ = strconv.ParseUint(fields[9], 10, 64)
		s.ErrorsOut, _ = strconv.ParseUint(fields[10], 10, 64)
		s.DropsOut, _ = strconv.ParseUint(fields[11], 10, 64)
		return s, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", procNetDev, err)
	}
	return nil, fmt.Errorf("interface %s not found in %s", iface, procNetDev)
}
