package monitor

import "time"

// TaskType is the kind of netlink events to watch.
type TaskType string

const (
	TypeInterface TaskType = "interface"
	TypeIP        TaskType = "ip"
	TypeRoute     TaskType = "route"
)

// ChangeKind is add/delete/update.
type ChangeKind string

const (
	ChangeAdd    ChangeKind = "add"
	ChangeDelete ChangeKind = "delete"
	ChangeUpdate ChangeKind = "update"
)

// Filter restricts which events a task cares about.
type Filter struct {
	InterfaceName  string `json:"interface_name,omitempty"`  // e.g. "eth0"
	InterfaceIndex int    `json:"interface_index,omitempty"` // 0 = any
	Prefix         string `json:"prefix,omitempty"`          // for ip: e.g. "192.168.0.0/24"
	// Route filters (for type=route)
	RouteTable int    `json:"route_table,omitempty"` // 0 = any table
	RouteDst   string `json:"route_dst,omitempty"`   // CIDR prefix to match route destination, e.g. "0.0.0.0/0"
}

// Task is a monitor task (interface, ip, or route).
type Task struct {
	ID         string    `json:"id"`
	Type       TaskType  `json:"type"`
	Filter     Filter    `json:"filter"`
	WebhookURL string    `json:"webhook_url,omitempty"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Event is a single netlink-derived change.
type Event struct {
	ID      string      `json:"id"`
	TaskID  string      `json:"task_id"`
	Type    TaskType    `json:"type"`
	Change  ChangeKind  `json:"change"`
	Detail  interface{} `json:"detail"`
	At      time.Time   `json:"at"`
}

// DetailInterface is Event.Detail for type=interface.
type DetailInterface struct {
	Index     int    `json:"index"`
	Name      string `json:"name"`
	Flags     string `json:"flags,omitempty"`
	MTU       int    `json:"mtu,omitempty"`
	OperState string `json:"oper_state,omitempty"`
}

// DetailIP is Event.Detail for type=ip.
type DetailIP struct {
	InterfaceIndex int    `json:"interface_index"`
	InterfaceName  string `json:"interface_name,omitempty"`
	Address        string `json:"address"`
	PrefixLen      int    `json:"prefix_len,omitempty"`
}

// DetailRoute is Event.Detail for type=route.
type DetailRoute struct {
	Table          int    `json:"table"`
	Dst            string `json:"dst"`              // e.g. "192.168.0.0/24" or "default"
	Gw             string `json:"gw,omitempty"`     // gateway IP
	LinkIndex      int    `json:"link_index"`
	InterfaceName  string `json:"interface_name,omitempty"`
	Protocol       int    `json:"protocol,omitempty"` // RTPROT_*
}
