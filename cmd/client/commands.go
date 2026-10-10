package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	commands "github.com/mpouillo/42-tree-nity/internal/ipc/protocol/commands"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

type ClientCommandInfo struct {
	serverEndpoint     *fifo.Fifo
	cli                *CLI
	context            context.Context
	responseFifo       *fifo.Fifo
	responseFifoReader *fifo.Fifo
}

type Command struct {
	run func(ClientCommandInfo) int
}

var commandList = map[string]Command{
	"list":      {run: cmdList},
	"info":      {run: cmdInfo},
	"create":    {run: cmdCreate},
	"produce":   {run: cmdProduce},
	"subscribe": {run: cmdSubscribe},
}

func cmdList(commandInfo ClientCommandInfo) int {
	request := commandInfo.cli.IPC.List
	request.IPCPath = commandInfo.responseFifoReader.Path()

	resp, errCode := dialogWithServer(commandInfo, commands.CmdListTopics, request, true)
	if resp == nil {
		return errCode
	}

	data, errCode := unmarshallResponse[response.ListTopicsData](resp)
	if errCode != int(response.NoError) {
		return errCode
	}
	fmt.Println(strings.Join(data.Topics, ","))

	return int(response.NoError)
}

func cmdInfo(commandInfo ClientCommandInfo) int {
	request := commandInfo.cli.IPC.Info
	request.IPCPath = commandInfo.responseFifoReader.Path()

	resp, errCode := dialogWithServer(commandInfo, commands.CmdInfoClient, request, true)
	if resp == nil {
		return errCode
	}

	_, errCode = unmarshallResponse[response.InfoClientData](resp)
	if errCode != int(response.NoError) {
		return errCode
	}
	fmt.Println(string(resp.Data))

	return int(response.NoError)
}

func cmdCreate(commandInfo ClientCommandInfo) int {
	request := commandInfo.cli.IPC.Create
	request.IPCPath = commandInfo.responseFifoReader.Path()

	resp, errCode := dialogWithServer(commandInfo, commands.CmdCreateTopic, request, true)
	if resp == nil {
		return errCode
	}

	fmt.Println("topic created")

	return int(response.NoError)
}

func cmdProduce(commandInfo ClientCommandInfo) int {
	listRequest := commandInfo.cli.IPC.List
	request := commandInfo.cli.IPC.Produce
	listRequest.IPCPath = commandInfo.responseFifoReader.Path()
	request.IPCPath = commandInfo.responseFifoReader.Path()

	resp, errCode := dialogWithServer(commandInfo, commands.CmdListTopics, listRequest, false)
	if resp == nil {
		return errCode
	}

	data, errCode := unmarshallResponse[response.ListTopicsData](resp)
	if errCode != int(response.NoError) {
		return errCode
	}

	if !slices.Contains(data.Topics, request.Topic) {
		log.Errorf("topic %q does not exist", request.Topic)
		return int(response.TopicError)
	}

	messageChannel := readMessages(os.Stdin, request.Raw)
	for {
		select {
		case <-commandInfo.context.Done():
			return int(response.NoError)
		case message := <-messageChannel:
			if errors.Is(message.err, io.EOF) {
				return int(response.NoError)
			}
			if message.err != nil {
				log.Errorf("failed to read message: %v", message.err)
				return int(response.GeneralError)
			}

			request.Message = *message.msg
			resp, errCode := dialogWithServer(commandInfo, commands.CmdProduce, request, false)
			if resp == nil {
				return errCode
			}
		}
	}
}

func cmdSubscribe(commandInfo ClientCommandInfo) int {
	request := commandInfo.cli.IPC.Subscribe
	request.IPCPath = commandInfo.responseFifoReader.Path()
	request.ConsumerPath = commandInfo.cli.IPC.IPC + "." + request.Client

	consumerFifo, err := fifo.Create(request.ConsumerPath, fifo.ReadWrite)
	if err != nil {
		log.Errorf("failed to create consumer fifo: %v", err)
		return int(response.GeneralError)
	}
	defer func() { _ = consumerFifo.Close() }()
	resp, errCode := dialogWithServer(commandInfo, commands.CmdSubscribe, request, true)
	if resp == nil {
		return errCode
	}
	fmt.Printf("subscribed to %s\n", request.Topic)

	defer func() {
		_ = SendCommand(&commandInfo, commands.CmdDisconnect, commands.Disconnect{Client: request.Client})
	}()
	for {
		cmd, resp, errCode := ReadCommandResponse(commandInfo.context, consumerFifo)
		if resp == nil { // signal received (code 0), or error (refused ack included)
			return errCode
		}

		switch cmd {
		case commands.CmdDisconnect:
			return int(response.NoError)
		case commands.CmdAckOffset:
			continue
		case commands.CmdSubscribe:
			data, errCode := unmarshallResponse[response.SubscribeData](resp)
			if errCode != int(response.NoError) {
				return errCode
			}


		default:
			log.Errorf("unexpected command %d on consumer fifo", cmd)
			return int(response.IpcError)
		}
	}
}
