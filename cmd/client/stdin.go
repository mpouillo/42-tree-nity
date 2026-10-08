package main

import (
	"bufio"
	"bytes"
	"io"

	"github.com/charmbracelet/log"
	messages "github.com/mpouillo/42-tree-nity/internal/message"
)

type stdinMessage struct {
	msg *messages.Message
	err error
}

//Read in a go routine to not block the main go routine
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


// reads the next line, it skips invalid lines
func readText(scanner *bufio.Scanner) (*messages.Message, error) {
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if !bytes.Contains(line, []byte(":")) {
			log.Errorf("invalid message %q (expected key:body), skipped", line)
			continue
		}

		msg := messages.FromSeparator(line, ":")
		if len(msg.Key)+len(msg.Body) > messages.MaxSize {
			log.Errorf("%v, skipped", messages.ErrTooLarge)
			continue
		}
		return msg, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}
