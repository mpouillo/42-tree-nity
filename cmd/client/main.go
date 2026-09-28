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
	args := os.Args[1:]
	if len(args) < 2 {
		gocolorlog.Errorf("usage: client <ipc> <command> [args]")
		os.Exit(int(response.GeneralError))
	}
	ipc, cmd, rest := args[0], args[1], args[2:]

	serverEndpoint := connectToEndpoint(ipc)
	defer serverEndpoint.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	
	command := ClientCommand{serverEndpoint: serverEndpoint, args: rest, context: ctx}


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