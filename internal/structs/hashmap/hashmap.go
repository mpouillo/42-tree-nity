
type entry struct {
    key   string
    value uint64
}

type bucket struct {
    entries []entry
}

type Hashmap struct {
    nb_entries uint64
    elements []bucket
}
