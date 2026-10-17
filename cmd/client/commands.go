package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	commands "github.com/mpouillo/42-tree-nity/internal/ipc/protocol/commands"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/packet"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

type ClientCommandInfo struct {
	serverEndpoint     *fifo.Fifo
	cli                *CLI
	context            context.Context
	responseFifo       *fifo.Fifo
	responseFifoReader *fifo.Fifo
	responseReader     *bufio.Reader
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

// cmdList prints the topics of the server as a comma-separated list.
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

// cmdInfo prints the metadata of a registered client as a JSON object.
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

// cmdCreate creates a topic and prints a confirmation.
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

// cmdProduce checks that the topic exists, then sends every message read from
// stdin to it until EOF or a signal.
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

// cmdSubscribe registers the consumer through a PID FIFO handshake, then prints
// and acknowledges each message received on the consumer FIFO.
func cmdSubscribe(commandInfo ClientCommandInfo) int {
	request := commandInfo.cli.IPC.Subscribe
	consumerFifo := commandInfo.responseFifoReader
	request.ConsumerPath = consumerFifo.Path()
	request.PID = os.Getpid()

	pidFifo, errCode := CreateResponseFifo(pidFifoPath(commandInfo.cli.IPC.IPC))
	if errCode != int(response.NoError) {
		return errCode
	}
	defer func() { _ = pidFifo.Close() }()
	request.IPCPath = pidFifo.Path()
	handshake := commandInfo
	handshake.responseFifo = pidFifo
	handshake.responseFifoReader = pidFifo
	handshake.responseReader = packet.NewReader(pidFifo)
	resp, errCode := dialogWithServer(handshake, commands.CmdSubscribe, request, true)
	if resp == nil {
		return errCode
	}
	fmt.Printf("subscribed to %s\n", request.Topic)

	_ = commandInfo.responseFifo.Close()

	for {
		cmd, resp, errCode := ReadCommandResponse(commandInfo.context, commandInfo.responseReader)
		if resp == nil {
			if errCode == int(response.NoError){
				SendCommand(&commandInfo, commands.CmdDisconnect, commands.Disconnect{IPCPath: request.ConsumerPath, Client: request.Client})
			}
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
			msg := data.Message

			if request.Raw {
				_, _ = os.Stdout.Write(msg.ToRaw())
			} else {
				fmt.Printf("%s\n", msg.ToSeparator(":"))
			}

			ack := commands.AckOffset{IPCPath: request.ConsumerPath, Client: request.Client, Offset: msg.Offset + 1}
			if errCode := SendCommand(&commandInfo, commands.CmdAckOffset, ack); errCode != int(response.NoError) {
				return errCode
			}
		default:
			log.Errorf("unexpected command %d on consumer fifo", cmd)
			return int(response.IpcError)
		}
	}
}
