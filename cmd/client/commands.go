package main

import (
	"context"

	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"

	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

type ClientCommandInfo struct {
	serverEndpoint *fifo.Fifo
	cli            *CLI
	context        context.Context
	responseFifoPath string
}

type Command struct {
	run    func(ClientCommandInfo) int
}

var commandList = map[string]Command{
	"list":      {run: cmdList},
	"info":      {run: cmdInfo},
	"create":    {run: cmdCreate},	
	"produce":   {run: cmdProduce},
	"subscribe": {run: cmdSubscribe},
}



func cmdList(clientCommand ClientCommandInfo) int {
	return int(response.NoError)
}
func cmdInfo(clientCommand ClientCommandInfo) int {
	return int(response.NoError)

}
func cmdCreate(clientCommand ClientCommandInfo) int {
	return int(response.NoError)
}

func cmdProduce(clientCommand ClientCommandInfo) int {
	return int(response.NoError)

}

func cmdSubscribe(clientCommand ClientCommandInfo) int {
	return int(response.NoError)

}
