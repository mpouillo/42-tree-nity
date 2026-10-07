package main

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/packet"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

func connectToEndpoint(ipc string) (*fifo.Fifo, int) {
	f, err := fifo.Open(ipc, fifo.Write)
	if err != nil {
		log.Errorf("failed to connect to endpoint: %v", err)
		return nil, int(response.GeneralError)
	}
	return f, int(response.NoError)
}

func CreateResponseFifo(ipc string) (*fifo.Fifo, int) {
	f, err := fifo.Create(ipc, fifo.ReadWrite)
	if err != nil {
		log.Errorf("failed to create response fifo: %v", err)
		return nil, int(response.GeneralError)
	}
	return f, int(response.NoError)
}

func OpenResponseFifo(ipc string) (*fifo.Fifo, int) {
	f, err := fifo.Open(ipc, fifo.Read)
	if err != nil {
		log.Errorf("failed to open response fifo: %v", err)
		return nil, int(response.GeneralError)
	}
	return f, int(response.NoError)
}

func SendCommand(commandInfo *ClientCommandInfo, cmd uint8, request any) int {
	payload, err := json.Marshal(request)
	if err != nil {
		return int(response.GeneralError)
	}
	if err := packet.WritePacket(commandInfo.serverEndpoint, cmd, payload); err != nil {
		return int(response.IpcError)
	}
	return int(response.NoError)
}

func ReadResponse(ctx context.Context, fifo *fifo.Fifo) (*response.Response, int) {
	pkt, err := packet.ReadPacket(ctx, fifo)
	if errors.Is(err, context.Canceled) {
		return nil, int(response.NoError)
	}
	if err != nil {
		log.Errorf("read response: %v", err)
		return nil, int(response.IpcError)
	}

	var resp response.Response
	if err := json.Unmarshal(pkt.Payload, &resp); err != nil {
		log.Errorf("decode response: %v", err)
		return nil, int(response.IpcError)
	}

	if resp.Code != response.NoError {
		log.Errorf("%s", resp.ErrorMsg)
		return nil, int(resp.Code)
	}
	return &resp, int(response.NoError)
}

func GetResponse(c ClientCommandInfo, stopNewConnectionsAfter bool) (*response.Response, int) {
	resp, code := ReadResponse(c.context, c.responseFifoReader)
	if stopNewConnectionsAfter {
		_ = c.responseFifo.Close()
	}
	return resp, code
}


func dialogWithServer(commandInfo ClientCommandInfo, cmd uint8, request any, stopNewConnectionsAfter bool) (*response.Response, int) {
	if errCode := SendCommand(&commandInfo, cmd, request); errCode != int(response.NoError) {
		return nil, errCode
	}

	resp, errCode := GetResponse(commandInfo, stopNewConnectionsAfter)
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
