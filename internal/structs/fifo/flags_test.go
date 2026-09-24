package fifo

import (
	"os"
	"syscall"
	"testing"
)

func TestModeFlags(t *testing.T) {
	t.Run("check known modes", func(t *testing.T) {
		cases := []struct {
			name string
			mode Mode
			want int
		}{
			{"read", Read, os.O_RDONLY},
			{"write", Write, os.O_WRONLY | syscall.O_NONBLOCK},
			{"read write", ReadWrite, os.O_RDWR},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := tc.mode.flags()
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if got != tc.want {
					t.Fatalf("expected flags %#o, got %#o", tc.want, got)
				}
			})
		}
	})

	t.Run("check unknown mode", func(t *testing.T) {
		got, err := Mode(42).flags()
		if err == nil {
			t.Fatalf("expected an error, got none")
		}
		if got != 0 {
			t.Fatalf("expected flags 0, got %#o", got)
		}
	})
}
