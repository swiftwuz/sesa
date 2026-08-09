package cli

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want invocation
	}{
		{name: "help", args: []string{"help"}, want: invocation{action: actionHelp}},
		{name: "short help", args: []string{"-h"}, want: invocation{action: actionHelp}},
		{name: "long help", args: []string{"--help"}, want: invocation{action: actionHelp}},
		{name: "doctor", args: []string{"doctor"}, want: invocation{action: actionDoctor}},
		{name: "list", args: []string{"list"}, want: invocation{action: actionList}},
		{name: "link", args: []string{"link", "work"}, want: invocation{action: actionLink, context: "work"}},
		{name: "current", args: []string{"current"}, want: invocation{action: actionCurrent}},
		{name: "unlink", args: []string{"unlink"}, want: invocation{action: actionUnlink}},
		{name: "login", args: []string{"login", "personal"}, want: invocation{action: actionLogin, context: "personal", codexArgs: []string{"login"}}},
		{name: "status", args: []string{"status", "personal"}, want: invocation{action: actionStatus, context: "personal", codexArgs: []string{"login", "status"}}},
		{name: "run", args: []string{"run", "work"}, want: invocation{action: actionRun, context: "work"}},
		{name: "mapped run", args: []string{"run"}, want: invocation{action: actionRun}},
		{name: "mapped run with Codex arguments", args: []string{"run", "--", "-C", "/tmp/project"}, want: invocation{action: actionRun, codexArgs: []string{"-C", "/tmp/project"}}},
		{name: "run with mismatch override", args: []string{"run", "personal", "--allow-mismatch"}, want: invocation{action: actionRun, context: "personal", allowMismatch: true}},
		{name: "run with Codex arguments", args: []string{"run", "work", "--", "-C", "/tmp/project"}, want: invocation{action: actionRun, context: "work", codexArgs: []string{"-C", "/tmp/project"}}},
		{name: "code default path", args: []string{"code", "personal"}, want: invocation{action: actionCode, context: "personal", target: "."}},
		{name: "code path", args: []string{"code", "work", "/tmp/project"}, want: invocation{action: actionCode, context: "work", target: "/tmp/project"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if err != nil {
				t.Fatalf("parseArgs() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseArgsRejectsUnsafeOrAmbiguousInput(t *testing.T) {
	tests := [][]string{
		nil,
		{"help", "extra"},
		{"doctor", "extra"},
		{"list", "work"},
		{"current", "extra"},
		{"unlink", "extra"},
		{"run", "../work"},
		{"run", "Work"},
		{"run", "work", "-C", "/tmp/project"},
		{"run", "--allow-mismatch"},
		{"code"},
		{"code", "Work"},
		{"code", "work", ".", "extra"},
		{"login", "work", "extra"},
		{"status", "work", "extra"},
		{"unknown", "work"},
	}

	for _, args := range tests {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%q) unexpectedly succeeded", args)
		}
	}
}
