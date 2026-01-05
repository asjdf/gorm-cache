package cache

import (
	"sync"
	"testing"
)

func TestGroup_Forget(t *testing.T) {
	g := &Group{}
	
	// Test Forget with nil map
	g.Forget("test-key")
	
	// Test Forget with existing key
	g.m = make(map[string]*call)
	c := &call{key: "test-key"}
	g.m["test-key"] = c
	g.Forget("test-key")
	
	if c.forgotten != true {
		t.Error("expected forgotten to be true")
	}
	if _, ok := g.m["test-key"]; ok {
		t.Error("expected key to be deleted")
	}
	
	// Test Forget with non-existing key
	g.Forget("non-existing-key")
}

func TestGroup_Forget_Concurrent(t *testing.T) {
	g := &Group{
		m: make(map[string]*call),
	}
	
	// Add some calls
	for i := 0; i < 10; i++ {
		key := "key-" + string(rune('0'+i))
		g.m[key] = &call{key: key}
	}
	
	// Concurrent Forget calls
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key-" + string(rune('0'+i))
			g.Forget(key)
		}(i)
	}
	wg.Wait()
	
	// All keys should be deleted
	if len(g.m) != 0 {
		t.Errorf("expected map to be empty, got %d keys", len(g.m))
	}
}

