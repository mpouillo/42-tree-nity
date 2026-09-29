package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ayberkgezer/gocolorlog"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)


func connectToEndpoint(ipc string) *fifo.Fifo{
	f, err := fifo.Open(ipc, fifo.Write)
	if err != nil {
		gocolorlog.Errorf("failed to connect to endpoint: %v", err)
		os.Exit(int(response.GeneralError))
	}
	return f
}


func main() {
	os.Exit(run())
}

func run() int {
	args := os.Args[1:]
	if len(args) < 2 {
		gocolorlog.Errorf("usage: client <ipc> <command> [args]")
		return int(response.GeneralError)
	}
	ipc, cmdName, rest := args[0], args[1], args[2:]

	serverEndpoint := connectToEndpoint(ipc)
	defer serverEndpoint.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	
	commandInfo := ClientCommandInfo{serverEndpoint: serverEndpoint, args: rest, context: ctx}

	cmd, ok := commandList[cmdName]
	if !ok{
		gocolorlog.Errorf("unknown command %s\n", cmdName)
		return int(response.GeneralError)
	}
	if !checkNbArgs(cmd.nbArgs, commandInfo.args){
		gocolorlog.Errorf("wrong number of args for %s command (expected %d)", cmdName, cmd.nbArgs)
		return int(response.GeneralError)
	}
	return cmd.run(commandInfo)
}