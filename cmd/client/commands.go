package main

import (
	"context"

	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

type ClientCommandInfo struct {
	serverEndpoint *fifo.Fifo
	args           []string
	context        context.Context
}

type Command struct {
	nbArgs int
	run    func(ClientCommandInfo) int
}

var commandList = map[string]Command{
	"list": {
		nbArgs: 0,
		run:    cmdList,
	},
	"info": {
		nbArgs: 1,
		run:    cmdInfo,
	},
	"create": {
		nbArgs: 1,
		run:    cmdCreate,
	},
	"produce": {
		nbArgs: 1,
		run:    cmdProduce,
	},
	"subscribe": {
		nbArgs: 2,
		run:    cmdSubscribe,
	},
}

func checkNbArgs(nbArgsNeeded int, args []string) bool {
	return len(args) == nbArgsNeeded
}

func cmdList(command ClientCommandInfo) int {
	return int(response.NoError)

}
func cmdInfo(command ClientCommandInfo) int {
	return int(response.NoError)

}
func cmdCreate(command ClientCommandInfo) int {
	return int(response.NoError)
}

func cmdProduce(command ClientCommandInfo) int {
	return int(response.NoError)

}

func cmdSubscribe(command ClientCommandInfo) int {
	return int(response.NoError)

}
