package repository

import (
	"errors"
	"testing"
)

func TestGitRootRejectsNonRepository(t *testing.T) {
	_, err := GitRoot(t.TempDir())
	if !errors.Is(err, ErrNotRepository) {
		t.Fatalf("GitRoot() error = %v, want ErrNotRepository", err)
	}
}
