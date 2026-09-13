package trie

import (
	"slices"
	"testing"
)

func TestNewTrie(t *testing.T) {
	trie := NewTrie()

	if trie == nil {
		t.Fatal("Expected NewTrie() to return a non-nil Trie instance")
	}
	if trie.RootNode == nil {
		t.Fatal("Expected RootNode to be initialized, got nil")
	}
	if trie.RootNode.Char != -1 {
		t.Errorf("Expected root node character to be '-1', got %q", trie.RootNode.Char)
	}
}

func TestInsert(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Insert single character string",
			input:    "-",
			expected: []string{"-"},
		},
		{
			name:     "Insert single lowercase word",
			input:    "cat",
			expected: []string{"c", "a", "t"},
		},
		{
			name:     "Insert uppercase word with spaces",
			input:    "L33t",
			expected: []string{"L", "3", "3", "t"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := NewTrie()
			err := trie.Insert(tt.input)

			if err != nil {
				t.Fatalf("Unexpected error during Insert(%q): %v", tt.input, err)
			}

			current := trie.RootNode
			for _, char := range tt.expected {
				if current.Children[rune(char[0])] == nil {
					t.Fatalf("Expected child node for character %q, but found nil", char)
				}
				current = current.Children[rune(char[0])]
				if current.Char != rune(char[0]) {
					t.Errorf("Expected node character %q, got %q", char, current.Char)
				}
			}
		})
	}
}

func TestSearchWord(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Search full match",
			input:    "doggo",
			expected: true,
		},
		{
			name:     "Search partial match",
			input:    "dog",
			expected: false,
		},
		{
			name:     "Search invalid match",
			input:    "cat",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := NewTrie()
			err := trie.Insert("doggo")

			if err != nil {
				t.Fatalf("Unexpected error during Insert(\"doggo\"): %v", err)
			}

			got, err := trie.SearchWord(tt.input)
			want := tt.expected

			if err != nil {
				t.Fatalf("Unexpected error during Search(%q): %v", tt.input, err)
			}

			if got != want {
				t.Errorf("got %t want %t", got, want)
			}

		})
	}
}

func TestSearchPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Search full match",
			input:    "boogie-woogie.",
			expected: true,
		},
		{
			name:     "Search partial match",
			input:    "boogie",
			expected: true,
		},
		{
			name:     "Search invalid match",
			input:    "woogie",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := NewTrie()
			err := trie.Insert("boogie-woogie.")

			if err != nil {
				t.Fatalf("Unexpected error during Insert(\"doggo\"): %v", err)
			}

			got, err := trie.SearchPrefix(tt.input)
			want := tt.expected

			if err != nil {
				t.Fatalf("Unexpected error during Search(%q): %v", tt.input, err)
			}

			if got != want {
				t.Errorf("got %t want %t", got, want)
			}

		})
	}
}

func TestFindMatches(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Search full match",
			input:    "boogie-woogie",
			expected: []string{"boogie-woogie"},
		},
		{
			name:     "Search partial match",
			input:    "boogie",
			expected: []string{"boogie-woogie", "boogiepop"},
		},
		{
			name:     "Search invalid match",
			input:    "rocketship",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := NewTrie()
			for _, word := range []string{"boogie-woogie", "boogiepop", "boolean", "woogie"} {
				err := trie.Insert(word)

				if err != nil {
					t.Fatalf("Unexpected error during Insert(\"%v\"): %v", word, err)
				}
			}

			got, err := trie.FindMatches(tt.input)
			want := tt.expected

			if err != nil {
				t.Fatalf("Unexpected error during Search(%q): %v", tt.input, err)
			}

			if !hasSameElements(got, want) {
				t.Errorf("got %v want %v", got, want)
			}

		})
	}
}

func hasSameElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aCopy := slices.Clone(a)
	bCopy := slices.Clone(b)

	slices.Sort(aCopy)
	slices.Sort(bCopy)

	return slices.Equal(aCopy, bCopy)
}
