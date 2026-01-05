package cache

import (
	"sync"
	"testing"
)

func TestStats_ResetHitCount(t *testing.T) {
	st := &stats{
		hitCount:  10,
		missCount: 5,
	}
	
	st.ResetHitCount()
	
	if st.HitCount() != 0 {
		t.Errorf("expected hit count to be 0, got %d", st.HitCount())
	}
	if st.MissCount() != 0 {
		t.Errorf("expected miss count to be 0, got %d", st.MissCount())
	}
}

func TestStats_IncrHitCount(t *testing.T) {
	st := &stats{}
	
	// Increment multiple times
	for i := uint64(1); i <= 5; i++ {
		result := st.IncrHitCount()
		if result != i {
			t.Errorf("expected hit count to be %d, got %d", i, result)
		}
	}
	
	if st.HitCount() != 5 {
		t.Errorf("expected hit count to be 5, got %d", st.HitCount())
	}
}

func TestStats_IncrMissCount(t *testing.T) {
	st := &stats{}
	
	// Increment multiple times
	for i := uint64(1); i <= 3; i++ {
		result := st.IncrMissCount()
		if result != i {
			t.Errorf("expected miss count to be %d, got %d", i, result)
		}
	}
	
	if st.MissCount() != 3 {
		t.Errorf("expected miss count to be 3, got %d", st.MissCount())
	}
}

func TestStats_HitCount(t *testing.T) {
	st := &stats{}
	
	if st.HitCount() != 0 {
		t.Errorf("expected initial hit count to be 0, got %d", st.HitCount())
	}
	
	st.IncrHitCount()
	st.IncrHitCount()
	
	if st.HitCount() != 2 {
		t.Errorf("expected hit count to be 2, got %d", st.HitCount())
	}
}

func TestStats_MissCount(t *testing.T) {
	st := &stats{}
	
	if st.MissCount() != 0 {
		t.Errorf("expected initial miss count to be 0, got %d", st.MissCount())
	}
	
	st.IncrMissCount()
	st.IncrMissCount()
	
	if st.MissCount() != 2 {
		t.Errorf("expected miss count to be 2, got %d", st.MissCount())
	}
}

func TestStats_LookupCount(t *testing.T) {
	st := &stats{}
	
	if st.LookupCount() != 0 {
		t.Errorf("expected initial lookup count to be 0, got %d", st.LookupCount())
	}
	
	st.IncrHitCount()
	st.IncrHitCount()
	st.IncrMissCount()
	
	if st.LookupCount() != 3 {
		t.Errorf("expected lookup count to be 3, got %d", st.LookupCount())
	}
}

func TestStats_HitRate(t *testing.T) {
	tests := []struct {
		name        string
		hitCount    uint64
		missCount   uint64
		expected    float64
		description string
	}{
		{
			name:        "no lookups",
			hitCount:    0,
			missCount:   0,
			expected:    0.0,
			description: "should return 0.0 when no lookups",
		},
		{
			name:        "all hits",
			hitCount:    10,
			missCount:   0,
			expected:    1.0,
			description: "should return 1.0 when all hits",
		},
		{
			name:        "all misses",
			hitCount:    0,
			missCount:   10,
			expected:    0.0,
			description: "should return 0.0 when all misses",
		},
		{
			name:        "half hits",
			hitCount:    5,
			missCount:   5,
			expected:    0.5,
			description: "should return 0.5 when half hits",
		},
		{
			name:        "one third hits",
			hitCount:    1,
			missCount:   2,
			expected:    1.0 / 3.0,
			description: "should return 1/3 when one third hits",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &stats{}
			for i := uint64(0); i < tt.hitCount; i++ {
				st.IncrHitCount()
			}
			for i := uint64(0); i < tt.missCount; i++ {
				st.IncrMissCount()
			}
			
			rate := st.HitRate()
			if rate != tt.expected {
				t.Errorf("%s: expected %f, got %f", tt.description, tt.expected, rate)
			}
		})
	}
}

func TestStats_ConcurrentAccess(t *testing.T) {
	st := &stats{}
	
	var wg sync.WaitGroup
	numGoroutines := 10
	iterationsPerGoroutine := 100
	
	// Concurrent increments
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				if j%2 == 0 {
					st.IncrHitCount()
				} else {
					st.IncrMissCount()
				}
			}
		}()
	}
	
	wg.Wait()
	
	expectedHits := uint64(numGoroutines * iterationsPerGoroutine / 2)
	expectedMisses := uint64(numGoroutines * iterationsPerGoroutine / 2)
	
	if st.HitCount() != expectedHits {
		t.Errorf("expected hit count to be %d, got %d", expectedHits, st.HitCount())
	}
	if st.MissCount() != expectedMisses {
		t.Errorf("expected miss count to be %d, got %d", expectedMisses, st.MissCount())
	}
	if st.LookupCount() != expectedHits+expectedMisses {
		t.Errorf("expected lookup count to be %d, got %d", expectedHits+expectedMisses, st.LookupCount())
	}
}

