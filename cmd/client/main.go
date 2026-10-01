package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
)

func main() {
	os.Exit(run())
}

func run() int {
	args := os.Args[1:]
	if len(args) < 2 {
		log.Errorf("usage: client <ipc> <command> [args]")
		return int(response.GeneralError)
	}
	ipc, cmdName, rest := args[0], args[1], args[2:]

	serverEndpoint, errCode := connectToEndpoint(ipc)
	if errCode != int(response.NoError) {
		return errCode
	}
	defer func() { _ = serverEndpoint.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	commandInfo := ClientCommandInfo{serverEndpoint: serverEndpoint, args: rest, context: ctx}

	cmd, ok := commandList[cmdName]
	if !ok {
		log.Errorf("unknown command %s", cmdName)
		return int(response.GeneralError)
	}
	if !checkNbArgs(cmd.nbArgs, commandInfo.args) {
		log.Errorf("wrong number of args for %s command (expected %d)", cmdName, cmd.nbArgs)
		return int(response.GeneralError)
	}

	var responseFifoName string
	if cmdName == "subscribe" {
		responseFifoName = fmt.Sprintf("%s", ipc)
	} else {
		responseFifoName = ipc
	}
	responseFifo, errCode := CreateResponseFifo(ipc)
	if errCode != int(response.NoError) {
		return errCode
	}
	defer func() { _ = responseFifo.Close() }()
	responseFifoReader, errCode := OpenResponseFifo(ipc)
	if errCode != int(response.NoError) {
		return errCode
	}
	defer func() { _ = responseFifoReader.Close() }()
	
	return cmd.run(commandInfo)
}
