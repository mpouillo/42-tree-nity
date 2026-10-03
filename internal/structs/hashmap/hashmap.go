package hashmap

type entry[T any] struct {
	key   string
	value T
}

type bucket[T any] struct {
	entries []entry[T]
}

func (bucket *bucket[T]) append(key string, value T) {
	bucket.entries = append(bucket.entries, entry[T]{key, value})
}

type Hashmap[T any] struct {
	nb_inserted uint64
	elements    []bucket[T]
}

func NewHashMap[T any]() *Hashmap[T] {
	return &Hashmap[T]{
		nb_inserted: 0,
		elements:    make([]bucket[T], base_hashmap_size),
	}
}

func (hashmap *Hashmap[T]) cap() uint64 {
	return uint64(cap(hashmap.elements))
}

func (hashmap *Hashmap[T]) resize() {
	new_size := hashmap.cap() * 2
	new_elements := make([]bucket[T], new_size)
	for _, bucket := range hashmap.elements {
		for _, entry := range bucket.entries {
			new_index := hash(entry.key, new_size)
			new_elements[new_index].append(entry.key, entry.value)
		}
	}
	hashmap.elements = new_elements
}

func (hashmap *Hashmap[T]) Insert(key string, value T) {
	var index = hash(key, hashmap.cap())
	for i := range hashmap.elements[index].entries {
		if hashmap.elements[index].entries[i].key == key {
			hashmap.elements[index].entries[i].value = value
			return
		}
	}

	// only a new key grows the map, so resize after the lookup
	inserted := hashmap.nb_inserted + 1
	max_entries_before_resize := uint64(float32(hashmap.cap()) * filled_factor_before_resize)
	if inserted >= max_entries_before_resize {
		hashmap.resize()
		index = hash(key, hashmap.cap())
	}
	hashmap.nb_inserted++
	hashmap.elements[index].append(key, value)
}

func (hashmap *Hashmap[T]) Get(key string) (any, error) {
	index := hash(key, hashmap.cap())
	for _, entry := range hashmap.elements[index].entries {
		if entry.key == key {
			return entry.value, nil
		}
	}
	return nil, ErrKeyNotFound
}
