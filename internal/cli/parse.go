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
	actionLink
	actionCurrent
	actionUnlink
	actionLogin
	actionStatus
	actionRun
	actionCode
)

type invocation struct {
	action        action
	context       string
	codexArgs     []string
	target        string
	allowMismatch bool
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
	case "current":
		return parseStandalone(args, "current", actionCurrent)
	case "unlink":
		return parseStandalone(args, "unlink", actionUnlink)
	case "run":
		return parseRun(args)
	case "code":
		return parseCode(args)
	default:
		return parseContextCommand(args)
	}
}

func parseCode(args []string) (invocation, error) {
	if len(args) < 2 {
		return invocation{}, errors.New("expected a context")
	}
	if len(args) > 3 {
		return invocation{}, errors.New("code accepts one optional path")
	}
	if err := contexts.ValidateName(args[1]); err != nil {
		return invocation{}, err
	}
	target := "."
	if len(args) == 3 {
		target = args[2]
	}
	return invocation{action: actionCode, context: args[1], target: target}, nil
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
	case "link":
		return parseExactContext(args, actionLink, nil)
	case "login":
		return parseExactContext(args, actionLogin, []string{"login"})
	case "status":
		return parseExactContext(args, actionStatus, []string{"login", "status"})
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
	inv := invocation{action: actionRun}
	if len(args) == 1 {
		return inv, nil
	}

	index := 1
	if args[index] != "--" && args[index] != "--allow-mismatch" {
		if err := contexts.ValidateName(args[index]); err != nil {
			return invocation{}, err
		}
		inv.context = args[index]
		index++
	}
	if index < len(args) && args[index] == "--allow-mismatch" {
		if inv.context == "" {
			return invocation{}, errors.New("--allow-mismatch requires an explicit context")
		}
		inv.allowMismatch = true
		index++
	}
	if index == len(args) {
		return inv, nil
	}
	if args[index] != "--" {
		return invocation{}, errors.New("put Codex arguments after --")
	}
	inv.codexArgs = args[index+1:]
	return inv, nil
}
