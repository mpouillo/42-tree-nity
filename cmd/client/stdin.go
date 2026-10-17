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

// Read in a go routine to not block the main go routine
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

// reads the next line, it skips empty lines
func readText(scanner *bufio.Scanner) (*messages.Message, error) {
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// like a partial raw record, a line without separator is a producer error
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
