package main

import (
	"os"
	"fmt"
	"github.com/ayberkgezer/gocolorlog"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
)

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: client <ipc> <command> [args]")
		os.Exit(int(response.GeneralError))
	}
	ipc, cmd, rest := args[0], args[1], args[2:]
	command := ClientCommand{ipc: ipc, args: rest}
	switch cmd {
	case "create":
		cmdCreate(command)
	case "list":
		cmdList(command)
	case "produce":
		cmdProduce(command)
	case "subscribe":
		cmdSubscribe(command)
	case "info":
		cmdInfo(command)
	default:
		gocolorlog.Errorf("unknown command %q\n", cmd)
		os.Exit(int(response.GeneralError))
	}
}