package filewatcher

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Watcher watches a set of files, delivering events to a channel.
type Watcher struct {
	Events chan Event // Event chan.
	Errors chan error // Errors chan.

	mu       sync.Mutex          // Protects access to watcher data.
	paths    map[string]pathInfo // Map of watched files/directories.
	interval time.Duration       // Poll interval.

	ctx      context.Context    // Context.
	cancel   context.CancelFunc // Context cancel.
	isClosed *int32             // Closed flag.
}

// pathInfo represent a watched tree.
type pathInfo struct {
	name         string              // Absolute path of the watched element.
	lastModified time.Time           // Last modification time, to identify when there is a change.
	paths        map[string]pathInfo // Not null for directories.
}

// NewWatcher instantiates a new file system watcher with the background context.
func NewWatcher() (*Watcher, error) {
	return NewWatcherCtx(context.Background())
}

// NewWatcherCtx instantiates a new file system watcher with the provided context.
func NewWatcherCtx(ctx context.Context) (*Watcher, error) {
	ctx, cancel := context.WithCancel(ctx)
	w := &Watcher{
		Events:   make(chan Event),
		Errors:   make(chan error),
		paths:    map[string]pathInfo{},
		interval: 1 * time.Second,
		cancel:   cancel,
		isClosed: new(int32),
	}
	go w.readEvents(100 * time.Millisecond)
	return w, nil
}

// SetInterval updates the watcher poll interval.
func (w *Watcher) SetInterval(interval time.Duration) {
	w.mu.Lock()
	w.interval = interval
	w.mu.Unlock()
}

// GetInterval is an accessor for the poll interval value.
func (w *Watcher) GetInterval() time.Duration {
	w.mu.Lock()
	interval := w.interval
	w.mu.Unlock()
	return interval
}

// Close termintates the watcher.
func (w *Watcher) Close() error {
	// Make sure we close only once.
	if !atomic.CompareAndSwapInt32(w.isClosed, 0, 1) {
		return nil
	}

	// Cancel the context tree local to the watcher.
	w.cancel()
	return nil
}

// readEvents is the main loop. It fires the file system diff check
// upon the configured interval.
func (w *Watcher) readEvents(interval time.Duration) {
	for {
		timer := time.NewTimer(w.GetInterval())

		// Check if the context is not done.
		select {
		case <-w.ctx.Done():
			// If done, exit the loop.
			timer.Stop()
			close(w.Events)
			close(w.Errors)
			return
		case <-timer.C:

		}
	}
}
