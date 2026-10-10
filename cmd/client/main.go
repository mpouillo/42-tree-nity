package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
)

func main() {
	os.Exit(run())
}

func responseFifoPath(ipc, cmdName string, cli *CLI) string {
	return fmt.Sprintf("%s.%d", ipc, os.Getpid())
}

func run() int {
	var cli CLI
	parser, err := kong.New(&cli, kong.NoDefaultHelp())
	if err != nil {
		panic(err)
	}
	parsed, err := parser.Parse(os.Args[1:])
	if err != nil {
		log.Errorf("%v", err)
		return int(response.GeneralError)
	}

	cmdName := parsed.Selected().Name
	cmd, ok := commandList[cmdName]
	if !ok { // just in case but should not happen
		log.Errorf("unknown command %s", cmdName)
		return int(response.GeneralError)
	}
	if errCode := cli.validate(cmdName); errCode != int(response.NoError) {
		return errCode
	}

	ipc := cli.IPC.IPC
	serverEndpoint, errCode := connectToEndpoint(ipc)
	if errCode != int(response.NoError) {
		return errCode
	}
	defer func() { _ = serverEndpoint.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	responsePath := responseFifoPath(ipc, cmdName, &cli)

	responseFifo, errCode := CreateResponseFifo(responsePath)
	if errCode != int(response.NoError) {
		return errCode
	}
	defer func() { _ = responseFifo.Close() }()
	responseFifoReader, errCode := OpenResponseFifo(responsePath)
	if errCode != int(response.NoError) {
		return errCode
	}
	commandInfo := ClientCommandInfo{serverEndpoint: serverEndpoint, cli: &cli, context: ctx,
		responseFifo: responseFifo, responseFifoReader: responseFifoReader}
	defer func() { _ = responseFifoReader.Close() }()

	return cmd.run(commandInfo)
}
