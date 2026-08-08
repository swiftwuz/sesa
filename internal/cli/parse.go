package cli

import (
	"errors"
	"fmt"

	"sesa/internal/contexts"
)

type action uint8

const (
	actionHelp action = iota
	actionDoctor
	actionList
	actionLogin
	actionStatus
	actionRun
)

type invocation struct {
	action    action
	context   string
	codexArgs []string
}

func parseArgs(args []string) (invocation, error) {
	if len(args) == 0 {
		return invocation{}, errors.New("expected a command")
	}

	switch args[0] {
	case "help", "-h", "--help":
		return parseStandalone(args, "help", actionHelp)
	case "doctor":
		return parseStandalone(args, "doctor", actionDoctor)
	case "list":
		return parseStandalone(args, "list", actionList)
	default:
		return parseContextCommand(args)
	}
}

func parseStandalone(args []string, command string, commandAction action) (invocation, error) {
	if len(args) != 1 {
		return invocation{}, fmt.Errorf("%s does not accept arguments", command)
	}
	return invocation{action: commandAction}, nil
}

func parseContextCommand(args []string) (invocation, error) {
	if len(args) < 2 {
		return invocation{}, errors.New("expected a context")
	}

	command, context := args[0], args[1]
	if err := contexts.ValidateName(context); err != nil {
		return invocation{}, err
	}

	switch command {
	case "login":
		return parseExactContext(args, actionLogin, []string{"login"})
	case "status":
		return parseExactContext(args, actionStatus, []string{"login", "status"})
	case "run":
		return parseRun(args)
	default:
		return invocation{}, fmt.Errorf("unknown command %q", command)
	}
}

func parseExactContext(args []string, commandAction action, codexArgs []string) (invocation, error) {
	if len(args) != 2 {
		return invocation{}, fmt.Errorf("%s does not accept additional arguments", args[0])
	}
	return invocation{action: commandAction, context: args[1], codexArgs: codexArgs}, nil
}

func parseRun(args []string) (invocation, error) {
	if len(args) == 2 {
		return invocation{action: actionRun, context: args[1]}, nil
	}
	if args[2] != "--" {
		return invocation{}, errors.New("put Codex arguments after --")
	}
	return invocation{action: actionRun, context: args[1], codexArgs: args[3:]}, nil
}
