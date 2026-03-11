//go:build linux

package monitor

import (
	"context"

	"github.com/yanjiulab/lopa/internal/logger"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// Run starts the netlink listener and dispatches matching events to the store.
// It blocks until ctx is done. Linux-only.
func Run(ctx context.Context, store *Store) {
	log := logger.S()
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(done)
	}()

	linkCh := make(chan netlink.LinkUpdate)
	if err := netlink.LinkSubscribe(linkCh, done); err != nil {
		log.Warnw("monitor: link subscribe failed", "err", err)
		return
	}

	addrCh := make(chan netlink.AddrUpdate)
	if err := netlink.AddrSubscribe(addrCh, done); err != nil {
		log.Warnw("monitor: addr subscribe failed", "err", err)
		return
	}

	routeCh := make(chan netlink.RouteUpdate)
	if err := netlink.RouteSubscribe(routeCh, done); err != nil {
		log.Warnw("monitor: route subscribe failed", "err", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case lu, ok := <-linkCh:
			if !ok {
				return
			}
			dispatchLinkUpdate(store, lu)
		case au, ok := <-addrCh:
			if !ok {
				return
			}
			dispatchAddrUpdate(store, au)
		case ru, ok := <-routeCh:
			if !ok {
				return
			}
			dispatchRouteUpdate(store, ru)
		}
	}
}

func dispatchLinkUpdate(store *Store, lu netlink.LinkUpdate) {
	attrs := lu.Link.Attrs()
	change := ChangeUpdate
	if lu.Header.Type == unix.RTM_DELLINK {
		change = ChangeDelete
	}
	detail := &DetailInterface{
		Index:     attrs.Index,
		Name:      attrs.Name,
		MTU:       attrs.MTU,
		OperState: attrs.OperState.String(),
	}
	if attrs.Flags != 0 {
		detail.Flags = attrs.Flags.String()
	}

	tasks := store.TasksForType(TypeInterface)
	for _, t := range tasks {
		if !t.Match(TypeInterface, detail) {
			continue
		}
		evt := Event{Type: TypeInterface, Change: change, Detail: detail}
		evt = store.AppendEvent(t.ID, evt)
		if t.WebhookURL != "" {
			Notify(t.WebhookURL, evt)
		}
	}
}

func dispatchAddrUpdate(store *Store, au netlink.AddrUpdate) {
	change := ChangeAdd
	if !au.NewAddr {
		change = ChangeDelete
	}
	addrStr := au.LinkAddress.IP.String()
	prefixLen, _ := au.LinkAddress.Mask.Size()
	linkName := ""
	if link, err := netlink.LinkByIndex(au.LinkIndex); err == nil {
		linkName = link.Attrs().Name
	}
	detail := &DetailIP{
		InterfaceIndex: au.LinkIndex,
		InterfaceName:  linkName,
		Address:        addrStr,
		PrefixLen:     prefixLen,
	}

	tasks := store.TasksForType(TypeIP)
	for _, t := range tasks {
		if !t.Match(TypeIP, detail) {
			continue
		}
		evt := Event{Type: TypeIP, Change: change, Detail: detail}
		evt = store.AppendEvent(t.ID, evt)
		if t.WebhookURL != "" {
			Notify(t.WebhookURL, evt)
		}
	}
}

func dispatchRouteUpdate(store *Store, ru netlink.RouteUpdate) {
	change := ChangeAdd
	if ru.Type == unix.RTM_DELROUTE {
		change = ChangeDelete
	}
	r := &ru.Route
	dstStr := "default"
	if r.Dst != nil {
		dstStr = r.Dst.String()
	}
	gwStr := ""
	if r.Gw != nil {
		gwStr = r.Gw.String()
	}
	linkName := ""
	if r.LinkIndex > 0 {
		if link, err := netlink.LinkByIndex(r.LinkIndex); err == nil {
			linkName = link.Attrs().Name
		}
	}
	detail := &DetailRoute{
		Table:         r.Table,
		Dst:           dstStr,
		Gw:            gwStr,
		LinkIndex:     r.LinkIndex,
		InterfaceName: linkName,
		Protocol:      int(r.Protocol),
	}

	tasks := store.TasksForType(TypeRoute)
	for _, t := range tasks {
		if !t.Match(TypeRoute, detail) {
			continue
		}
		evt := Event{Type: TypeRoute, Change: change, Detail: detail}
		evt = store.AppendEvent(t.ID, evt)
		if t.WebhookURL != "" {
			Notify(t.WebhookURL, evt)
		}
	}
}
