//go:build !linux

package netstack

// IPv6PrefixWatcher is a no-op stub for non-Linux platforms (Windows/Darwin).
type IPv6PrefixWatcher struct {
}

// NewIPv6PrefixWatcher returns a no-op watcher stub on non-Linux platforms.
func NewIPv6PrefixWatcher(stopCh <-chan struct{}, onChanged func(event IPv6AddressEvent)) (*IPv6PrefixWatcher, error) {
	return &IPv6PrefixWatcher{}, nil
}

// Close is a no-op on non-Linux platforms.
func (w *IPv6PrefixWatcher) Close() {
}
