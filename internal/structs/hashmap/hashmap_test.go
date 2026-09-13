package hashmap

import (
	"fmt"
	"testing"
)

func TestInsert(t *testing.T) {
	hm := NewHashMap()

	if hm.nb_inserted != 0 {
		t.Fatalf("expected nb_inserted = 0, got %d", hm.nb_inserted)
	}

	hm.Insert("hello", 42)

	if hm.nb_inserted != 1 {
		t.Fatalf("expected nb_inserted = 1, got %d", hm.nb_inserted)
	}

	idx := hash("hello", hm.cap())
	found := false
	for _, entry := range hm.elements[idx].entries {
		if entry.key == "hello" && entry.value == 42 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("key 'hello' with value 42 not found in bucket %d", idx)
	}

	hm.Insert("world", 100)

	if hm.nb_inserted != 2 {
		t.Fatalf("expected nb_inserted = 2, got %d", hm.nb_inserted)
	}

	idx = hash("world", hm.cap())
	found = false
	for _, entry := range hm.elements[idx].entries {
		if entry.key == "world" && entry.value == 100 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("key 'world' with value 100 not found in bucket %d", idx)
	}

	// Override existing key "hello" with a new value
	hm.Insert("hello", 84)

	idx = hash("hello", hm.cap())
	found = false
	count := 0
	for _, entry := range hm.elements[idx].entries {
		if entry.key == "hello" {
			count++
			if entry.value == 84 {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("key 'hello' with overridden value 84 not found in bucket %d", idx)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 entry for key 'hello', got %d", count)
	}
}

func TestResize(t *testing.T) {
	hm := NewHashMap()

	hm.Insert("key1", 10)
	hm.Insert("key2", 20)

	initialCap := hm.cap()
	if initialCap != 16 {
		t.Fatalf("expected initial cap = 16, got %d", initialCap)
	}

	hm.resize()

	// Check that capacity doubled
	if hm.cap() != 32 {
		t.Fatalf("expected cap = 32 after resize, got %d", hm.cap())
	}

	// Check that inserted count is preserved
	if hm.nb_inserted != 2 {
		t.Errorf("expected nb_inserted = 2, got %d", hm.nb_inserted)
	}

	// Verify entries are still present at their new hashed index
	idx1 := hash("key1", hm.cap())
	found1 := false
	for _, entry := range hm.elements[idx1].entries {
		if entry.key == "key1" && entry.value == 10 {
			found1 = true
			break
		}
	}
	if !found1 {
		t.Errorf("key 'key1' not found after resize")
	}

	idx2 := hash("key2", hm.cap())
	found2 := false
	for _, entry := range hm.elements[idx2].entries {
		if entry.key == "key2" && entry.value == 20 {
			found2 = true
			break
		}
	}
	if !found2 {
		t.Errorf("key 'key2' not found after resize")
	}
}

func TestInsertTriggersResize(t *testing.T) {
	hm := NewHashMap()

	// With capacity 16 and load factor 0.75, the resize threshold is 12 (16 * 0.75)
	// Inserting 11 elements: capacity should remain 16
	for i := 1; i <= 11; i++ {
		hm.Insert(fmt.Sprintf("key_%d", i), uint64(i))
	}

	if hm.cap() != 16 {
		t.Errorf("expected cap = 16 before threshold, got %d", hm.cap())
	}

	// The 12th element reaches the threshold and triggers automatic resize
	hm.Insert("key_12", 12)

	if hm.cap() != 32 {
		t.Errorf("expected cap = 32 after 12 insertions, got %d", hm.cap())
	}

	// Verify the 12th element is accessible in the resized map
	idx := hash("key_12", hm.cap())
	found := false
	for _, entry := range hm.elements[idx].entries {
		if entry.key == "key_12" && entry.value == 12 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("key 'key_12' not found in hashmap after automatic resize")
	}
}

func TestGet(t *testing.T) {
	hm := NewHashMap()

	hm.Insert("alpha", 10)
	hm.Insert("beta", 20)
	hm.Insert("gamma", 0) // Testing 0 value edge-case

	tests := []struct {
		key      string
		expected uint64
	}{
		{"alpha", 10},
		{"beta", 20},
		{"gamma", 0},
	}

	for _, tt := range tests {
		val, err := hm.Get(tt.key)
		if err != nil {
			t.Errorf("expected no error for key '%s', got error: %v", tt.key, err)
		}
		if val != tt.expected {
			t.Errorf("for key '%s', expected %d, got %d", tt.key, tt.expected, val)
		}
	}
}

func TestGetNotFound(t *testing.T) {
	hm := NewHashMap()

	// Empty map
	val, err := hm.Get("missing")
	if err == nil {
		t.Errorf("expected error for missing key in empty map, got val=%d", val)
	}

	// Non-empty map with missing key
	hm.Insert("existing", 42)
	val, err = hm.Get("missing")
	if err == nil {
		t.Errorf("expected error for non-existent key, got val=%d", val)
	}
	if err != nil && err.Error() != "key not found" {
		t.Errorf("expected error 'key not found', got '%v'", err)
	}
}

func TestGetUpdatedValue(t *testing.T) {
	hm := NewHashMap()

	hm.Insert("key", 100)
	val, err := hm.Get("key")
	if err != nil || val != 100 {
		t.Fatalf("expected 100, got %d (err: %v)", val, err)
	}

	// Update value
	hm.Insert("key", 200)
	val, err = hm.Get("key")
	if err != nil {
		t.Fatalf("unexpected error after update: %v", err)
	}
	if val != 200 {
		t.Errorf("expected updated value 200, got %d", val)
	}
}

func TestGetCollision(t *testing.T) {
	hm := NewHashMap()

	// Find two keys that collide on the same bucket
	cap := hm.cap()
	var key1, key2 string
	foundCollision := false

	for i := 0; i < 100; i++ {
		candidate := fmt.Sprintf("collision_candidate_%d", i)
		idx := hash(candidate, cap)
		for j := i + 1; j < 100; j++ {
			candidate2 := fmt.Sprintf("collision_candidate_%d", j)
			if hash(candidate2, cap) == idx {
				key1 = candidate
				key2 = candidate2
				foundCollision = true
				break
			}
		}
		if foundCollision {
			break
		}
	}

	if !foundCollision {
		t.Skip("could not find colliding keys for test")
	}

	hm.Insert(key1, 111)
	hm.Insert(key2, 222)

	val1, err1 := hm.Get(key1)
	if err1 != nil || val1 != 111 {
		t.Errorf("failed to retrieve key1 (%s): val=%d, err=%v", key1, val1, err1)
	}

	val2, err2 := hm.Get(key2)
	if err2 != nil || val2 != 222 {
		t.Errorf("failed to retrieve key2 (%s): val=%d, err=%v", key2, val2, err2)
	}
}

func TestGetAfterResize(t *testing.T) {
	hm := NewHashMap()

	const count = 50
	for i := 0; i < count; i++ {
		hm.Insert(fmt.Sprintf("item_%d", i), uint64(i*10))
	}

	// Ensure resize occurred
	if hm.cap() <= base_hashmap_size {
		t.Fatalf("expected hashmap to have resized, capacity is %d", hm.cap())
	}

	// Ensure all elements can still be retrieved
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("item_%d", i)
		val, err := hm.Get(key)
		if err != nil {
			t.Errorf("expected key '%s' to be found after resize, got error: %v", key, err)
		}
		if val != uint64(i*10) {
			t.Errorf("for key '%s', expected %d, got %d", key, i*10, val)
		}
	}

	// Ensure non-existent key still returns not found
	_, err := hm.Get("does_not_exist")
	if err == nil {
		t.Errorf("expected error for non-existent key after resize, got nil")
	}
}

