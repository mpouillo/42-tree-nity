package main

import (
	"context"
	"encoding/json"
	"fmt"
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

func dialogWithServer(commandInfo ClientCommandInfo, cmd uint8, request any) (*response.Response, int) {
	if errCode := SendCommand(&commandInfo, cmd, request); errCode != int(response.NoError) {
		return nil, errCode
	}

	resp, errCode := GetResponse(commandInfo)
	if resp == nil {
		return nil, errCode
	}

	return resp, int(response.NoError)
}

func unmarshallResponse[T any](resp *response.Response) (T, int) {
	var data T
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		log.Errorf("failed to unmarshal: %v", err)
		return *new(T), int(response.GeneralError)
	}
	return data, int(response.NoError)
}

func cmdList(commandInfo ClientCommandInfo) int {
	request := commandInfo.cli.IPC.List
	request.IPCPath = commandInfo.responseFifoReader.Path()

	resp, errCode := dialogWithServer(commandInfo, commands.CmdListTopics, request)
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

	resp, errCode := dialogWithServer(commandInfo, commands.CmdInfoClient, request)
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

	resp, errCode := dialogWithServer(commandInfo, commands.CmdCreateTopic, request)
	if resp == nil {
		return errCode
	}

	return int(response.NoError)
}

func cmdProduce(commandInfo ClientCommandInfo) int {
	request := commandInfo.cli.IPC.Produce
	request.IPCPath = commandInfo.responseFifoReader.Path()
	_ = request
	return int(response.NoError)
}

func cmdSubscribe(commandInfo ClientCommandInfo) int {
	request := commandInfo.cli.IPC.Subscribe
	request.IPCPath = commandInfo.responseFifoReader.Path()
	_ = request
	_ = request.Raw
	return int(response.NoError)
}
