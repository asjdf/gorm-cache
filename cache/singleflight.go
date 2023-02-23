package cache

import "sync"

// call is an in-flight or completed singleflight.Do call
type call struct {
	wg sync.WaitGroup

	key string

	// These fields will storage final result and will
	// be written once before the WaitGroup is done
	// and are only read after the WaitGroup is done.
	dest         interface{}
	rowsAffected int64
	err          error

	// forgotten indicates whether Forget was called with this call's key
	// while the call was still in flight.
	forgotten bool

	// These fields are read and written with the singleFlight
	// mutex held before the WaitGroup is done, and are read but
	// not written after the WaitGroup is done.
	dups int
}

// Group represents a class of work and forms a namespace in
// which units of work can be executed with duplicate suppression.
type Group struct {
	mu sync.Mutex       // protects m
	m  map[string]*call // lazily initialized
}

// Forget tells the singleflight to forget about a key.  Future calls
// to Do for this key will call the function rather than waiting for
// an earlier call to complete.
func (g *Group) Forget(key string) {
	g.mu.Lock()
	if c, ok := g.m[key]; ok {
		c.forgotten = true
	}
	delete(g.m, key)
	g.mu.Unlock()
}
