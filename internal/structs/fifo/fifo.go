package fifo

import (
    "os"
    "sync"
)

type Fifo struct {
    path  string
    file  *os.File
    owner bool      // true = send, false = receive
    once  sync.Once // Close peut être appelé plusieurs fois sans erreur
}
