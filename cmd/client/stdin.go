package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	messages "github.com/mpouillo/42-tree-nity/internal/message"
)

type stdinMessage struct {
	msg *messages.Message
	err error
}

// readMessages reads messages from reader in a goroutine and sends them on the
// returned channel, stopping after the first error (io.EOF at the end).
func readMessages(reader io.Reader, raw bool) <-chan stdinMessage {
	stdinChannel := make(chan stdinMessage)

	go func() {
		inputReader := bufio.NewReader(reader)
		scanner := bufio.NewScanner(inputReader)

		for {
			var message stdinMessage
			if raw {
				message.msg, message.err = messages.ReadRaw(inputReader)
			} else {
				message.msg, message.err = readText(scanner)
			}
			stdinChannel <- message
			if message.err != nil {
				return
			}
		}
	}()

	return stdinChannel
}

// readText returns the message of the next key:body line, skipping empty lines.
// A line without a colon is an error, like a partial raw record.
func readText(scanner *bufio.Scanner) (*messages.Message, error) {
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if !bytes.Contains(line, []byte(":")) {
			return nil, fmt.Errorf("invalid message %q (expected key:body)", line)
		}

		msg := messages.FromSeparator(line, ":")
		if len(msg.Key)+len(msg.Body) > messages.MaxSize {
			return nil, messages.ErrTooLarge
		}
		return msg, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}
