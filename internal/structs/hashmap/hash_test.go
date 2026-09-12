package hashmap

import "testing"

func TestHash(t *testing.T) {
    tests := []string{
        "salutations",
        "hello",
        "world",
        "go",
        "golang",
        "hashmap",
        "hash",
        "map",
        "struct",
        "structs",
        "internal",
        "test",
        "testing",
        "aleatoire",
        "mot",
        "des",
        "tu",
        "perlinpinpin",
    }

    sizes := []uint64{10, 100, 1000, 10000}

    for _, test := range tests {
        for _, size := range sizes {
            h1 := hash(test, size)
            h2 := hash(test, size)

            if h1 != h2 {
                t.Errorf("hash(%q, %d) is not deterministic", test, size)
            }

            if h1 >= size {
                t.Errorf("hash(%q, %d) = %d, want < %d",
                    test, size, h1, size)
            }
        }
    }
}