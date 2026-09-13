package hashmap

import "errors"

type entry struct {
	key   string
	value uint64
}

type bucket struct {
	entries []entry
}

func (bucket *bucket) append(key string, value uint64) {
	bucket.entries = append(bucket.entries, entry{key, value})
}

type Hashmap struct {
	nb_inserted uint64
	elements    []bucket
}

func NewHashMap() Hashmap {
	return Hashmap{
		nb_inserted: 0,
		elements:    make([]bucket, base_hashmap_size),
	}
}

func (hashmap *Hashmap) cap() uint64 {
	return uint64(cap(hashmap.elements))
}

func (hashmap *Hashmap) resize() {
	new_size := hashmap.cap() * 2
	new_elements := make([]bucket, new_size)
	for _, bucket := range hashmap.elements {
		for _, entry := range bucket.entries {
			new_index := hash(entry.key, new_size)
			new_elements[new_index].append(entry.key, entry.value)
		}
	}
    hashmap.elements = new_elements
}

func (hashmap *Hashmap) Insert(key string, value uint64) {
    
	inserted := hashmap.nb_inserted + 1
	max_entries_before_resize := uint64(float32(hashmap.cap()) * filled_factor_before_resize)
	if inserted >= max_entries_before_resize {
		hashmap.resize()
    }
	var index uint64 = hash(key, hashmap.cap())
    for i := range hashmap.elements[index].entries {
		if hashmap.elements[index].entries[i].key == key {
			hashmap.elements[index].entries[i].value = value
            return
        }
    }
	hashmap.nb_inserted++
	hashmap.elements[index].append(key, value)
}

func (hashmap *Hashmap) Get(key string) (uint64, error) {
	index := hash(key, hashmap.cap())
	for _, entry := range hashmap.elements[index].entries {
		if entry.key == key {
			return entry.value, nil
		}
	}
	return 0, errors.New("key not found")
}