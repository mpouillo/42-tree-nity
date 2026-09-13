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
