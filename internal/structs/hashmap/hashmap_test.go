package hashmap

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewHashMap(t *testing.T) {
	hm := NewHashMap()

	if hm.nb_inserted != 0 {
		t.Fatalf("expected nb_inserted = 0, got %d", hm.nb_inserted)
	}

	if cap(hm.elements) != 16 {
		t.Fatalf("expected cap = 16, got %d", hm.cap())
	}
}

func TestCap(t *testing.T) {
	hm := NewHashMap()
	if int(hm.cap()) != cap(hm.elements) {
		t.Fatalf("cap is not equal to cap(hm.elements), expected %d, got %d", cap(hm.elements), hm.cap())
	}
}

func TestResize(t *testing.T) {
	t.Run("check capacity", func(t *testing.T) {
		hm := NewHashMap()

		cap := hm.cap()
		hm.resize()
		if cap*2 != hm.cap() {
			t.Fatalf("resize did not double capacity, expected %d, got %d", cap*2, hm.cap())
		}
		if hm.nb_inserted != 0 {
			t.Fatalf("expected nb_inserted = 2, got %d", hm.nb_inserted)
		}
	})

	t.Run("check good index", func(t *testing.T) {
		hm := NewHashMap()

		key := "hello"
		value := uint64(1)
		index := hash(key, hm.cap())
		hm.elements[index].append(key, value)
		hm.resize()

		new_index := hash(key, hm.cap())
		if hm.elements[new_index].entries[0].key != key || hm.elements[new_index].entries[0].value != value {
			t.Fatalf("expected key 'hello' in bucket %d, got %s", new_index, hm.elements[new_index].entries[0].key)
		}
	})

	t.Run("check bad index", func(t *testing.T) {
		hm := NewHashMap()

		key := "hello"
		index := hash(key, hm.cap())
		hm.elements[index].append(key, 1)
		hm.resize()

		new_index := hash(key+"aijdwiawdipajwidp", hm.cap())
		if len(hm.elements[new_index].entries) != 0 {
			t.Fatalf("expected different key %d, got %s", new_index, hm.elements[new_index].entries[0].key)
		}

	})
}

func TestInsert(t *testing.T) {
	t.Run("check basic insert", func(t *testing.T) {
		hm := NewHashMap()

		hm.Insert("hello", 1)
		index := hash("hello", hm.cap())
		key_got := hm.elements[index].entries[0].key
		value_got := hm.elements[index].entries[0].value
		if key_got != "hello" || value_got != 1 {
			t.Fatalf("expected key 'hello' with value 1 in bucket %d, got <%s>:<%d>", index, key_got, value_got)
		}
	})
	t.Run("check insert with same key", func(t *testing.T) {
		hm := NewHashMap()
		hm.Insert("hello", 1)
		hm.Insert("hello", 2)
		index := hash("hello", hm.cap())
		key_got := hm.elements[index].entries[0].key
		value_got := hm.elements[index].entries[0].value
		if key_got != "hello" || value_got != 2 {
			t.Fatalf("expected key 'hello' with value 2 in bucket %d, got <%s>:<%d>", index, key_got, value_got)
		}
	})
	t.Run("check double capacity", func(t *testing.T) {
		hm := NewHashMap()
		cap := hm.cap()
		for i := 0; i < 12; i++ {
			hm.Insert(fmt.Sprintf("key_%d", i), uint64(i))
		}
		if hm.cap() != cap*2 {
			t.Fatalf("expected double cap after 13 insertions, got %d", hm.cap())
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("check basic get", func(t *testing.T) {
		hm := NewHashMap()
		hm.Insert("hello", 1)
		val, err := hm.Get("hello")
		if err != nil {
			t.Fatalf("expected no error for key 'hello', got error: %v", err)
		}
		if val != 1 {
			t.Fatalf("expected value 1 for key 'hello', got %d", val)
		}
	})
	t.Run("check get not found", func(t *testing.T) {
		hm := NewHashMap()
		val, err := hm.Get("hello")
		if err == nil {
			t.Fatalf("expected error for missing key in empty map, got val=%d", val)
		}
	})
	t.Run("check good error", func(t *testing.T) {
		hm := NewHashMap()
		_, err := hm.Get("hello")
		if !errors.Is(err, errKeyNotFound) {
			t.Fatalf("expected error 'key not found', got '%v'", err)
		}
	})
}
