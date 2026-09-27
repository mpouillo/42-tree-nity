package main

import (
	"github.com/ayberkgezer/gocolorlog"
)

type ClientCommand struct {
	ipc  string
	args []string
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
