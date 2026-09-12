package hashmap


func hash(str string, size uint64) uint64 {
	if size == 0 {
		panic("hash: size must be greater than 0")
	}

	h := offset64

	for i := 0; i < len(str); i++ {
		h ^= uint64(str[i])
		h *= prime64
	}

	return h % size
}
