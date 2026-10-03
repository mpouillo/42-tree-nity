package main

import (
	"regexp"

	commands "github.com/mpouillo/42-tree-nity/internal/ipc/protocol/commands"
	"github.com/charmbracelet/log"
	"github.com/mpouillo/42-tree-nity/internal/ipc/protocol/response"
)

// regexp for valid topic and client names
var validName = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,32}$`)

type CLI struct {
	IPC struct {
		IPC       string               `arg:""`
		Create    commands.CreateTopic `cmd:""`
		List      commands.ListTopics  `cmd:""`
		Info      commands.InfoClient  `cmd:""`
		Produce   commands.Produce     `cmd:""`
		Subscribe commands.Subscribe   `cmd:""`
	} `arg:""`
}

//checks that the names of the topics/clients are valid
func (cli *CLI) validate(cmdName string) int {
	check := func(what, value string) int {
		if !validName.MatchString(value) {
			log.Errorf("invalid %s name %q (expected %s)", what, value, validName)
			return int(response.GeneralError)
		}
		return int(response.NoError)
	}

	switch cmdName {
	case "create":
		return check("topic", cli.IPC.Create.Topic)
	case "info":
		return check("client", cli.IPC.Info.Client)
	case "produce":
		return check("topic", cli.IPC.Produce.Topic)
	case "subscribe":
		if err := check("topic", cli.IPC.Subscribe.Topic); err != int(response.NoError) {
			return err
		}
		return check("client", cli.IPC.Subscribe.Client)
	}
	return int(response.NoError)
}