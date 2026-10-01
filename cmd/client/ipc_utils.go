package main

import (
	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)


func connectToEndpoint(ipc string) (*fifo.Fifo, int) {
	f, err := fifo.Open(ipc, fifo.Write)
	if err != nil {
		log.Errorf("failed to connect to endpoint: %v", err)
		return nil, int(response.GeneralError)
	}
	return f, int(response.NoError)
}

func CreateResponseFifo(ipc string) (*fifo.Fifo, int) {
	f, err := fifo.Create(ipc, fifo.ReadWrite)
	if err != nil {
		log.Errorf("failed to create response fifo: %v", err)
		return nil, int(response.GeneralError)
	}
	return f, int(response.NoError)
}

func OpenResponseFifo(ipc string) (*fifo.Fifo, int) {
	f, err := fifo.Open(ipc, fifo.Read)
	if err != nil {
		log.Errorf("failed to open response fifo: %v", err)
		return nil, int(response.GeneralError)
	}
	return f, int(response.NoError)
}