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
		{name: "list", args: []string{"list"}, want: invocation{action: actionList}},
		{name: "login", args: []string{"login", "personal"}, want: invocation{action: actionLogin, context: "personal", codexArgs: []string{"login"}}},
		{name: "status", args: []string{"status", "personal"}, want: invocation{action: actionStatus, context: "personal", codexArgs: []string{"login", "status"}}},
		{name: "run", args: []string{"run", "work"}, want: invocation{action: actionRun, context: "work"}},
		{name: "run with Codex arguments", args: []string{"run", "work", "--", "-C", "/tmp/project"}, want: invocation{action: actionRun, context: "work", codexArgs: []string{"-C", "/tmp/project"}}},
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
		{"run"},
		{"list", "work"},
		{"run", "../work"},
		{"run", "Work"},
		{"run", "work", "-C", "/tmp/project"},
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
