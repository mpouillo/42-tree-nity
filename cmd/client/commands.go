package main

import (
	"context"

	"github.com/ayberkgezer/gocolorlog"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

type ClientCommand struct {
	serverEndpoint *fifo.Fifo
	args           []string
	context        context.Context
}

func checkNbArgs(max int, args []string) {
	if len(args) != max {
		gocolorlog.Errorf("too much args for create command\n")
	}
}

func cmdCreate(command ClientCommand) {
	checkNbArgs(1, command.args)

}

func cmdList(command ClientCommand) {
	checkNbArgs(0, command.args)
}

func cmdProduce(command ClientCommand) {
	checkNbArgs(1, command.args)
}

func cmdSubscribe(command ClientCommand) {
	checkNbArgs(2, command.args)
}

func cmdInfo(command ClientCommand) {
	checkNbArgs(1, command.args)
}
