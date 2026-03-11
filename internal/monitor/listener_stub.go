//go:build !linux

package monitor

import "context"

// Run is a no-op on non-Linux; netlink is Linux-only.
func Run(ctx context.Context, store *Store) {}
