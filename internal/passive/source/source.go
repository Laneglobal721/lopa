package source

// InterfaceSnapshot holds current interface counter values (e.g. from /proc/net/dev or netlink).
type InterfaceSnapshot struct {
	BytesIn    uint64
	BytesOut   uint64
	PacketsIn  uint64
	PacketsOut uint64
	ErrorsIn   uint64
	ErrorsOut  uint64
	DropsIn    uint64
	DropsOut   uint64
}

// Source fetches current interface statistics.
type Source interface {
	// InterfaceStats returns current counters for the named interface (e.g. "eth0").
	// If the interface does not exist or is not readable, returns an error.
	InterfaceStats(iface string) (*InterfaceSnapshot, error)
}
