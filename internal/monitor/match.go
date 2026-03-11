package monitor

import "net"

// Match returns true if the task filter matches the given event detail.
// For interface events, detail is *DetailInterface; for ip, *DetailIP.
func (t *Task) Match(typ TaskType, detail interface{}) bool {
	if t.Type != typ {
		return false
	}
	switch typ {
	case TypeInterface:
		if d, ok := detail.(*DetailInterface); ok {
			if t.Filter.InterfaceName != "" && t.Filter.InterfaceName != d.Name {
				return false
			}
			if t.Filter.InterfaceIndex != 0 && t.Filter.InterfaceIndex != d.Index {
				return false
			}
			return true
		}
	case TypeIP:
		if d, ok := detail.(*DetailIP); ok {
			if t.Filter.InterfaceName != "" && t.Filter.InterfaceName != d.InterfaceName {
				return false
			}
			if t.Filter.InterfaceIndex != 0 && t.Filter.InterfaceIndex != d.InterfaceIndex {
				return false
			}
			if t.Filter.Prefix != "" && !ipInPrefix(t.Filter.Prefix, d.Address) {
				return false
			}
			return true
		}
	case TypeRoute:
		if d, ok := detail.(*DetailRoute); ok {
			if t.Filter.RouteTable != 0 && t.Filter.RouteTable != d.Table {
				return false
			}
			if t.Filter.RouteDst != "" && !routeDstMatches(t.Filter.RouteDst, d.Dst) {
				return false
			}
			return true
		}
	}
	return false
}

// routeDstMatches returns true if the filter CIDR matches the route dst string.
// Filter "0.0.0.0/0" or "::/0" matches default route (dst "default" or "").
// Otherwise filter is a CIDR and we require route dst to be in that network.
func routeDstMatches(filterCIDR, routeDst string) bool {
	if routeDst == "" || routeDst == "default" {
		_, n, err := net.ParseCIDR(filterCIDR)
		if err != nil || n == nil {
			return false
		}
		return n.Contains(net.IPv4zero) || n.Contains(net.IPv6zero)
	}
	// Parse route Dst as CIDR or IP; check if it's in filter network
	_, filterNet, err := net.ParseCIDR(filterCIDR)
	if err != nil || filterNet == nil {
		return false
	}
	// routeDst is e.g. "192.168.0.0/24"
	ip, _, err := net.ParseCIDR(routeDst)
	if err != nil {
		ip = net.ParseIP(routeDst)
	}
	if ip == nil {
		return false
	}
	return filterNet.Contains(ip)
}

func ipInPrefix(cidr, addrStr string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(addrStr)
	if ip == nil {
		return false
	}
	return network.Contains(ip)
}
