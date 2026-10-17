package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/commands"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/packet"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
	"github.com/mpouillo/42-tree-nity/internal/structs/fifo"
)

// responseTimeout is how long the client waits for the server to answer a request.
const responseTimeout = 3 * time.Second

// connectToEndpoint opens the main server FIFO for writing. It fails with
// response.GeneralError if ipc is not a FIFO read by a running server.
func connectToEndpoint(ipc string) (*fifo.Fifo, int) {
	f, err := fifo.Open(ipc, fifo.Write)
	if err != nil {
		log.Errorf("failed to connect to endpoint: %v", err)
		return nil, int(response.GeneralError)
	}
	return f, int(response.NoError)
}

// CreateResponseFifo creates the FIFO at path ipc on which the client receives answers.
func CreateResponseFifo(ipc string) (*fifo.Fifo, int) {
	f, err := fifo.Create(ipc, fifo.ReadWrite)
	if err != nil {
		log.Errorf("failed to create response fifo: %v", err)
		return nil, int(response.GeneralError)
	}
	return f, int(response.NoError)
}

// OpenResponseFifo opens the response FIFO at path ipc for reading.
func OpenResponseFifo(ipc string) (*fifo.Fifo, int) {
	f, err := fifo.Open(ipc, fifo.Read)
	if err != nil {
		log.Errorf("failed to open response fifo: %v", err)
		return nil, int(response.GeneralError)
	}
	return f, int(response.NoError)
}

// SendCommand encodes request as JSON and writes it to the server as a cmd packet.
// It returns response.IpcError if the server cannot be reached.
func SendCommand(commandInfo *ClientCommandInfo, cmd uint8, request any) int {
	payload, err := json.Marshal(request)
	if err != nil {
		log.Errorf("failed to encode command %d: %v", cmd, err)
		return int(response.GeneralError)
	}
	if err := packet.WritePacket(commandInfo.serverEndpoint, cmd, payload); err != nil {
		log.Errorf("failed to send command %d to server: %v", cmd, err)
		return int(response.IpcError)
	}
	return int(response.NoError)
}

// ReadCommandResponse is like ReadResponse but also returns the command of the
// packet. A cancelled ctx gives a nil response with response.NoError.
func ReadCommandResponse(ctx context.Context, r *bufio.Reader) (uint8, *response.Response, int) {
	pkt, err := packet.ReadPacket(ctx, r)
	if errors.Is(err, context.Canceled) {
		return commands.CmdNone, nil, int(response.NoError)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		log.Errorf("server did not respond within %s", responseTimeout)
		return commands.CmdNone, nil, int(response.IpcError)
	}
	if err != nil {
		log.Errorf("read response: %v", err)
		return commands.CmdNone, nil, int(response.IpcError)
	}

	var resp response.Response
	if err := json.Unmarshal(pkt.Payload, &resp); err != nil {
		log.Errorf("decode response: %v", err)
		return commands.CmdNone, nil, int(response.IpcError)
	}

	if resp.Code != response.NoError {
		log.Errorf("%s", resp.ErrorMsg)
		return commands.CmdNone, nil, int(resp.Code)
	}
	return pkt.Header.Command, &resp, int(response.NoError)
}

// ReadResponse reads the next response from r and returns it with the exit
// code it implies; the response is nil if the server reported an error.
func ReadResponse(ctx context.Context, r *bufio.Reader) (*response.Response, int) {
	_, resp, errCode := ReadCommandResponse(ctx, r)
	return resp, errCode
}

// GetResponse waits up to responseTimeout for the answer to the last request,
// then closes the response FIFO if stopNewConnectionsAfter is true.
func GetResponse(c ClientCommandInfo, stopNewConnectionsAfter bool) (*response.Response, int) {
	ctx, cancel := context.WithTimeout(c.context, responseTimeout)
	defer cancel()

	resp, code := ReadResponse(ctx, c.responseReader)
	if stopNewConnectionsAfter {
		_ = c.responseFifo.Close()
	}
	return resp, code
}

// dialogWithServer sends request as a cmd packet and returns the answer of the server.
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

// unmarshallResponse decodes the data of resp into a value of type T.
func unmarshallResponse[T any](resp *response.Response) (T, int) {
	var data T
	err := json.Unmarshal(resp.Data, &data)
	if err != nil {
		log.Errorf("failed to unmarshal: %v", err)
		return *new(T), int(response.GeneralError)
	}
	return data, int(response.NoError)
}
