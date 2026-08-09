package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"sesa/internal/contexts"
	"sesa/internal/mappings"
	"sesa/internal/protocol"
	"sesa/internal/repository"
)

func (a App) repositoryCommand(inv invocation, contextStore contexts.Store, mappingStore mappings.Store) int {
	root, err := a.currentRepository()
	if err != nil {
		fmt.Fprintf(a.stderr, "sesa: %v\n", err)
		return 1
	}

	switch inv.action {
	case actionLink:
		return a.linkRepository(root, inv.context, contextStore, mappingStore)
	case actionCurrent:
		return a.showCurrentRepository(root, mappingStore, inv.jsonOutput)
	case actionUnlink:
		return a.unlinkRepository(root, mappingStore)
	default:
		fmt.Fprintln(a.stderr, "sesa: unsupported repository command")
		return 1
	}
}

func (a App) linkRepository(root, context string, contextStore contexts.Store, mappingStore mappings.Store) int {
	exists, err := contextStore.Exists(context)
	if err != nil {
		fmt.Fprintf(a.stderr, "sesa: inspect context %q: %v\n", context, err)
		return 1
	}
	if !exists {
		fmt.Fprintf(a.stderr, "sesa: context %q does not exist; run sesa login %s first\n", context, context)
		return 1
	}
	if err := mappingStore.Set(root, context); err != nil {
		fmt.Fprintf(a.stderr, "sesa: save repository mapping: %v\n", err)
		return 1
	}
	fmt.Fprintf(a.stdout, "Mapped %s to %s.\n", root, strings.ToUpper(context))
	return 0
}

func (a App) showCurrentRepository(root string, mappingStore mappings.Store, jsonOutput bool) int {
	context, ok, err := mappingStore.Get(root)
	if err != nil {
		fmt.Fprintf(a.stderr, "sesa: load repository mapping: %v\n", err)
		return 1
	}
	if jsonOutput {
		return a.writeCurrentJSON(root, context, ok)
	}
	if !ok {
		fmt.Fprintln(a.stderr, "sesa: current repository is not mapped")
		return 1
	}
	fmt.Fprintln(a.stdout, strings.ToUpper(context))
	return 0
}

func (a App) writeCurrentJSON(root, context string, mapped bool) int {
	var selected *string
	if mapped {
		selected = &context
	}
	encoder := json.NewEncoder(a.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(protocol.NewCurrentRepository(root, selected)); err != nil {
		fmt.Fprintf(a.stderr, "sesa: encode JSON output: %v\n", err)
		return 1
	}
	return 0
}

func (a App) unlinkRepository(root string, mappingStore mappings.Store) int {
	removed, err := mappingStore.Remove(root)
	if err != nil {
		fmt.Fprintf(a.stderr, "sesa: remove repository mapping: %v\n", err)
		return 1
	}
	if !removed {
		fmt.Fprintln(a.stderr, "sesa: current repository is not mapped")
		return 1
	}
	fmt.Fprintf(a.stdout, "Removed mapping for %s.\n", root)
	return 0
}

func (a App) selectRunContext(inv invocation, mappingStore mappings.Store) (string, bool, error) {
	root, rootErr := a.currentRepository()
	return a.selectRepositoryContext(inv.context, inv.allowMismatch, root, rootErr, mappingStore)
}

func (a App) selectRepositoryContext(requested string, allowMismatch bool, root string, rootErr error, mappingStore mappings.Store) (string, bool, error) {
	if requested == "" {
		if rootErr != nil {
			return "", false, rootErr
		}
		mapped, ok, err := mappingStore.Get(root)
		if err != nil {
			return "", false, fmt.Errorf("load repository mapping: %w", err)
		}
		if !ok {
			return "", false, errors.New("current repository is not mapped; run sesa link <context>")
		}
		return mapped, true, nil
	}

	if rootErr != nil {
		if errors.Is(rootErr, repository.ErrNotRepository) {
			return requested, false, nil
		}
		return "", false, rootErr
	}
	mapped, ok, err := mappingStore.Get(root)
	if err != nil {
		return "", false, fmt.Errorf("load repository mapping: %w", err)
	}
	if !ok || mapped == requested || allowMismatch {
		return requested, false, nil
	}
	if !a.confirmMismatch(mapped, requested) {
		return "", false, errors.New("context mismatch cancelled")
	}
	return requested, false, nil
}

func (a App) currentRepository() (string, error) {
	directory, err := a.workingDir()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return a.repositoryRoot(directory)
}

func (a App) confirmMismatch(mapped, requested string) bool {
	fmt.Fprintf(a.stderr, "Warning: this repository is mapped to %s, but %s was requested.\n", strings.ToUpper(mapped), strings.ToUpper(requested))
	fmt.Fprintf(a.stderr, "Only continue if your organization permits this code and data to be used with %s.\n", strings.ToUpper(requested))
	fmt.Fprintf(a.stderr, "Continue with %s? [y/N] ", strings.ToUpper(requested))
	answer, err := bufio.NewReader(a.stdin).ReadString('\n')
	if err != nil && len(answer) == 0 {
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}
